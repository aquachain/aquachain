package subcommands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/common/fdlimit"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/common/toml"
	"gitlab.com/aquachain/aquachain/consensus"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/state"
	"gitlab.com/aquachain/aquachain/core/vm"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/aquastats"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/p2p/discover"
	"gitlab.com/aquachain/aquachain/p2p/nat"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
)

const CommandHelpTemplate = `{{.cmd.Name}}{{if .cmd.Subcommands}} command{{end}}{{if .cmd.Flags}} [command options]{{end}} [arguments...]
{{if .cmd.Description}}{{.cmd.Description}}
{{end}}{{if .cmd.Subcommands}}
SUBCOMMANDS:
	{{range .cmd.Subcommands}}{{.cmd.Name}}{{with .cmd.ShortName}}, {{.cmd}}{{end}}{{ "\t" }}{{.cmd.Usage}}
	{{end}}{{end}}{{if .categorizedFlags}}
{{range $idx, $categorized := .categorizedFlags}}{{$categorized.Name}} OPTIONS:
{{range $categorized.Flags}}{{"\t"}}{{.}}
{{end}}
{{end}}{{end}}`

const appHelpTemplate = `{{.Name}} {{if .Flags}}[global options] {{end}}command{{if .Flags}} [command options]{{end}} [arguments...]

VERSION:
   {{.Version}}

COMMANDS:
   {{range .Commands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
   {{end}}{{if .Flags}}
GLOBAL OPTIONS:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

// NewApp creates an app with sane defaults.
func NewApp(name, gitCommit, usage string) *cli.Command {
	if name == "" {
		name = filepath.Base(os.Args[0])
	}
	app := &cli.Command{
		Name:                       name,
		Usage:                      usage,
		Version:                    params.VersionWithCommit(gitCommit),
		ReadArgsFromStdin:          false,
		EnableShellCompletion:      true,
		Suggest:                    true,
		ShellCompletionCommandName: "generate-shell-completion",
		ShellComplete: func(ctx context.Context, cmd *cli.Command) {
			// This will complete if no args are passed
			if cmd.NArg() > 0 {
				fmt.Println("complete")
				return
			}
			for _, c := range cmd.VisibleCommands() {
				fmt.Println(c.Name)
			}
		},
	}
	return app
}

// MakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MakeDataDir(cmd *cli.Command) string {
	if datadir := cmd.String(aquaflags.DataDirFlag.Name); datadir != "" {
		return datadir
	}
	chainName := cmd.String(aquaflags.ChainFlag.Name)
	if chainName == "" {
		Fatalf("No chain selected, no data directory specified")
	}
	if chainName == params.MainnetChainConfig.Name() {
		return node.DefaultDatadir() // skip subdirectory for mainnet
	}
	return filepath.Join(node.DefaultDatadir(), chainName)
}

// 	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
// 	return ""
// }

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func setNodeKey(cmd *cli.Command, cfg *p2p.Config) {
	var (
		hex  = cmd.String(aquaflags.NodeKeyHexFlag.Name)
		file = cmd.String(aquaflags.NodeKeyFileFlag.Name)
		key  *btcec.PrivateKey
		err  error
	)
	switch {
	case file != "" && hex != "":
		Fatalf("Options %q and %q are mutually exclusive", aquaflags.NodeKeyFileFlag.Name, aquaflags.NodeKeyHexFlag.Name)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			Fatalf("Option %q: %v", aquaflags.NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	case hex != "":
		if key, err = crypto.HexToBtcec(hex); err != nil {
			Fatalf("Option %q: %v", aquaflags.NodeKeyHexFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
}

// setNodeUserIdent creates the user identifier from CLI flags.
func setNodeUserIdent(cmd *cli.Command, cfg *node.Config) {
	if identity := cmd.String(aquaflags.IdentityFlag.Name); len(identity) > 0 {
		cfg.UserIdent = identity
	}
}

// returns chainname, chaincfg, bootnodes, datadir
func getStuff(cmd *cli.Command) (string, *params.ChainConfig, []*discover.Node, DirectoryConfig) {
	chainName := cmd.String(aquaflags.ChainFlag.Name)
	log.Warn("chainName", "chainName", chainName)
	if chainName == "" {
		Fatalf("No chain selected")
		panic("no chain name")
	}
	chaincfg := params.GetChainConfig(chainName)
	if chaincfg == nil {
		// check directory
		expected := filepath.Join(node.DefaultDatadir(), chainName)
		stat, err := os.Stat(expected)
		if err == nil && stat.IsDir() {
			chaincfg, err = params.LoadChainConfigFile(expected)
			if err != nil {
				Fatalf("Failed to load custom chain config: %v", err)
			}
		}
	}
	if chaincfg == nil {
		Fatalf("invalid chain name: %q", cmd.String(aquaflags.ChainFlag.Name))
		panic("bad chain name")
	}
	switch chainName { // TODO: remove once disabled chain-flags are cleaned
	case "dev":
		cmd.Set(aquaflags.DeveloperFlag.Name, "true")
	case "testnet":
		cmd.Set(aquaflags.TestnetFlag.Name, "true")
	case "testnet2":
		cmd.Set(aquaflags.Testnet2Flag.Name, "true")
	case "testnet3":
		cmd.Set(aquaflags.Testnet3Flag.Name, "true")
	}
	return chainName, chaincfg, getBootstrapNodes(cmd), switchDatadir(cmd, chaincfg)
}

func getBootstrapNodes(cmd *cli.Command) []*discover.Node {
	if cmd.IsSet(aquaflags.BootnodesFlag.Name) { // custom bootnodes flag
		return StringToBootstraps(strings.Split(cmd.String(aquaflags.BootnodesFlag.Name), ","))
	}
	if cmd.IsSet(aquaflags.NoDiscoverFlag.Name) {
		return []*discover.Node{}
	}
	chainName := cmd.String(aquaflags.ChainFlag.Name)
	if chainName == "" {
		panic("woops") // should be already set
	}
	var urls []string
	switch chainName {
	case "aqua":
		urls = params.MainnetBootnodes
	case "testnet":
		urls = params.TestnetBootnodes
	case "testnet2":
		urls = params.Testnet2Bootnodes
	case "testnet3":
		urls = params.Testnet3Bootnodes
	default: // no bootnodes (testnet3, etc)
		return []*discover.Node{} // non-nil but empty
	}
	return StringToBootstraps(urls)

}

func StringToBootstraps(ss []string) []*discover.Node {
	var nodes []*discover.Node
	for _, url := range ss {
		url = strings.TrimSpace(url)
		node, err := discover.ParseNode(url)
		if err != nil {
			log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
		}
		nodes = append(nodes, node)
	}
	return nodes
}

// // setBootstrapNodes creates a list of bootstrap nodes from the command line
// // flags, reverting to pre-configured ones if none have been specified.
// func setBootstrapNodes(cmd *cli.Command, cfg *p2p.Config) {
// 	if cfg.BootstrapNodes != nil {
// 		return // already set, don't apply defaults.
// 	}
// 	var urls []string
// 	if cmd.IsSet(aquaflags.ChainFlag.Name) {
// 		chainName := cmd.String(aquaflags.ChainFlag.Name)
// 		switch chainName {
// 		default: // no bootnodes
// 		case "aqua":
// 			urls = params.MainnetBootnodes
// 		case "testnet":
// 			urls = params.TestnetBootnodes
// 		case "testnet2":
// 			urls = params.Testnet2Bootnodes
// 		}
// 	} else {

// 	}
// 	nodes := make([]*discover.Node, 0, len(urls))
// 	for _, url := range urls {
// 		node, err := discover.ParseNode(url)
// 		if err != nil {
// 			log.Crit("Bootstrap URL invalid", "enode", url, "err", err)
// 		}
// 		nodes = append(nodes, node)
// 	}
// 	cfg.BootstrapNodes = nodes
// }
/*
// user might have set "-testnet" or "-chain testnet" flags
// this should be set before any other flags are processed
func setChainConfig(cmd *cli.Command) *params.ChainConfig {
	chainName := cmd.String(aquaflags.ChainFlag.Name)
	if chainName == "" {
		switch {
		case cmd.Bool(aquaflags.TestnetFlag.Name):
			chainName = "testnet"
		case cmd.Bool(Testnet2Flag.Name):
			chainName = "testnet2"
		default:
			chainName = "aqua"
		}
	}
	chaincfg := params.GetChainConfig(chainName)
	if chaincfg == nil {
		Fatalf("invalid chain name: %q", cmd.String(aquaflags.ChainFlag.Name))
	}
	return chaincfg
}*/

// setListenAddress creates a TCP listening address string from set command
// line flags.
func getListenAddress(cmd *cli.Command) string {
	var listenaddr string
	chainName := cmd.String(aquaflags.ChainFlag.Name)
	if chainName == "" {
		panic("no chain name")
	}
	chainCfg := params.GetChainConfig(chainName)
	if chainCfg != nil && chainCfg.DefaultPortNumber != 0 {
		listenaddr = fmt.Sprintf("%s:%d", "0.0.0.0", chainCfg.DefaultPortNumber)
	}

	// flag overrides
	if cmd.IsSet(aquaflags.ListenAddrFlag.Name) && cmd.IsSet(aquaflags.ListenPortFlag.Name) {
		listenaddr = fmt.Sprintf("%s:%d", cmd.String(aquaflags.ListenAddrFlag.Name), cmd.Int(aquaflags.ListenPortFlag.Name))
	} else if !cmd.IsSet(aquaflags.ListenAddrFlag.Name) && cmd.IsSet(aquaflags.ListenPortFlag.Name) {
		listenaddr = fmt.Sprintf(":%d", cmd.Int(aquaflags.ListenPortFlag.Name))
	} else if cmd.IsSet(aquaflags.ListenAddrFlag.Name) {
		listenaddr = cmd.String(aquaflags.ListenAddrFlag.Name)
	}

	if listenaddr == "" {
		Fatalf("No listen address specified")
	}

	return listenaddr
}

// setNAT creates a port mapper from command line flags.
func setNAT(cmd *cli.Command, cfg *p2p.Config) {
	if cmd.IsSet(aquaflags.NATFlag.Name) {
		_, err := nat.Parse(cmd.String(aquaflags.NATFlag.Name))
		if err != nil {
			Fatalf("Option %s: %v", aquaflags.NATFlag.Name, err)
		}
		cfg.NAT = nat.NatString(cmd.String(aquaflags.NATFlag.Name))
	}
	if cmd.IsSet(aquaflags.OfflineFlag.Name) {
		cfg.NAT = ""
	}
}

// splitAndTrim splits input separated by a comma
// and trims excessive white space from the substrings.
func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// setHTTP creates the HTTP RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setHTTP(cmd *cli.Command, cfg *node.Config) {
	var nokeys = cmd.Bool(aquaflags.NoKeysFlag.Name) || cmd.Bool(aquaflags.NoSignFlag.Name)
	if nokeys {
		cfg.NoKeys = true
	}
	if cmd.IsSet(aquaflags.RPCListenAddrFlag.Name) && cmd.IsSet(aquaflags.UnlockedAccountFlag.Name) && !cmd.IsSet(aquaflags.RPCUnlockFlag.Name) {
		Fatalf("Woah there! By default, using -rpc and -unlock is \"safe\", (localhost).\n" +
			"But you shouldn't use --rpcaddr with --unlock flag.\n" +
			"If you really know what you are doing and would like to unlock a wallet while" +
			"hosting a public HTTP RPC node, use the -UNSAFE_RPC_UNLOCK flag. See -allowip flag to restrict access")
		os.Exit(1)
	}
	if cmd.Bool(aquaflags.RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if cmd.IsSet(aquaflags.RPCListenAddrFlag.Name) && cmd.IsSet(aquaflags.UnlockedAccountFlag.Name) {
			// allow public rpc with unlocked account, exposed only via 'private' api namespace (aqua.sendTransaction and aqua.sign are disabled)
			keystore.SetNoSignMode()
		}
		if cmd.IsSet(aquaflags.RPCListenAddrFlag.Name) {
			cfg.HTTPHost = cmd.String(aquaflags.RPCListenAddrFlag.Name)
		}
	}

	if cmd.IsSet(aquaflags.RPCPortFlag.Name) {
		cfg.HTTPPort = int(cmd.Int(aquaflags.RPCPortFlag.Name))
	}
	if cmd.IsSet(aquaflags.RPCCORSDomainFlag.Name) {
		cfg.HTTPCors = splitAndTrim(cmd.String(aquaflags.RPCCORSDomainFlag.Name))
	}
	if cmd.IsSet(aquaflags.RPCApiFlag.Name) {
		cfg.HTTPModules = parseRpcFlags(cfg.HTTPModules, splitAndTrim(cmd.String(aquaflags.RPCApiFlag.Name)))
	}

	cfg.HTTPVirtualHosts = splitAndTrim(cmd.String(aquaflags.RPCVirtualHostsFlag.Name))
	cfg.RPCAllowIP = splitAndTrim(cmd.String(aquaflags.RPCAllowIPFlag.Name))
}

// allow '+' prefixed flags to append to the default modules
// eg: --rpcapi +testing  (adds 'testing' to the default modules)
func parseRpcFlags(defaultModules, maybe []string) []string {
	if len(defaultModules) == 0 {
		defaultModules = node.DefaultConfig.HTTPModules
	}
	if len(defaultModules) == 0 {
		Fatalf("No default modules set")
	}
	if len(maybe) == 0 {
		return defaultModules
	}
	shouldAppend := false
	for i := range maybe {
		if maybe[i][0] == '+' {
			maybe[i] = maybe[i][1:]
			shouldAppend = true
		}
	}
	if shouldAppend {
		return append(defaultModules, maybe...)
	}
	return maybe
}

// setWS creates the WebSocket RPC listener interface string from the set
// command line flags, returning empty if the HTTP endpoint is disabled.
func setWS(cmd *cli.Command, cfg *node.Config) {
	if cmd.Bool(aquaflags.WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if cmd.IsSet(aquaflags.WSListenAddrFlag.Name) {
			cfg.WSHost = cmd.String(aquaflags.WSListenAddrFlag.Name)
		}
		log.Info("Websocket enabled!", "wshost", cfg.WSHost)
	}

	if cmd.IsSet(aquaflags.WSPortFlag.Name) {
		cfg.WSPort = int(cmd.Int(aquaflags.WSPortFlag.Name))
	}
	if cmd.IsSet(aquaflags.WSAllowedOriginsFlag.Name) {
		cfg.WSOrigins = splitAndTrim(cmd.String(aquaflags.WSAllowedOriginsFlag.Name))
	}
	if cmd.IsSet(aquaflags.WSApiFlag.Name) {
		cfg.WSModules = splitAndTrim(cmd.String(aquaflags.WSApiFlag.Name))
	}
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func setIPC(cmd *cli.Command, cfg *node.Config) {
	checkExclusive(cmd, aquaflags.IPCDisabledFlag, aquaflags.IPCPathFlag)
	switch {
	case cmd.Bool(aquaflags.IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case cmd.IsSet(aquaflags.IPCPathFlag.Name):
		cfg.IPCPath = cmd.String(aquaflags.IPCPathFlag.Name)
	}
}

// makeDatabaseHandles raises out the number of allowed file handles per process
// for Aquachain and returns half of the allowance to assign to the database.
func makeDatabaseHandles() int {
	limit, err := fdlimit.Current()
	if err != nil {
		Fatalf("Failed to retrieve file descriptor allowance: %v", err)
	}
	if limit < 2048 {
		if err := fdlimit.Raise(2048); err != nil {
			Fatalf("Failed to raise file descriptor allowance: %v", err)
		}
	}
	if limit > 2048 { // cap database file descriptors even if more is available
		limit = 2048
	}
	return limit / 2 // Leave half for networking and other stuff
}

// MakeAddress converts an account specified directly as a hex encoded string or
// a key index in the key store to an internal account representation.
func MakeAddress(ks *keystore.KeyStore, account string) (accounts.Account, error) {
	// If the specified account is a valid address, return it
	if common.IsHexAddress(account) {
		return accounts.Account{Address: common.HexToAddress(account)}, nil
	}
	// Otherwise try to interpret the account as a keystore index
	index, err := strconv.Atoi(account)
	if err != nil || index < 0 {
		return accounts.Account{}, fmt.Errorf("invalid account address or index %q", account)
	}
	log.Warn("-------------------------------------------------------------------")
	log.Warn("Referring to accounts by order in the keystore folder is dangerous!")
	log.Warn("This functionality is deprecated and will be removed in the future!")
	log.Warn("Please use explicit addresses! (can search via `aquachain account list`)")
	log.Warn("-------------------------------------------------------------------")

	accs := ks.Accounts()
	if len(accs) <= index {
		return accounts.Account{}, fmt.Errorf("index %d higher than number of accounts %d", index, len(accs))
	}
	return accs[index], nil
}

// setAquabase retrieves the aquabase either from the directly specified
// command line flags or from the keystore if CLI indexed.
func setAquabase(cmd *cli.Command, ks *keystore.KeyStore, cfg *aqua.Config) {
	if cmd.IsSet(aquaflags.AquabaseFlag.Name) {
		account, err := MakeAddress(ks, cmd.String(aquaflags.AquabaseFlag.Name))
		if err != nil {
			Fatalf("Option %q: %v", aquaflags.AquabaseFlag.Name, err)
		}
		cfg.Aquabase = account.Address
	}
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
//
// If using a password file, the file should contain one password per line.
// Using env example: `aquachain --password '$PASSWORD'` (use single quotes to prevent shell expansion)
//
// eg: -unlock 0,1 -password ',$PASSWORD'  (use empty pw for first, use env var for second)
func MakePasswordList(cmd *cli.Command) []string {
	path := strings.TrimSpace(cmd.String(aquaflags.PasswordFileFlag.Name))
	if path == "" {
		return nil
	}
	// only treat unexpanded env vars as passwords. otherwise, read the file.
	if path[0] == '$' || strings.Contains(path, ",$") {
		return strings.Split(os.ExpandEnv(path), ",") // skip file read, use env pw
	}
	text, err := os.ReadFile(path)
	if err != nil {
		Fatalf("Failed to read password file: %v", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}

func SetP2PConfig(cmd *cli.Command, cfg *p2p.Config) {
	// cant be zero
	if cfg.ChainId == 0 {
		panic("P2P config has no chain ID")
	}

	setNodeKey(cmd, cfg)
	setNAT(cmd, cfg)
	cfg.ListenAddr = getListenAddress(cmd)
	if cmd.IsSet(aquaflags.MaxPeersFlag.Name) {
		cfg.MaxPeers = int(cmd.Int(aquaflags.MaxPeersFlag.Name))
	}

	if cmd.IsSet(aquaflags.MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = int(cmd.Int(aquaflags.MaxPendingPeersFlag.Name))
	}

	if cmd.IsSet(aquaflags.OfflineFlag.Name) {
		log.Info("Offline mode enabled")
		cfg.NoDiscovery = true
		cfg.Offline = true
		cfg.NAT = ""
		cfg.BootstrapNodes = nil
		// also set aqua.SyncMode ...
	} else {
		log.Info("Listen Address:", "tcp+udp", cfg.ListenAddr)
		log.Debug("Maximum peer count", "AQUA", cfg.MaxPeers)
	}

	if cmd.IsSet(aquaflags.NoDiscoverFlag.Name) {
		cfg.NoDiscovery = true
	}
	if cmd.Bool(aquaflags.Testnet2Flag.Name) {
		cfg.NoDiscovery = true
	}
	if netrestrict := cmd.String(aquaflags.NetrestrictFlag.Name); netrestrict != "" {
		list, err := netutil.ParseNetlist(netrestrict)
		if err != nil {
			Fatalf("Option %q: %v", aquaflags.NetrestrictFlag.Name, err)
		}
		cfg.NetRestrict = list
	}

	if cmd.Bool(aquaflags.DeveloperFlag.Name) {
		// --dev mode can't use p2p networking.
		cfg.MaxPeers = 0
		cfg.ListenAddr = "127.0.0.1:0" // random local port
		cfg.NoDiscovery = true
	}

	/*
		switch cmd.String(aquaflags.ChainFlag.Name) {
		case "aqua":
			// cfg.ListenAddr = ":21303"
		case "testnet":
			// cfg.ListenAddr = ":21304"
		case "testnet2":
			// cfg.MaxPeers = 0
			// cfg.ListenAddr = "127.0.0.1:0"
			// cfg.NoDiscovery = true
			// cfg.Offline = true
		case "testnet3":
			cfg.MaxPeers = 0
			cfg.ListenAddr = "127.0.0.1:0"
			cfg.NoDiscovery = true
			cfg.Offline = true
		}
		// if cfg.ListenAddr == "" && cmd.Bool(aquaflags.TestnetFlag.Name) && !cmd.IsSet(aquaflags.ListenPortFlag.Name) {
		// 	cfg.ListenAddr = "0.0.0.0:21304"
		// }
		// if cfg.ListenAddr == "" && cmd.Bool(aquaflags.NetworkEthFlag.Name) && !cmd.IsSet(aquaflags.ListenPortFlag.Name) {
		// 	cfg.ListenAddr = ":30303"
		// }
	*/

}

type DirectoryConfig struct {
	KeyStoreDir string
	DataDir     string
}

// switchDatadir switches the data directory based on the chain name.
// override with --datadir
func switchDatadir(cmd *cli.Command, chaincfg *params.ChainConfig) DirectoryConfig {
	if chaincfg == nil {
		panic("switchDatadir: no chain config")
	}
	if chaincfg == nil && !cmd.IsSet(aquaflags.DataDirFlag.Name) {
		Fatalf("No chain and no data directory specified. Please specify a chain with --chain or a data directory with --datadir")
		return DirectoryConfig{}
	}
	chainName := "(unknown)"
	if chaincfg != nil {
		chainName = chaincfg.Name()
		if chainName == "" {
			panic("chain config has no name, use params.AddChainConfig or add a chain to params package")
		}
	}
	var cfg DirectoryConfig

	// custom datadir
	if cmd.IsSet(aquaflags.DataDirFlag.Name) {
		cfg.DataDir = cmd.String(aquaflags.DataDirFlag.Name)
	} else {
		cfg.DataDir = node.DefaultDatadirByChain(chaincfg)
	}
	if cfg.DataDir == "" {
		Fatalf("Cannot determine default data directory, please set manually (--datadir)")
		return cfg
	}
	// custom keystore
	switch {
	case sense.IsNoKeys():
		cfg.KeyStoreDir = ""
	case cmd.IsSet(aquaflags.KeyStoreDirFlag.Name):
		cfg.KeyStoreDir = cmd.String(aquaflags.KeyStoreDirFlag.Name)
		log.Info("set custom keystore", "path", cfg.KeyStoreDir)
	default:
		cfg.KeyStoreDir = filepath.Join(cfg.DataDir, "keystore")
	}
	log.Info("assumed datadir", "path", cfg.DataDir, "chain", chainName)
	return cfg

	// switch {
	// case cmd.IsSet(aquaflags.ChainFlag.Name):
	// 	chainName := cmd.String(aquaflags.ChainFlag.Name)
	// 	chaincfg := params.GetChainConfig(chainName)
	// 	if chaincfg == nil {
	// 		return errors.New("invalid config name")
	// 	}
	// 	if chaincfg == params.MainnetChainConfig {
	// 		newdatadir = node.DefaultDatadir()
	// 	} else {
	// 		newdatadir = filepath.Join(node.DefaultDatadir(), "chains", chainName)
	// 	}
	// 	cfg.P2P.ChainId = chaincfg.ChainId.Uint64()
	// case cmd.IsSet(aquaflags.NetworkIdFlag.Name):
	// 	cfg.P2P.ChainId = cmd.Uint(aquaflags.NetworkIdFlag.Name)
	// 	newdatadir = filepath.Join(node.DefaultDatadir(), fmt.Sprintf("chainid-%v", cfg.P2P.ChainId))
	// case cmd.Bool(aquaflags.DeveloperFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultDatadir(), "develop")
	// 	cfg.P2P.ChainId = 1337
	// case cmd.Bool(aquaflags.TestnetFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultDatadir(), "testnet")
	// 	cfg.P2P.ChainId = params.TestnetChainConfig.ChainId.Uint64()
	// case cmd.Bool(Testnet2Flag.Name):
	// 	newdatadir = filepath.Join(node.DefaultDatadir(), "testnet2")
	// 	cfg.P2P.ChainId = params.Testnet2ChainConfig.ChainId.Uint64()
	// case cmd.Bool(aquaflags.NetworkEthFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultDatadir(), "ethereum")
	// 	cfg.P2P.ChainId = params.EthnetChainConfig.ChainId.Uint64()
	// default:
	// 	// mainnet
	// 	cfg.P2P.ChainId = params.MainnetChainConfig.ChainId.Uint64()
	// 	newdatadir = node.DefaultDatadir()
	// }

	// if cmd.IsSet(aquaflags.KeyStoreDirFlag.Name) {
	// 	cfg.KeyStoreDir = cmd.String(aquaflags.KeyStoreDirFlag.Name)
	// 	if cfg.KeyStoreDir == "" {
	// 		cfg.NoKeys = true
	// 	}
	// }
	// if cmd.IsSet(aquaflags.DataDirFlag.Name) {
	// 	newdatadir = cmd.String(aquaflags.DataDirFlag.Name)
	// }
	// return newdatadir

}

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(cmd *cli.Command, cfg *node.Config) error {

	// setBootstrapNodes(ctx, cfg)
	var (
		chaincfg       *params.ChainConfig
		chainName      string
		bootstrapNodes []*discover.Node
		directoryCfg   DirectoryConfig
	)

	chainName, chaincfg, bootstrapNodes, directoryCfg = getStuff(cmd)
	log.Info("Loading...",
		"chain", chainName,
		"chainID", chaincfg.ChainId,
		"datadir", directoryCfg.DataDir,
		"keystore", directoryCfg.KeyStoreDir,
		"bootnodes", len(bootstrapNodes),
		"rpc", cfg.HTTPHost,
		"ws", cfg.WSHost,
		"p2p", cfg.P2P.ListenAddr,
		"ipc", cfg.IPCPath,
		"nokeys", cfg.NoKeys,
	)
	cfg.DataDir = directoryCfg.DataDir
	cfg.KeyStoreDir = directoryCfg.KeyStoreDir
	if cfg.KeyStoreDir == "" {
		cfg.NoKeys = true
		os.Setenv("NO_KEYS", "true") // might be too late
		log.Warn("keystore set to empty string, disabling keys")
	}
	cfg.P2P.ChainId = chaincfg.ChainId.Uint64()
	cfg.P2P.BootstrapNodes = bootstrapNodes

	SetP2PConfig(cmd, cfg.P2P)
	setIPC(cmd, cfg)
	setHTTP(cmd, cfg)
	setWS(cmd, cfg)
	setNodeUserIdent(cmd, cfg)
	if cmd.IsSet(aquaflags.NoKeysFlag.Name) {
		cfg.NoKeys = cmd.Bool(aquaflags.NoKeysFlag.Name)
		log.Info("no keys mode", "enabled", cfg.NoKeys)
	}
	if cfg.NoKeys {
		log.Info("No-Keys mode")
	}
	if cmd.IsSet(aquaflags.UseUSBFlag.Name) {
		cfg.UseUSB = cmd.Bool(aquaflags.UseUSBFlag.Name)
	}
	if cmd.IsSet(aquaflags.RPCBehindProxyFlag.Name) || sense.EnvBool("RPC_BEHIND_PROXY") {
		cfg.RPCBehindProxy = cmd.Bool(aquaflags.RPCBehindProxyFlag.Name)
	}
	return nil
}

func setGPO(cmd *cli.Command, cfg *gasprice.Config) {
	if cmd.IsSet(aquaflags.GpoBlocksFlag.Name) {
		cfg.Blocks = int(cmd.Int(aquaflags.GpoBlocksFlag.Name))
	}
	if cmd.IsSet(aquaflags.GpoPercentileFlag.Name) {
		cfg.Percentile = int(cmd.Int(aquaflags.GpoPercentileFlag.Name))
	}
}

func setTxPool(cmd *cli.Command, cfg *core.TxPoolConfig) {
	if cmd.IsSet(aquaflags.TxPoolNoLocalsFlag.Name) {
		cfg.NoLocals = cmd.Bool(aquaflags.TxPoolNoLocalsFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolJournalFlag.Name) {
		cfg.Journal = cmd.String(aquaflags.TxPoolJournalFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolRejournalFlag.Name) {
		cfg.Rejournal = cmd.Duration(aquaflags.TxPoolRejournalFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolPriceLimitFlag.Name) {
		cfg.PriceLimit = cmd.Uint(aquaflags.TxPoolPriceLimitFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolPriceBumpFlag.Name) {
		cfg.PriceBump = cmd.Uint(aquaflags.TxPoolPriceBumpFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolAccountSlotsFlag.Name) {
		cfg.AccountSlots = cmd.Uint(aquaflags.TxPoolAccountSlotsFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolGlobalSlotsFlag.Name) {
		cfg.GlobalSlots = cmd.Uint(aquaflags.TxPoolGlobalSlotsFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolAccountQueueFlag.Name) {
		cfg.AccountQueue = cmd.Uint(aquaflags.TxPoolAccountQueueFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolGlobalQueueFlag.Name) {
		cfg.GlobalQueue = cmd.Uint(aquaflags.TxPoolGlobalQueueFlag.Name)
	}
	if cmd.IsSet(aquaflags.TxPoolLifetimeFlag.Name) {
		cfg.Lifetime = cmd.Duration(aquaflags.TxPoolLifetimeFlag.Name)
	}
}

func setAquahash(cmd *cli.Command, cfg *aqua.Config) {
	if cmd.IsSet(aquaflags.AquahashCacheDirFlag.Name) {
		cfg.Aquahash.CacheDir = cmd.String(aquaflags.AquahashCacheDirFlag.Name)
	}
	if cmd.IsSet(aquaflags.AquahashDatasetDirFlag.Name) {
		cfg.Aquahash.DatasetDir = cmd.String(aquaflags.AquahashDatasetDirFlag.Name)
	}
	if cmd.IsSet(aquaflags.AquahashCachesInMemoryFlag.Name) {
		cfg.Aquahash.CachesInMem = int(cmd.Int(aquaflags.AquahashCachesInMemoryFlag.Name))
	}
	if cmd.IsSet(aquaflags.AquahashCachesOnDiskFlag.Name) {
		cfg.Aquahash.CachesOnDisk = int(cmd.Int(aquaflags.AquahashCachesOnDiskFlag.Name))
	}
	if cmd.IsSet(aquaflags.AquahashDatasetsInMemoryFlag.Name) {
		cfg.Aquahash.DatasetsInMem = int(cmd.Int(aquaflags.AquahashDatasetsInMemoryFlag.Name))
	}
	if cmd.IsSet(aquaflags.AquahashDatasetsOnDiskFlag.Name) {
		cfg.Aquahash.DatasetsOnDisk = int(cmd.Int(aquaflags.AquahashDatasetsOnDiskFlag.Name))
	}
}

// checkExclusive verifies that only a single isntance of the provided flags was
// set by the user. Each flag might optionally be followed by a string type to
// specialize it further.
func checkExclusive(cmd *cli.Command, args ...cli.Flag) {
	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		flag := args[i]
		// Check if next arg extends current and expand its name if so
		names := flag.Names()
		name := names[0]
		option := names[0]
		if i+1 < len(args) {
			switch args[i+1].(type) {
			case *cli.StringFlag:
				// Extended flag, expand the name and shift the arguments
				if cmd.String(name) == option {
					name += "=" + option
				}
				i++

			case cli.Flag:
			default:
				panic(fmt.Sprintf("invalid argument, not cli.aquaflags.Flag or string extension: %T", args[i+1]))
			}
		}
		// Mark the flag if it's set
		if cmd.IsSet(name) {
			set = append(set, "--"+name)
		}
	}
	if len(set) > 1 {
		Fatalf("Flags %v can't be used at the same time", strings.Join(set, ", "))
	}
}

func setHardforkFlagParams(cmd *cli.Command, chaincfg *params.ChainConfig) {
	// activate HF8 at block number X (not activated by default)
	if cmd.IsSet(aquaflags.HF8MainnetFlag.Name) && chaincfg.HF[8] == nil {
		chaincfg.HF[8] = big.NewInt(0).SetUint64(uint64(cmd.Uint(aquaflags.HF8MainnetFlag.Name)))
	}
}

// SetAquaConfig applies aqua-related command line flags to the config.
func SetAquaConfig(cmd *cli.Command, stack *node.Node, cfg *aqua.Config) {
	// Avoid conflicting network flags
	// note: these are disabled flags, but Set is still called before this function
	checkExclusive(cmd, aquaflags.DeveloperFlag, aquaflags.TestnetFlag, aquaflags.Testnet2Flag, aquaflags.NetworkEthFlag)
	checkExclusive(cmd, aquaflags.DeveloperFlag, aquaflags.NoKeysFlag)
	checkExclusive(cmd, aquaflags.FastSyncFlag, aquaflags.SyncModeFlag, aquaflags.OfflineFlag)
	if cmd.Bool(aquaflags.AlertModeFlag.Name) {
		cfgAlerts, err := alerts.ParseAlertConfig()
		if err != nil {
			Fatalf("Failed to parse alert config: %v", err)
		}
		log.Info("alert config", "platform", cfgAlerts.Platform, "channel", cfgAlerts.Channel)
	}
	chaincfg := SetChainId(cmd, cfg)
	setHardforkFlagParams(cmd, chaincfg) // modify HF map

	// get aquabase if exists in keystore if that exists (-nokeys)
	am := stack.AccountManager()
	if am != nil {
		ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		setAquabase(cmd, ks, cfg)
	}

	setGPO(cmd, &cfg.GPO)
	setTxPool(cmd, &cfg.TxPool)
	setAquahash(cmd, cfg)

	switch {
	default:
		cfg.SyncMode = downloader.FullSync
	case cmd.Bool(aquaflags.OfflineFlag.Name):
		cfg.SyncMode = downloader.OfflineSync
	case cmd.IsSet(aquaflags.SyncModeFlag.Name):
		err := cfg.SyncMode.UnmarshalText([]byte(cmd.String(aquaflags.SyncModeFlag.Name)))
		if err != nil {
			Fatalf("Failed to parse sync mode: %v", err)
		}
	case cmd.Bool(aquaflags.FastSyncFlag.Name):
		cfg.SyncMode = downloader.FastSync
	}

	if cmd.IsSet(aquaflags.CacheFlag.Name) || cmd.IsSet(aquaflags.CacheDatabaseFlag.Name) {
		cfg.DatabaseCache = int(cmd.Int(aquaflags.CacheFlag.Name) * cmd.Int(aquaflags.CacheDatabaseFlag.Name) / 100)
	}
	cfg.DatabaseHandles = makeDatabaseHandles()

	if gcmode := cmd.String(aquaflags.GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive', use 'archive' for full state", aquaflags.GCModeFlag.Name)
	}
	cfg.NoPruning = cfg.NoPruning || cmd.String(aquaflags.GCModeFlag.Name) == "archive"

	if cmd.IsSet(aquaflags.CacheFlag.Name) || cmd.IsSet(aquaflags.CacheGCFlag.Name) {
		cfg.TrieCache = int(cmd.Int(aquaflags.CacheFlag.Name) * cmd.Int(aquaflags.CacheGCFlag.Name) / 100)
	}
	if cmd.IsSet(aquaflags.MinerThreadsFlag.Name) {
		cfg.MinerThreads = int(cmd.Int(aquaflags.MinerThreadsFlag.Name))
	}
	if cmd.IsSet(aquaflags.WorkingDirectoryFlag.Name) {
		Fatalf("Option %q is not supported", aquaflags.WorkingDirectoryFlag.Name)
		//cfg.WorkingDirectory = cmd.String(aquaflags.WorkingDirectoryFlag.Name)
	}
	if cmd.IsSet(aquaflags.JavascriptDirectoryFlag.Name) && (cmd.Name == "console" || cmd.Name == "attach") {
		cfg.JavascriptDirectory = cmd.String(aquaflags.JavascriptDirectoryFlag.Name)
	}
	if cmd.IsSet(aquaflags.ExtraDataFlag.Name) {
		cfg.ExtraData = []byte(cmd.String(aquaflags.ExtraDataFlag.Name))
	}
	if cmd.IsSet(aquaflags.GasPriceFlag.Name) {
		cfg.GasPrice = cmd.Uint(aquaflags.GasPriceFlag.Name)
	}
	if cmd.IsSet(aquaflags.VMEnableDebugFlag.Name) {
		// TODO(fjl): force-enable this in --dev mode
		cfg.EnablePreimageRecording = cmd.Bool(aquaflags.VMEnableDebugFlag.Name)
	}

	if cmd.Bool(aquaflags.DeveloperFlag.Name) {
		// Create new developer account or reuse existing one
		var (
			developer accounts.Account
			err       error
		)
		am := stack.AccountManager()
		if am != nil {
			ks := am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			if accs := ks.Accounts(); len(accs) > 0 {
				developer = ks.Accounts()[0]
			} else {
				developer, err = ks.NewAccount("")
				if err != nil {
					Fatalf("Failed to create developer account: %v", err)
				}
			}
			if err := ks.Unlock(developer, ""); err != nil {
				Fatalf("Failed to unlock developer account: %v", err)
			}
			log.Info("Using developer account", "address", developer.Address)
			cfg.Genesis = core.DeveloperGenesisBlock(uint64(cmd.Int(aquaflags.DeveloperPeriodFlag.Name)), developer.Address)
			if cfg.Genesis.Config.Clique == nil {
				panic("nope")
			}

		}

	}

}

// SetChainId sets the chain ID based on the command line flags (eg --testnet or --chain testnet).
func SetChainId(cmd *cli.Command, cfg *aqua.Config) *params.ChainConfig {
	var chaincfg *params.ChainConfig
	switch {
	case cmd.IsSet(aquaflags.ChainFlag.Name):
		chaincfg = params.GetChainConfig(cmd.String(aquaflags.ChainFlag.Name))
		if chaincfg == nil {
			Fatalf("invalid chain name: %q", cmd.String(aquaflags.ChainFlag.Name))
		}
		cfg.Genesis = core.DefaultGenesisByName(cmd.String(aquaflags.ChainFlag.Name))
	default:
		chaincfg = params.MainnetChainConfig
		cfg.Genesis = core.DefaultGenesisByName("aqua")
	}
	cfg.ChainId = chaincfg.ChainId.Uint64()

	// TODO(fjl): move trie cache generations into config
	if gen := cmd.Int(aquaflags.TrieCacheGenFlag.Name); gen > 0 {
		state.MaxTrieCacheGen = uint16(gen)
	}
	return chaincfg
}

// RegisterAquaService adds an Aquachain client to the stack.
func RegisterAquaService(ctx context.Context, stack *node.Node, cfg *aqua.Config, p2pnodename string) {
	err := stack.Register(func(nodectx *node.ServiceContext) (node.Service, error) {
		return aqua.New(ctx, nodectx, cfg, p2pnodename)
	})
	if err != nil {
		Fatalf("Failed to register the Aquachain service: %v", err)
	}
}

// RegisterAquaStatsService configures the Aquachain Stats daemon and adds it to
// th egiven node.
func RegisterAquaStatsService(stack *node.Node, url string) {
	if err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		// Retrieve both aqua and les services
		var ethServ *aqua.Aquachain
		ctx.Service(&ethServ)

		return aquastats.New(url, ethServ)
	}); err != nil {
		Fatalf("Failed to register the Aquachain Stats service: %v", err)
	}
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(cmd *cli.Command, stack *node.Node) aquadb.Database {
	var (
		cache   = int(cmd.Int(aquaflags.CacheFlag.Name) * cmd.Int(aquaflags.CacheDatabaseFlag.Name) / 100)
		handles = makeDatabaseHandles()
	)
	name := "chaindata"
	chainDb, err := stack.OpenDatabase(name, cache, handles)
	if err != nil {
		Fatalf("Could not open database: %v", err)
	}
	return chainDb
}

func MakeGenesis(cmd *cli.Command) *core.Genesis {
	chain := cmd.String(aquaflags.ChainFlag.Name)
	if chain != "" {
		return core.DefaultGenesisByName(chain)
	}
	return core.DefaultGenesisBlock()
}
func GenesisByChain(chain string) *core.Genesis {
	return core.DefaultGenesisByName(chain)
}

func MakeConsensusEngine(cmd *cli.Command, stack *node.Node) consensus.Engine {
	if cmd.Bool(aquaflags.FakePoWFlag.Name) {
		return aquahash.NewFaker()
	}
	return aquahash.New(&aquahash.Config{
		CacheDir:       stack.ResolvePath(aqua.DefaultConfig.Aquahash.CacheDir),
		CachesInMem:    aqua.DefaultConfig.Aquahash.CachesInMem,
		CachesOnDisk:   aqua.DefaultConfig.Aquahash.CachesOnDisk,
		DatasetDir:     stack.ResolvePath(aqua.DefaultConfig.Aquahash.DatasetDir),
		DatasetsInMem:  aqua.DefaultConfig.Aquahash.DatasetsInMem,
		DatasetsOnDisk: aqua.DefaultConfig.Aquahash.DatasetsOnDisk,
	})
}

// MakeChain creates a chain manager from set command line flags.
func MakeChain(cmd *cli.Command, stack *node.Node) (chain *core.BlockChain, chainDb aquadb.Database) {
	var err error
	chainDb = MakeChainDatabase(cmd, stack)

	config, _, err := core.SetupGenesisBlock(chainDb, MakeGenesis(cmd))
	if err != nil {
		Fatalf("%v", err)
	}

	var engine consensus.Engine = MakeConsensusEngine(cmd, stack)

	if gcmode := cmd.String(aquaflags.GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive'", aquaflags.GCModeFlag.Name)
	}
	cache := &core.CacheConfig{
		Disabled:      cmd.String(aquaflags.GCModeFlag.Name) == "archive",
		TrieNodeLimit: aqua.DefaultConfig.TrieCache,
		TrieTimeLimit: aqua.DefaultConfig.TrieTimeout,
	}
	if cmd.IsSet(aquaflags.CacheFlag.Name) || cmd.IsSet(aquaflags.CacheGCFlag.Name) {
		cache.TrieNodeLimit = int(cmd.Int(aquaflags.CacheFlag.Name) * cmd.Int(aquaflags.CacheGCFlag.Name) / 100)
	}
	vmcfg := vm.Config{EnablePreimageRecording: cmd.Bool(aquaflags.VMEnableDebugFlag.Name)}
	chain, err = core.NewBlockChain(stack.Context(), chainDb, cache, config, engine, vmcfg)
	if err != nil {
		Fatalf("Can't create BlockChain: %v", err)
	}
	return chain, chainDb
}

// MakeConsolePreloads retrieves the absolute paths for the console JavaScript
// scripts to preload before starting.
func MakeConsolePreloads(cmd *cli.Command) []string {
	// Skip preloading if there's nothing to preload
	if cmd.String(aquaflags.PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := cmd.String(aquaflags.JavascriptDirectoryFlag.Name)
	for _, file := range strings.Split(cmd.String(aquaflags.PreloadJSFlag.Name), ",") {
		preloads = append(preloads, common.AbsolutePath(assets, strings.TrimSpace(file)))
	}
	return preloads
}

// MigrateFlags sets the global flag from a local flag when it's set.
// This is a temporary function used for migrating old command/flags to the
// new format.
//
// e.g. aquachain account new --keystore /tmp/mykeystore
//
// is equivalent after calling this method with:
//
// aquachain --keystore /tmp/mykeystore account new
//
// This allows the use of the existing configuration functionality.
// When all flags are migrated this function can be removed and the existing
// configuration functionality must be changed that is uses local flags
func MigrateFlags(action func(_ context.Context, cmd *cli.Command) error) func(context.Context, *cli.Command) error {
	migrated := &MigratedCommand{
		Action: action,
	}
	return migrated.Run
}

type MigratedCommand struct {
	Action func(context.Context, *cli.Command) error
}

func (m *MigratedCommand) Run(ctx context.Context, cmd *cli.Command) error {
	cmdmap := map[string]string{}
	log.Debug("migrating command", "name", cmd.Name,
		"flags", cmd.FlagNames(), "rootname", cmd.Root().Name, "rootflags", cmd.Root().FlagNames(),
		"local", cmd.LocalFlagNames(), "args", cmd.Arguments)
	for _, c := range cmd.Commands {
		cmdmap[c.Name] = c.Name
	}
	for _, name := range cmd.Root().FlagNames() {
		if cmd.Root().IsSet(name) { // set all flags just in case
			if !cmd.IsSet(name) {
				log.Debug("migrating flag from global to subcommand", "name", name, "value", cmd.Root().String(name))
			}
			cmd.Set(name, cmd.Root().String(name))
		}

	}
	// log.Warn("running migrated action", "name", cmd.Name, "args", cmd.Args().Slice(), "flagsEnabled", cmd.LocalFlagNames())
	return m.Action(ctx, cmd)
}

type Cfgopt int

const (
	NoPreviousConfig Cfgopt = 1
)

// MakeConfigNode created a Node and Config, and is called by a subcommand at startup.
func MakeConfigNode(ctx context.Context, cmd *cli.Command, gitCommit string, clientIdentifier string, closemain func(error), opts ...Cfgopt) (*node.Node, *AquachainConfig) {
	// Load defaults.
	useprev := true
	for _, v := range opts {
		if v == NoPreviousConfig { // so dumpconfig doesn't read auto-config without '-config <name>' flag
			useprev = false
		}
	}
	// log.Info("Calling MkConfig", "chain", cmd.String(aquaflags.ChainFlag.Name),
	// 	"config", cmd.String(aquaflags.ConfigFileFlag.Name), "useprev", fmt.Sprint(useprev), "gitCommit", gitCommit, "id", clientIdentifier)
	cfgptr := Mkconfig(cmd.String(aquaflags.ChainFlag.Name), cmd.String(aquaflags.ConfigFileFlag.Name), useprev, gitCommit, clientIdentifier)
	// Apply flags.
	if err := SetNodeConfig(cmd, cfgptr.Node); err != nil {
		Fatalf("Fatal: could not set node config %+v", err)
	}
	cfgptr.Node.Context = ctx
	cfgptr.Node.NoInProc = cmd.Name != "" && cmd.Name != "console" && sense.Getenv("NO_INPROC") == "1"
	cfgptr.Node.CloseMain = closemain
	stack, err := node.New(cfgptr.Node)
	if err != nil {
		Fatalf("Failed to create the protocol stack: %v", err)
	}

	SetAquaConfig(cmd, stack, cfgptr.Aqua)
	if cmd.IsSet(aquaflags.AquaStatsURLFlag.Name) {
		cfgptr.Aquastats.URL = cmd.String(aquaflags.AquaStatsURLFlag.Name)
	}

	return stack, cfgptr
}

func Fatalf(format string, args ...interface{}) {
	log.Crit(fmt.Sprintf(format, args...))
	os.Exit(1)
}

func DefaultNodeConfig(gitCommit, clientIdentifier string) *node.Config {
	if clientIdentifier == "" {
		panic("clientIdentifier must be set")
	}
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "aqua")
	cfg.WSModules = append(cfg.WSModules, "aqua")
	cfg.IPCPath = "aquachain.ipc"
	cfg.P2P.Name = node.GetNodeName(cfg) // cached
	return cfg
}

type AquachainConfig = config.AquachainConfigFull

func Mkconfig(chainName string, configFileOptional string, checkDefaultConfigFiles bool, gitCommit, clientIdentifier string) *AquachainConfig {
	cfgptr := &AquachainConfig{
		Aqua: aqua.DefaultConfig,
		Node: DefaultNodeConfig(gitCommit, clientIdentifier),
	}
	// Load config file.
	file := configFileOptional
	switch {
	default:
		if err := LoadConfigFromFile(file, cfgptr); err != nil {
			Fatalf("error loading config file: %v", err)
		}
		log.Info("Loaded config", "file", file)
	case file == "" && !checkDefaultConfigFiles || file == "none":
		// default config, flags only
	case file == "" && checkDefaultConfigFiles: // find config if exists in working-directory, ~/.aquachain/aquachain.toml, or /etc/aquachain/aquachain.toml
		var userdatadir string
		chainName := chainName
		chainCfg := params.GetChainConfig(chainName)
		if chainCfg == nil {
			Fatalf("invalid chain name: %q, try one of %q", chainName, params.ValidChainNames())
		}
		var slug string // eg: "_testnet" for testnet, "" for mainnet
		if params.MainnetChainConfig == chainCfg {
			userdatadir = node.DefaultDatadir()
		} else {
			userdatadir = node.DefaultDatadirByChain(chainCfg)
		}
		if chainName != "aquachain" && chainName != "mainnet" && chainName != "aqua" {
			slug = "_" + chainName
		}

		fn := "aquachain" + slug + ".toml" // eg: aquachain_testnet.toml or aquachain.toml
		for _, file := range []string{"./" + fn, filepath.Join(userdatadir, fn), "/etc/aquachain/" + fn} {
			log.Debug("Looking for autoconfig file", "file", file)
			if err := LoadConfigFromFile(file, cfgptr); err == nil {
				log.Info("Loaded autoconfig", "file", file)
				break
			} else if !errors.Is(err, fs.ErrNotExist) { // error loading an existing config file
				Fatalf("error loading autoconfig file: %v", err)
			}
		}
	}

	// set defaults asap
	node.DefaultConfig = cfgptr.Node
	return cfgptr
}

func LoadConfigFromFile(file string, cfg *AquachainConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = toml.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// after toml decode, lets expand DataDir (tilde, environmental variables)
	// this keeps config file tidy and sharable.
	// TODO re-tilde on save? or make replacer func
	if err == nil {
		cfg.Node.DataDir = strings.Replace(cfg.Node.DataDir, "~/", "$HOME/", 1)
		cfg.Node.DataDir = os.ExpandEnv(cfg.Node.DataDir)
		cfg.Aqua.Aquahash.DatasetDir = strings.Replace(cfg.Aqua.Aquahash.DatasetDir, "~/", "$HOME/", 1)
		cfg.Aqua.Aquahash.DatasetDir = os.ExpandEnv(cfg.Aqua.Aquahash.DatasetDir)
		cfg.Aqua.Aquahash.CacheDir = strings.Replace(cfg.Aqua.Aquahash.CacheDir, "~/", "$HOME/", 1)
		cfg.Aqua.Aquahash.CacheDir = os.ExpandEnv(cfg.Aqua.Aquahash.CacheDir)
	}
	return err
}

// daemonCommand is the main entry point into the system if the 'daemon' subcommand
// is ran. It creates a default node based on the command line arguments
// and runs it in blocking mode, waiting for it to be shut down.
func daemonStart(ctx context.Context, cmd *cli.Command) error {
	node := MakeFullNode(ctx, cmd)
	startNode(ctx, cmd, node)
	node.Wait()
	return nil
}
