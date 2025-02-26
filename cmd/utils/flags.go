// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for aquachain commands.
package utils

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/aqua/downloader"
	"gitlab.com/aquachain/aquachain/aqua/gasprice"
	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/fdlimit"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/metrics"
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
func NewApp(gitCommit, usage string) *cli.Command {
	app := &cli.Command{
		Name:    filepath.Base(os.Args[0]),
		Usage:   usage,
		Version: params.Version,
	}
	//app.Flags
	app.Name = filepath.Base(os.Args[0])
	app.Version = params.Version
	app.Usage = usage
	return app
}

// These are all the command line flags we support.
// If you add to this list, please remember to include the
// flag in the appropriate command definition.
//
// The flags are defined here so their names and help texts
// are the same for all commands.

var (
	// General settings
	JsonFlag = &cli.BoolFlag{
		Name:  "json",
		Usage: "Print paper keypair in machine-readable JSON format",
	}
	VanityFlag = &cli.StringFlag{
		Name:  "vanity",
		Usage: "Prefix for generating a vanity address (do not include 0x, start small)",
	}
	VanityEndFlag = &cli.StringFlag{
		Name:  "vanityend",
		Usage: "Suffix for generating a vanity address (start small)",
	}
	// General settings
	DataDirFlag = &DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases, IPC socket, and keystore (also see -keystore flag)",
		Value: DirectoryString{node.DefaultConfig.DataDir, false},
	}
	KeyStoreDirFlag = &DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	UseUSBFlag = &cli.BoolFlag{
		Name:  "usb",
		Usage: "Enables monitoring for and managing USB hardware wallets (disabled in pure-go builds)",
	}
	DoitNowFlag = &cli.BoolFlag{
		Name:  "now",
		Usage: "Start the node immediately, do not start countdown",
	}
	ChainFlag = &cli.StringFlag{
		Name:  "chain",
		Usage: "Chain select (aqua, testnet, testnet2, testnet3)",
		Value: "aqua",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			cfg := params.GetChainConfig(v)
			if cfg == nil {
				return fmt.Errorf("invalid chain name: %q, try one of %q", v, params.ValidChainNames())
			}
			// cmd.Set("chain", v)
			return nil
		},
	}
	AlertModeFlag = &cli.BoolFlag{
		Name:  "alerts",
		Usage: "Enable alert notifications (requires env $ALERT_TOKEN, $ALERT_PLATFORM, and $ALERT_CHANNEL)",
	}
	TestnetFlag = &cli.BoolFlag{
		Name:  "testnet",
		Usage: "Deprecated: use --chain=testnet",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain <name>'", "testnet")
		},
	}
	Testnet2Flag = &cli.BoolFlag{
		Name:  "testnet2",
		Usage: "Deprecated: use --chain=testnet2",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain <name>'", "testnet2")
		},
	}
	Testnet3Flag = &cli.BoolFlag{
		Name:  "testnet3",
		Usage: "Deprecated: use --chain=testnet3",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain <name>'", "testnet3")
		},
	}
	NetworkEthFlag = &cli.BoolFlag{
		Name:  "ethereum",
		Usage: "Deprecated: dont use",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain <name>'", "ethereum")
		},
	}
	DeveloperFlag = &cli.BoolFlag{
		Name:  "dev",
		Usage: "Ephemeral proof-of-authority network with a pre-funded developer account, mining enabled",
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			return fmt.Errorf("flag %q is deprecated, use '-chain <name>'", "dev")
		},
	} // TODO: '-chain dev'
	DeveloperPeriodFlag = &cli.IntFlag{
		Name:  "dev.period",
		Usage: "Block period to use in developer mode (0 = mine only if transaction pending)",
	}
	IdentityFlag = &cli.StringFlag{
		Name:  "identity",
		Usage: "Custom node name (used in p2p networking, default is aquachain version)",
	}
	WorkingDirectoryFlag = &cli.StringFlag{
		Name: "WorkingDirectory",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v != "" {
				return fmt.Errorf("flag %q is deprecated, use '-jspath <path>'", "WorkingDirectory")
			}
			return nil
		},
		Hidden: true,
	}
	JavascriptDirectoryFlag = &cli.StringFlag{
		Name:      "jspath",
		TakesFile: true,
		Usage:     "Working directory for importing JS files into console (default = current directory)",
		Value:     ".",
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v == "none" {
				return nil
			}
			if v == "" {
				return fmt.Errorf("invalid directory: %q", v)
			}
			if v == "." {
				return nil
			}
			stat, err := os.Stat(v)
			if err != nil {
				return err
			}
			if !stat.IsDir() {
				return fmt.Errorf("invalid directory: %q", v)
			}
			return nil
		},
	}

	FastSyncFlag = &cli.BoolFlag{
		Name:  "fast",
		Usage: "Enable fast syncing through state downloads",
	}
	tmpdefaultSyncMode = aqua.DefaultConfig.SyncMode

	SyncModeFlag = &cli.StringFlag{
		Name:  "syncmode",
		Usage: `Blockchain sync mode ("fast", "full")`,
		Value: tmpdefaultSyncMode.String(),
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v != "fast" && v != "full" && v != "offline" {
				return fmt.Errorf("invalid sync mode: %q", v)
			}
			return nil
		},
	}
	GCModeFlag = &cli.StringFlag{
		Name:  "gcmode",
		Usage: `GC mode to use, either "full" or "archive". Use "archive" for full accurate state (for example, 'admin.supply')`,
		Value: "archive",
	}
)
var (
	// Aquahash settings
	AquahashCacheDirFlag = &DirectoryFlag{
		Name:  "aquahash.cachedir",
		Usage: "Directory to store the aquahash verification caches (default = inside the datadir)",
	}
	AquahashCachesInMemoryFlag = &cli.IntFlag{
		Name:  "aquahash.cachesinmem",
		Usage: "Number of recent aquahash caches to keep in memory (16MB each)",
		Value: int64(aqua.DefaultConfig.Aquahash.CachesInMem),
	}
	AquahashCachesOnDiskFlag = &cli.IntFlag{
		Name:  "aquahash.cachesondisk",
		Usage: "Number of recent aquahash caches to keep on disk (16MB each)",
		Value: int64(aqua.DefaultConfig.Aquahash.CachesOnDisk),
	}
	AquahashDatasetDirFlag = &DirectoryFlag{
		Name:  "aquahash.dagdir",
		Usage: "Directory to store the aquahash mining DAGs (default = inside home folder)",
		Value: DirectoryString{aqua.DefaultConfig.Aquahash.DatasetDir, false},
	}
	AquahashDatasetsInMemoryFlag = &cli.IntFlag{
		Name:  "aquahash.dagsinmem",
		Usage: "Number of recent aquahash mining DAGs to keep in memory (1+GB each)",
		Value: int64(aqua.DefaultConfig.Aquahash.DatasetsInMem),
	}
	AquahashDatasetsOnDiskFlag = &cli.IntFlag{
		Name:  "aquahash.dagsondisk",
		Usage: "Number of recent aquahash mining DAGs to keep on disk (1+GB each)",
		Value: int64(aqua.DefaultConfig.Aquahash.DatasetsOnDisk),
	}
	// Transaction pool settings
	TxPoolNoLocalsFlag = &cli.BoolFlag{
		Name:  "txpool.nolocals",
		Usage: "Disables price exemptions for locally submitted transactions",
	}
	TxPoolJournalFlag = &cli.StringFlag{
		Name:  "txpool.journal",
		Usage: "Disk journal for local transaction to survive node restarts",
		Value: core.DefaultTxPoolConfig.Journal,
	}
	TxPoolRejournalFlag = &cli.DurationFlag{
		Name:  "txpool.rejournal",
		Usage: "Time interval to regenerate the local transaction journal",
		Value: core.DefaultTxPoolConfig.Rejournal,
	}
	TxPoolPriceLimitFlag = &cli.UintFlag{
		Name:  "txpool.pricelimit",
		Usage: "Minimum gas price limit to enforce for acceptance into the pool",
		Value: aqua.DefaultConfig.TxPool.PriceLimit,
	}
	TxPoolPriceBumpFlag = &cli.UintFlag{
		Name:  "txpool.pricebump",
		Usage: "Price bump percentage to replace an already existing transaction",
		Value: aqua.DefaultConfig.TxPool.PriceBump,
	}
	TxPoolAccountSlotsFlag = &cli.UintFlag{
		Name:  "txpool.accountslots",
		Usage: "Minimum number of executable transaction slots guaranteed per account",
		Value: aqua.DefaultConfig.TxPool.AccountSlots,
	}
	TxPoolGlobalSlotsFlag = &cli.UintFlag{
		Name:  "txpool.globalslots",
		Usage: "Maximum number of executable transaction slots for all accounts",
		Value: aqua.DefaultConfig.TxPool.GlobalSlots,
	}
	TxPoolAccountQueueFlag = &cli.UintFlag{
		Name:  "txpool.accountqueue",
		Usage: "Maximum number of non-executable transaction slots permitted per account",
		Value: aqua.DefaultConfig.TxPool.AccountQueue,
	}
	TxPoolGlobalQueueFlag = &cli.UintFlag{
		Name:  "txpool.globalqueue",
		Usage: "Maximum number of non-executable transaction slots for all accounts",
		Value: aqua.DefaultConfig.TxPool.GlobalQueue,
	}
	TxPoolLifetimeFlag = &cli.DurationFlag{
		Name:  "txpool.lifetime",
		Usage: "Maximum amount of time non-executable transaction are queued",
		Value: aqua.DefaultConfig.TxPool.Lifetime,
	}
	// Performance tuning settings
	CacheFlag = &cli.IntFlag{
		Name:  "cache",
		Usage: "Megabytes of memory allocated to internal caching (consider 2048)",
		Value: 1024,
	}
	CacheDatabaseFlag = &cli.IntFlag{
		Name:  "cache.database",
		Usage: "Percentage of cache memory allowance to use for database io",
		Value: 75,
	}
	CacheGCFlag = &cli.IntFlag{
		Name:  "cache.gc",
		Usage: "Percentage of cache memory allowance to use for trie pruning",
		Value: 25,
	}
	TrieCacheGenFlag = &cli.IntFlag{
		Name:  "trie-cache-gens",
		Usage: "Number of trie node generations to keep in memory",
		Value: int64(state.MaxTrieCacheGen),
	}
	// Miner settings
	MiningEnabledFlag = &cli.BoolFlag{
		Name:  "mine",
		Usage: "Enable mining (not optimized, not recommended for mainnet)",
	}
	MinerThreadsFlag = &cli.IntFlag{
		Name:  "minerthreads",
		Usage: "Number of CPU threads to use for mining",
		Value: int64(runtime.NumCPU()),
	}
	TargetGasLimitFlag = &cli.UintFlag{
		Name:        "targetgaslimit",
		Usage:       "Target gas limit sets the artificial target gas floor for the blocks to mine",
		Value:       params.GenesisGasLimit,
		Destination: &params.TargetGasLimit,
	}
	AquabaseFlag = &cli.StringFlag{
		Name:  "aquabase",
		Usage: "Public address for block mining rewards (default = first account created)",
		Value: "0",
	}
	GasPriceFlag = &cli.GenericFlag{
		Name:  "gasprice",
		Usage: "Minimal gas price to accept for mining a transactions",
		Value: (*bigValue)(aqua.DefaultConfig.GasPrice),
	}
	ExtraDataFlag = &cli.StringFlag{
		Name:  "extradata",
		Usage: "Block extra data set by the miner (default = client version)",
	}
	// Account settings
	UnlockedAccountFlag = &cli.StringFlag{
		Name:  "unlock",
		Usage: "Comma separated list of accounts to unlock (CAREFUL!)",
		Value: "",
	}
	PasswordFileFlag = &cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}

	VMEnableDebugFlag = &cli.BoolFlag{
		Name:  "vmdebug",
		Usage: "Record information useful for VM and contract debugging",
	}
	// Logging and debug settings
	AquaStatsURLFlag = &cli.StringFlag{
		Name:  "aquastats",
		Usage: "Reporting URL of a aquastats service (nodename:secret@host:port)",
	}
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:  metrics.MetricsEnabledFlag,
		Usage: "Enable metrics collection and reporting",
	}
	FakePoWFlag = &cli.BoolFlag{
		Name:  "fakepow",
		Usage: "Disables proof-of-work verification",
	}
	NoCompactionFlag = &cli.BoolFlag{
		Name:  "nocompaction",
		Usage: "Disables db compaction after import",
	}
	// RPC settings
	RPCEnabledFlag = &cli.BoolFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = &cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: node.DefaultHTTPHost,
	}
	RPCPortFlag = &cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: node.DefaultHTTPPort,
	}
	RPCCORSDomainFlag = &cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCVirtualHostsFlag = &cli.StringFlag{
		Name:  "rpcvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: "localhost",
	}

	RPCApiFlag = &cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	RPCUnlockFlag = &cli.BoolFlag{
		Name:  "UNSAFE_RPC_UNLOCK",
		Usage: "",
	}
	IPCDisabledFlag = &cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = &DirectoryFlag{
		Name:  "ipcpath",
		Usage: "Filename for IPC socket/pipe within the datadir (explicit paths escape it)",
	}
	WSEnabledFlag = &cli.BoolFlag{
		Name:  "ws",
		Usage: "Enable the WS-RPC server",
	}
	WSListenAddrFlag = &cli.StringFlag{
		Name:  "wsaddr",
		Usage: "WS-RPC server listening interface",
		Value: node.DefaultWSHost,
	}
	WSPortFlag = &cli.IntFlag{
		Name:  "wsport",
		Usage: "WS-RPC server listening port",
		Value: node.DefaultWSPort,
	}
	WSApiFlag = &cli.StringFlag{
		Name:  "wsapi",
		Usage: "API's offered over the WS-RPC interface",
		Value: "",
	}
	WSAllowedOriginsFlag = &cli.StringFlag{
		Name:  "wsorigins",
		Usage: "Origins from which to accept websockets requests (see also rpcvhosts)",
		Value: "",
	}
	RPCAllowIPFlag = &cli.StringFlag{
		Name:  "allowip",
		Usage: "Comma separated allowed RPC clients (CIDR notation OK) (http/ws)",
		Value: "127.0.0.1/24",
	}
	RPCBehindProxyFlag = &cli.BoolFlag{
		Name:  "behindproxy",
		Usage: "If RPC is behind a reverse proxy. Changes the way IP is fetched when comparing to allowed IP addresses",
	}
	ExecFlag = &cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement",
	}
	PreloadJSFlag = &cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}

	// Network Settings
	MaxPeersFlag = &cli.IntFlag{
		Name:  "maxpeers",
		Usage: "Maximum number of network peers (network disabled if set to 0)",
		Value: 25,
	}
	MaxPendingPeersFlag = &cli.IntFlag{
		Name:  "maxpendpeers",
		Usage: "Maximum number of pending connection attempts (defaults used if set to 0)",
		Value: 0,
	}
	ListenPortFlag = &cli.IntFlag{
		Name:  "port",
		Usage: "Network listening port",
		Value: 21303,
	}
	ListenAddrFlag = &cli.StringFlag{
		Name:  "addr",
		Usage: "Network listening addr (all interfaces, port 21303 TCP and UDP)",
		Value: "",
	}
	BootnodesFlag = &cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value: "",
	}
	// BootnodesV4Flag = &cli.StringFlag{
	// 	Name:  "bootnodesv4",
	// 	Usage: "Comma separated enode URLs for P2P v4 discovery bootstrap (light server, full nodes)",
	// 	Value: "",
	// }
	NodeKeyFileFlag = &cli.StringFlag{
		Name:  "nodekey",
		Usage: "P2P node key file",
	}
	NodeKeyHexFlag = &cli.StringFlag{
		Name:  "nodekeyhex",
		Usage: "P2P node key as hex (for testing)",
	}
	NATFlag = &cli.StringFlag{
		Name:  "nat",
		Usage: "NAT port mapping mechanism (any|none|upnp|pmp|extip:<IP>)",
		Value: "any",
	}
	NoDiscoverFlag = &cli.BoolFlag{
		Name:  "nodiscover",
		Usage: "Disables the peer discovery mechanism (manual peer addition)",
	}
	OfflineFlag = &cli.BoolFlag{
		Name:  "offline",
		Usage: "Disables peer discovery and sets nat=none, still listens on tcp/udp port",
	}
	NoKeysFlag = &cli.BoolFlag{
		Name:  "nokeys",
		Usage: "Disables keystore",
	}
	NetrestrictFlag = &cli.StringFlag{
		Name:  "netrestrict",
		Usage: "Restricts network communication to the given IP networks (CIDR masks)",
	}
	// Gas price oracle settings
	GpoBlocksFlag = &cli.IntFlag{
		Name:  "gpoblocks",
		Usage: "Number of recent blocks to check for gas prices",
		Value: int64(aqua.DefaultConfig.GPO.Blocks),
	}
	GpoPercentileFlag = &cli.IntFlag{
		Name:  "gpopercentile",
		Usage: "Suggested gas price is the given percentile of a set of recent transaction gas prices",
		Value: int64(aqua.DefaultConfig.GPO.Percentile),
	}
	HF8MainnetFlag = &cli.IntFlag{
		Name:  "hf8",
		Usage: "Hard fork #8 activation block",
		Value: -1,
	}
)

// MakeDataDir retrieves the currently requested data directory, terminating
// if none (or the empty string) is specified. If the node is starting a testnet,
// the a subdirectory of the specified datadir will be used.
func MakeDataDir(cmd *cli.Command) string {
	if datadir := cmd.String(DataDirFlag.Name); datadir != "" {
		return datadir
	}
	chainName := cmd.String(ChainFlag.Name)
	if chainName == "" {
		Fatalf("No chain selected, no data directory specified")
	}
	return filepath.Join(node.DefaultConfig.DataDir, chainName)
}

// 	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
// 	return ""
// }

// setNodeKey creates a node key from set command line flags, either loading it
// from a file or as a specified hex value. If neither flags were provided, this
// method returns nil and an emphemeral key is to be generated.
func setNodeKey(cmd *cli.Command, cfg *p2p.Config) {
	var (
		hex  = cmd.String(NodeKeyHexFlag.Name)
		file = cmd.String(NodeKeyFileFlag.Name)
		key  *btcec.PrivateKey
		err  error
	)
	switch {
	case file != "" && hex != "":
		Fatalf("Options %q and %q are mutually exclusive", NodeKeyFileFlag.Name, NodeKeyHexFlag.Name)
	case file != "":
		if key, err = crypto.LoadECDSA(file); err != nil {
			Fatalf("Option %q: %v", NodeKeyFileFlag.Name, err)
		}
		cfg.PrivateKey = key
	case hex != "":
		if key, err = crypto.HexToBtcec(hex); err != nil {
			Fatalf("Option %q: %v", NodeKeyHexFlag.Name, err)
		}
		cfg.PrivateKey = key
	}
}

// setNodeUserIdent creates the user identifier from CLI flags.
func setNodeUserIdent(cmd *cli.Command, cfg *node.Config) {
	if identity := cmd.String(IdentityFlag.Name); len(identity) > 0 {
		cfg.UserIdent = identity
	}
}

// returns chainname, chaincfg, bootnodes, datadir
func getStuff(cmd *cli.Command) (string, *params.ChainConfig, []*discover.Node, DirectoryConfig) {
	chainName := cmd.String(ChainFlag.Name)
	log.Warn("chainName", "chainName", chainName)
	if chainName == "" {
		Fatalf("No chain selected")
		panic("no chain name")
	}
	chaincfg := params.GetChainConfig(chainName)
	if chaincfg == nil {
		// check directory
		expected := filepath.Join(node.DefaultConfig.DataDir, chainName)
		stat, err := os.Stat(expected)
		if err == nil && stat.IsDir() {
			chaincfg, err = params.LoadChainConfigFile(expected)
			if err != nil {
				Fatalf("Failed to load custom chain config: %v", err)
			}
		}
	}
	if chaincfg == nil {
		Fatalf("invalid chain name: %q", cmd.String(ChainFlag.Name))
		panic("bad chain name")
	}
	switch chainName { // TODO: remove once disabled chain-flags are cleaned
	case "dev":
		cmd.Set(DeveloperFlag.Name, "true")
	case "testnet":
		cmd.Set(TestnetFlag.Name, "true")
	case "testnet2":
		cmd.Set(Testnet2Flag.Name, "true")
	case "testnet3":
		cmd.Set(Testnet3Flag.Name, "true")
	}
	return chainName, chaincfg, getBootstrapNodes(cmd), switchDatadir(cmd, chaincfg)
}

func getBootstrapNodes(cmd *cli.Command) []*discover.Node {
	if cmd.IsSet(BootnodesFlag.Name) { // custom bootnodes flag
		return StringToBootstraps(strings.Split(cmd.String(BootnodesFlag.Name), ","))
	}
	if cmd.IsSet(NoDiscoverFlag.Name) {
		return []*discover.Node{}
	}
	chainName := cmd.String(ChainFlag.Name)
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
// 	if cmd.IsSet(ChainFlag.Name) {
// 		chainName := cmd.String(ChainFlag.Name)
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
	chainName := cmd.String(ChainFlag.Name)
	if chainName == "" {
		switch {
		case cmd.Bool(TestnetFlag.Name):
			chainName = "testnet"
		case cmd.Bool(Testnet2Flag.Name):
			chainName = "testnet2"
		default:
			chainName = "aqua"
		}
	}
	chaincfg := params.GetChainConfig(chainName)
	if chaincfg == nil {
		Fatalf("invalid chain name: %q", cmd.String(ChainFlag.Name))
	}
	return chaincfg
}*/

// setListenAddress creates a TCP listening address string from set command
// line flags.
func getListenAddress(cmd *cli.Command) string {
	var listenaddr string
	chainName := cmd.String(ChainFlag.Name)
	if chainName == "" {
		panic("no chain name")
	}
	chainCfg := params.GetChainConfig(chainName)
	if chainCfg != nil && chainCfg.DefaultPortNumber != 0 {
		listenaddr = fmt.Sprintf("%s:%d", "0.0.0.0", chainCfg.DefaultPortNumber)
	}

	// flag overrides
	if cmd.IsSet(ListenAddrFlag.Name) && cmd.IsSet(ListenPortFlag.Name) {
		listenaddr = fmt.Sprintf("%s:%d", cmd.String(ListenAddrFlag.Name), cmd.Int(ListenPortFlag.Name))
	} else if !cmd.IsSet(ListenAddrFlag.Name) && cmd.IsSet(ListenPortFlag.Name) {
		listenaddr = fmt.Sprintf(":%d", cmd.Int(ListenPortFlag.Name))
	} else if cmd.IsSet(ListenAddrFlag.Name) {
		listenaddr = cmd.String(ListenAddrFlag.Name)
	}

	if listenaddr == "" {
		Fatalf("No listen address specified")
	}

	return listenaddr
}

// setNAT creates a port mapper from command line flags.
func setNAT(cmd *cli.Command, cfg *p2p.Config) {
	if cmd.IsSet(NATFlag.Name) {
		_, err := nat.Parse(cmd.String(NATFlag.Name))
		if err != nil {
			Fatalf("Option %s: %v", NATFlag.Name, err)
		}
		cfg.NAT = nat.NatString(cmd.String(NATFlag.Name))
	}
	if cmd.IsSet(OfflineFlag.Name) {
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
	if cmd.Bool(RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if cmd.IsSet(RPCListenAddrFlag.Name) && cmd.IsSet(UnlockedAccountFlag.Name) && !cmd.IsSet(RPCUnlockFlag.Name) {
			Fatalf("Woah there! By default, using -rpc and -unlock is \"safe\", (localhost).\n" +
				"But you shouldn't use --rpcaddr with --unlock flag.\n" +
				"If you really know what you are doing and would like to unlock a wallet while" +
				"hosting a public HTTP RPC node, use the -UNSAFE_RPC_UNLOCK flag. See -allowip flag to restrict access")
		}
		if cmd.IsSet(RPCListenAddrFlag.Name) && cmd.IsSet(UnlockedAccountFlag.Name) {
			// allow public rpc with unlocked account, exposed only via 'private' api namespace (aqua.sendTransaction and aqua.sign are disabled)
			keystore.SetNoSignMode()
		}
		if cmd.IsSet(RPCListenAddrFlag.Name) {
			cfg.HTTPHost = cmd.String(RPCListenAddrFlag.Name)
		}
	}

	if cmd.IsSet(RPCPortFlag.Name) {
		cfg.HTTPPort = int(cmd.Int(RPCPortFlag.Name))
	}
	if cmd.IsSet(RPCCORSDomainFlag.Name) {
		cfg.HTTPCors = splitAndTrim(cmd.String(RPCCORSDomainFlag.Name))
	}
	if cmd.IsSet(RPCApiFlag.Name) {
		cfg.HTTPModules = parseRpcFlags(cfg.HTTPModules, splitAndTrim(cmd.String(RPCApiFlag.Name)))
	}

	cfg.HTTPVirtualHosts = splitAndTrim(cmd.String(RPCVirtualHostsFlag.Name))
	cfg.RPCAllowIP = splitAndTrim(cmd.String(RPCAllowIPFlag.Name))
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
	if cmd.Bool(WSEnabledFlag.Name) && cfg.WSHost == "" {
		cfg.WSHost = "127.0.0.1"
		if cmd.IsSet(WSListenAddrFlag.Name) {
			cfg.WSHost = cmd.String(WSListenAddrFlag.Name)
		}
	}

	if cmd.IsSet(WSPortFlag.Name) {
		cfg.WSPort = int(cmd.Int(WSPortFlag.Name))
	}
	if cmd.IsSet(WSAllowedOriginsFlag.Name) {
		cfg.WSOrigins = splitAndTrim(cmd.String(WSAllowedOriginsFlag.Name))
	}
	if cmd.IsSet(WSApiFlag.Name) {
		cfg.WSModules = splitAndTrim(cmd.String(WSApiFlag.Name))
	}
}

// setIPC creates an IPC path configuration from the set command line flags,
// returning an empty string if IPC was explicitly disabled, or the set path.
func setIPC(cmd *cli.Command, cfg *node.Config) {
	checkExclusive(cmd, IPCDisabledFlag, IPCPathFlag)
	switch {
	case cmd.Bool(IPCDisabledFlag.Name):
		cfg.IPCPath = ""
	case cmd.IsSet(IPCPathFlag.Name):
		cfg.IPCPath = cmd.String(IPCPathFlag.Name)
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
	if cmd.IsSet(AquabaseFlag.Name) {
		account, err := MakeAddress(ks, cmd.String(AquabaseFlag.Name))
		if err != nil {
			Fatalf("Option %q: %v", AquabaseFlag.Name, err)
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
	path := strings.TrimSpace(cmd.String(PasswordFileFlag.Name))
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
	if cmd.IsSet(MaxPeersFlag.Name) {
		cfg.MaxPeers = int(cmd.Int(MaxPeersFlag.Name))
	}

	if cmd.IsSet(MaxPendingPeersFlag.Name) {
		cfg.MaxPendingPeers = int(cmd.Int(MaxPendingPeersFlag.Name))
	}

	if cmd.IsSet(OfflineFlag.Name) {
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

	if cmd.IsSet(NoDiscoverFlag.Name) {
		cfg.NoDiscovery = true
	}
	if cmd.Bool(Testnet2Flag.Name) {
		cfg.NoDiscovery = true
	}
	if netrestrict := cmd.String(NetrestrictFlag.Name); netrestrict != "" {
		list, err := netutil.ParseNetlist(netrestrict)
		if err != nil {
			Fatalf("Option %q: %v", NetrestrictFlag.Name, err)
		}
		cfg.NetRestrict = list
	}

	if cmd.Bool(DeveloperFlag.Name) {
		// --dev mode can't use p2p networking.
		cfg.MaxPeers = 0
		cfg.ListenAddr = "127.0.0.1:0" // random local port
		cfg.NoDiscovery = true
	}

	/*
		switch cmd.String(ChainFlag.Name) {
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
		// if cfg.ListenAddr == "" && cmd.Bool(TestnetFlag.Name) && !cmd.IsSet(ListenPortFlag.Name) {
		// 	cfg.ListenAddr = "0.0.0.0:21304"
		// }
		// if cfg.ListenAddr == "" && cmd.Bool(NetworkEthFlag.Name) && !cmd.IsSet(ListenPortFlag.Name) {
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
	var cfg DirectoryConfig
	// var newdatadir string
	if cmd.IsSet(KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = cmd.String(KeyStoreDirFlag.Name)
	}
	if cmd.IsSet(DataDirFlag.Name) {
		cfg.DataDir = cmd.String(DataDirFlag.Name)
		log.Info("set custom datadir", "path", cfg.DataDir)
	}
	if cfg.DataDir == "" && chaincfg == nil {
		Fatalf("No chain and no data directory specified. Please specify a chain with --chain or a data directory with --datadir")
	}
	if cfg.DataDir != "" {
		println("datadir already set:", cfg.DataDir)
		return cfg
	}
	cfg.DataDir = node.DefaultDatadirByChain(chaincfg)
	return cfg

	// switch {
	// case cmd.IsSet(ChainFlag.Name):
	// 	chainName := cmd.String(ChainFlag.Name)
	// 	chaincfg := params.GetChainConfig(chainName)
	// 	if chaincfg == nil {
	// 		return errors.New("invalid config name")
	// 	}
	// 	if chaincfg == params.MainnetChainConfig {
	// 		newdatadir = node.DefaultConfig.DataDir
	// 	} else {
	// 		newdatadir = filepath.Join(node.DefaultConfig.DataDir, "chains", chainName)
	// 	}
	// 	cfg.P2P.ChainId = chaincfg.ChainId.Uint64()
	// case cmd.IsSet(NetworkIdFlag.Name):
	// 	cfg.P2P.ChainId = cmd.Uint(NetworkIdFlag.Name)
	// 	newdatadir = filepath.Join(node.DefaultConfig.DataDir, fmt.Sprintf("chainid-%v", cfg.P2P.ChainId))
	// case cmd.Bool(DeveloperFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultConfig.DataDir, "develop")
	// 	cfg.P2P.ChainId = 1337
	// case cmd.Bool(TestnetFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultConfig.DataDir, "testnet")
	// 	cfg.P2P.ChainId = params.TestnetChainConfig.ChainId.Uint64()
	// case cmd.Bool(Testnet2Flag.Name):
	// 	newdatadir = filepath.Join(node.DefaultConfig.DataDir, "testnet2")
	// 	cfg.P2P.ChainId = params.Testnet2ChainConfig.ChainId.Uint64()
	// case cmd.Bool(NetworkEthFlag.Name):
	// 	newdatadir = filepath.Join(node.DefaultConfig.DataDir, "ethereum")
	// 	cfg.P2P.ChainId = params.EthnetChainConfig.ChainId.Uint64()
	// default:
	// 	// mainnet
	// 	cfg.P2P.ChainId = params.MainnetChainConfig.ChainId.Uint64()
	// 	newdatadir = node.DefaultConfig.DataDir
	// }

	// if cmd.IsSet(KeyStoreDirFlag.Name) {
	// 	cfg.KeyStoreDir = cmd.String(KeyStoreDirFlag.Name)
	// 	if cfg.KeyStoreDir == "" {
	// 		cfg.NoKeys = true
	// 	}
	// }
	// if cmd.IsSet(DataDirFlag.Name) {
	// 	newdatadir = cmd.String(DataDirFlag.Name)
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
	log.Info("Loading...", "Chain Select", chainName, "ChainID", chaincfg.ChainId, "Datadir", directoryCfg.DataDir)
	cfg.DataDir = directoryCfg.DataDir
	node.DefaultConfig.DataDir = directoryCfg.DataDir // in case something uses it
	cfg.KeyStoreDir = directoryCfg.KeyStoreDir
	cfg.P2P.ChainId = chaincfg.ChainId.Uint64()
	cfg.P2P.BootstrapNodes = bootstrapNodes

	SetP2PConfig(cmd, cfg.P2P)
	setIPC(cmd, cfg)
	setHTTP(cmd, cfg)
	setWS(cmd, cfg)
	setNodeUserIdent(cmd, cfg)
	if cmd.IsSet(NoKeysFlag.Name) {
		cfg.NoKeys = cmd.Bool(NoKeysFlag.Name)
	}

	if cfg.NoKeys {
		log.Info("No-Keys mode")
	}
	if cmd.IsSet(UseUSBFlag.Name) {
		cfg.UseUSB = cmd.Bool(UseUSBFlag.Name)
	}
	if cmd.IsSet(RPCBehindProxyFlag.Name) || os.Getenv("REVERSE_PROXY") != "" {
		cfg.RPCBehindProxy = cmd.Bool(RPCBehindProxyFlag.Name)
	}
	return nil
}

func setGPO(cmd *cli.Command, cfg *gasprice.Config) {
	if cmd.IsSet(GpoBlocksFlag.Name) {
		cfg.Blocks = int(cmd.Int(GpoBlocksFlag.Name))
	}
	if cmd.IsSet(GpoPercentileFlag.Name) {
		cfg.Percentile = int(cmd.Int(GpoPercentileFlag.Name))
	}
}

func setTxPool(cmd *cli.Command, cfg *core.TxPoolConfig) {
	if cmd.IsSet(TxPoolNoLocalsFlag.Name) {
		cfg.NoLocals = cmd.Bool(TxPoolNoLocalsFlag.Name)
	}
	if cmd.IsSet(TxPoolJournalFlag.Name) {
		cfg.Journal = cmd.String(TxPoolJournalFlag.Name)
	}
	if cmd.IsSet(TxPoolRejournalFlag.Name) {
		cfg.Rejournal = cmd.Duration(TxPoolRejournalFlag.Name)
	}
	if cmd.IsSet(TxPoolPriceLimitFlag.Name) {
		cfg.PriceLimit = cmd.Uint(TxPoolPriceLimitFlag.Name)
	}
	if cmd.IsSet(TxPoolPriceBumpFlag.Name) {
		cfg.PriceBump = cmd.Uint(TxPoolPriceBumpFlag.Name)
	}
	if cmd.IsSet(TxPoolAccountSlotsFlag.Name) {
		cfg.AccountSlots = cmd.Uint(TxPoolAccountSlotsFlag.Name)
	}
	if cmd.IsSet(TxPoolGlobalSlotsFlag.Name) {
		cfg.GlobalSlots = cmd.Uint(TxPoolGlobalSlotsFlag.Name)
	}
	if cmd.IsSet(TxPoolAccountQueueFlag.Name) {
		cfg.AccountQueue = cmd.Uint(TxPoolAccountQueueFlag.Name)
	}
	if cmd.IsSet(TxPoolGlobalQueueFlag.Name) {
		cfg.GlobalQueue = cmd.Uint(TxPoolGlobalQueueFlag.Name)
	}
	if cmd.IsSet(TxPoolLifetimeFlag.Name) {
		cfg.Lifetime = cmd.Duration(TxPoolLifetimeFlag.Name)
	}
}

func setAquahash(cmd *cli.Command, cfg *aqua.Config) {
	if cmd.IsSet(AquahashCacheDirFlag.Name) {
		cfg.Aquahash.CacheDir = cmd.String(AquahashCacheDirFlag.Name)
	}
	if cmd.IsSet(AquahashDatasetDirFlag.Name) {
		cfg.Aquahash.DatasetDir = cmd.String(AquahashDatasetDirFlag.Name)
	}
	if cmd.IsSet(AquahashCachesInMemoryFlag.Name) {
		cfg.Aquahash.CachesInMem = int(cmd.Int(AquahashCachesInMemoryFlag.Name))
	}
	if cmd.IsSet(AquahashCachesOnDiskFlag.Name) {
		cfg.Aquahash.CachesOnDisk = int(cmd.Int(AquahashCachesOnDiskFlag.Name))
	}
	if cmd.IsSet(AquahashDatasetsInMemoryFlag.Name) {
		cfg.Aquahash.DatasetsInMem = int(cmd.Int(AquahashDatasetsInMemoryFlag.Name))
	}
	if cmd.IsSet(AquahashDatasetsOnDiskFlag.Name) {
		cfg.Aquahash.DatasetsOnDisk = int(cmd.Int(AquahashDatasetsOnDiskFlag.Name))
	}
}

// checkExclusive verifies that only a single isntance of the provided flags was
// set by the user. Each flag might optionally be followed by a string type to
// specialize it further.
func checkExclusive(cmd *cli.Command, args ...cli.Flag) {
	set := make([]string, 0, 1)
	for i := 0; i < len(args); i++ {
		// Make sure the next argument is a flag and skip if not set
		flag, ok := args[i].(cli.Flag)
		if !ok {
			panic(fmt.Sprintf("invalid argument, not cli.Flag type: %T", args[i]))
		}
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
				panic(fmt.Sprintf("invalid argument, not cli.Flag or string extension: %T", args[i+1]))
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
	if cmd.IsSet(HF8MainnetFlag.Name) && chaincfg.HF[8] == nil {
		chaincfg.HF[8] = big.NewInt(0).SetUint64(uint64(cmd.Uint(HF8MainnetFlag.Name)))
	}
}

// SetAquaConfig applies aqua-related command line flags to the config.
func SetAquaConfig(cmd *cli.Command, stack *node.Node, cfg *aqua.Config) {
	// Avoid conflicting network flags
	// note: these are disabled flags, but Set is still called before this function
	checkExclusive(cmd, DeveloperFlag, TestnetFlag, Testnet2Flag, NetworkEthFlag)
	checkExclusive(cmd, DeveloperFlag, NoKeysFlag)
	checkExclusive(cmd, FastSyncFlag, SyncModeFlag, OfflineFlag)
	if cmd.Bool(AlertModeFlag.Name) {
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
	case cmd.Bool(OfflineFlag.Name):
		cfg.SyncMode = downloader.OfflineSync
	case cmd.IsSet(SyncModeFlag.Name):
		err := cfg.SyncMode.UnmarshalText([]byte(cmd.String(SyncModeFlag.Name)))
		if err != nil {
			Fatalf("Failed to parse sync mode: %v", err)
		}
	case cmd.Bool(FastSyncFlag.Name):
		cfg.SyncMode = downloader.FastSync
	}

	if cmd.IsSet(CacheFlag.Name) || cmd.IsSet(CacheDatabaseFlag.Name) {
		cfg.DatabaseCache = int(cmd.Int(CacheFlag.Name) * cmd.Int(CacheDatabaseFlag.Name) / 100)
	}
	cfg.DatabaseHandles = makeDatabaseHandles()

	if gcmode := cmd.String(GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive', use 'archive' for full state", GCModeFlag.Name)
	}
	cfg.NoPruning = cfg.NoPruning || cmd.String(GCModeFlag.Name) == "archive"

	if cmd.IsSet(CacheFlag.Name) || cmd.IsSet(CacheGCFlag.Name) {
		cfg.TrieCache = int(cmd.Int(CacheFlag.Name) * cmd.Int(CacheGCFlag.Name) / 100)
	}
	if cmd.IsSet(MinerThreadsFlag.Name) {
		cfg.MinerThreads = int(cmd.Int(MinerThreadsFlag.Name))
	}
	if cmd.IsSet(WorkingDirectoryFlag.Name) {
		Fatalf("Option %q is not supported", WorkingDirectoryFlag.Name)
		//cfg.WorkingDirectory = cmd.String(WorkingDirectoryFlag.Name)
	}
	if cmd.IsSet(JavascriptDirectoryFlag.Name) && (cmd.Name == "console" || cmd.Name == "attach") {
		cfg.JavascriptDirectory = cmd.String(JavascriptDirectoryFlag.Name)
	}
	if cmd.IsSet(ExtraDataFlag.Name) {
		cfg.ExtraData = []byte(cmd.String(ExtraDataFlag.Name))
	}
	if cmd.IsSet(GasPriceFlag.Name) {
		cfg.GasPrice = GlobalBig(cmd, GasPriceFlag.Name)
	}
	if cmd.IsSet(VMEnableDebugFlag.Name) {
		// TODO(fjl): force-enable this in --dev mode
		cfg.EnablePreimageRecording = cmd.Bool(VMEnableDebugFlag.Name)
	}

	if cmd.Bool(DeveloperFlag.Name) {
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
			cfg.Genesis = core.DeveloperGenesisBlock(uint64(cmd.Int(DeveloperPeriodFlag.Name)), developer.Address)
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
	case cmd.IsSet(ChainFlag.Name):
		chaincfg = params.GetChainConfig(cmd.String(ChainFlag.Name))
		if chaincfg == nil {
			Fatalf("invalid chain name: %q", cmd.String(ChainFlag.Name))
		}
		cfg.Genesis = core.DefaultGenesisByName(cmd.String(ChainFlag.Name))
	default:
		chaincfg = params.MainnetChainConfig
		cfg.Genesis = core.DefaultGenesisByName("aqua")
	}
	cfg.ChainId = chaincfg.ChainId.Uint64()

	// TODO(fjl): move trie cache generations into config
	if gen := cmd.Int(TrieCacheGenFlag.Name); gen > 0 {
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

// SetupNetwork configures the system for either the main net or some test network.
func SetupNetworkGasLimit(cmd *cli.Command) {
	// TODO(fjl): move target gas limit into config
	params.TargetGasLimit = cmd.Uint(TargetGasLimitFlag.Name)
}

// MakeChainDatabase open an LevelDB using the flags passed to the client and will hard crash if it fails.
func MakeChainDatabase(cmd *cli.Command, stack *node.Node) aquadb.Database {
	var (
		cache   = int(cmd.Int(CacheFlag.Name) * cmd.Int(CacheDatabaseFlag.Name) / 100)
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
	chain := cmd.String(ChainFlag.Name)
	if chain != "" {
		return core.DefaultGenesisByName(chain)
	}
	return core.DefaultGenesisBlock()
}
func GenesisByChain(chain string) *core.Genesis {
	return core.DefaultGenesisByName(chain)
}

func MakeConsensusEngine(cmd *cli.Command, stack *node.Node) consensus.Engine {
	if cmd.Bool(FakePoWFlag.Name) {
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

	if gcmode := cmd.String(GCModeFlag.Name); gcmode != "full" && gcmode != "archive" {
		Fatalf("--%s must be either 'full' or 'archive'", GCModeFlag.Name)
	}
	cache := &core.CacheConfig{
		Disabled:      cmd.String(GCModeFlag.Name) == "archive",
		TrieNodeLimit: aqua.DefaultConfig.TrieCache,
		TrieTimeLimit: aqua.DefaultConfig.TrieTimeout,
	}
	if cmd.IsSet(CacheFlag.Name) || cmd.IsSet(CacheGCFlag.Name) {
		cache.TrieNodeLimit = int(cmd.Int(CacheFlag.Name) * cmd.Int(CacheGCFlag.Name) / 100)
	}
	vmcfg := vm.Config{EnablePreimageRecording: cmd.Bool(VMEnableDebugFlag.Name)}
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
	if cmd.String(PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := cmd.String(JavascriptDirectoryFlag.Name)
	for _, file := range strings.Split(cmd.String(PreloadJSFlag.Name), ",") {
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
	for _, c := range cmd.Commands {
		cmdmap[c.Name] = c.Name
		log.Warn("migrating command", "name", c.Name, "flags", c.Flags)
	}
	for _, name := range cmd.FlagNames() {
		if cmd.Root().IsSet(name) {
			cmd.Set(name, cmd.Root().String(name))
		}
	}
	log.Info("running migrated action", "name", cmd.Name)
	return m.Action(ctx, cmd)
}
