// all the flags
package aquaflags

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/core/state"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/params"
)

// type DirectoryFlag = utils.DirectoryFlag // TODO remove these type aliases
// type DirectoryString = utils.DirectoryString

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
	// DataDirFlag = &cli.StringFlag{
	// 	Name:  "datadir",
	// 	Usage: "Data directory for the databases, IPC socket, and keystore (also see -keystore flag)",
	// 	Value: NewDirectoryString(node.DefaultDatadir()),
	// }
	DataDirFlag = &cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases, IPC socket, and keystore (also see -keystore flag)",
		Value: node.DefaultDatadir(),
		Action: func(ctx context.Context, cmd *cli.Command, v string) error {
			if v == "" {
				return fmt.Errorf("invalid directory: %q", v)
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
	KeyStoreDirFlag = &cli.StringFlag{
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
		Action: func(ctx context.Context, cmd *cli.Command, v bool) error {
			if v {
				cmd.Set("now", "true")
				p2p.NoCountdown = true
				node.NoCountdown = true
			}
			return nil
		},
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
	AquahashCacheDirFlag = &cli.StringFlag{
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
	AquahashDatasetDirFlag = &cli.StringFlag{
		Name:  "aquahash.dagdir",
		Usage: "Directory to store the aquahash mining DAGs (default = inside home folder)",
		Value: aqua.DefaultConfig.Aquahash.DatasetDir,
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
	GasPriceFlag = &cli.UintFlag{
		Name:  "gasprice",
		Usage: "Minimal gas price to accept for mining a transactions",
		Value: aqua.DefaultConfig.GasPrice,
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
		Usage: "for allowing keystore and signing via RPC endpoints. do not use. use a separate signer instance.",
	}
	IPCDisabledFlag = &cli.BoolFlag{
		Name:  "ipcdisable",
		Usage: "Disable the IPC-RPC server",
	}
	IPCPathFlag = &cli.StringFlag{
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
		Usage: "If RPC is behind a reverse proxy. (RPC_BEHIND_PROXY env) Changes the way IP is fetched when comparing to allowed IP addresses",
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
		Usage: "Disables keystore entirely (env: NO_KEYS)",
		Value: sense.EnvBool(sense.Getenv("NO_KEYS")) || sense.EnvBool(sense.Getenv("NOKEYS")), // both just in case
	}
	NoSignFlag = &cli.BoolFlag{
		Name:  "nosign",
		Usage: "Disables all signing via RPC endpoints (env:NO_SIGN) (useful when wallet is unlocked for signing blocks on a public testnet3 server)",
		Value: sense.EnvBool(sense.Getenv("NO_SIGN")) || sense.EnvBool(sense.Getenv("NOSIGN")), // both just in case
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

var (
	// The app that holds all commands and flags.
	// app = NewApp(gitCommit, "the aquachain command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		// DoitNowFlag,
		IdentityFlag,
		UnlockedAccountFlag,
		PasswordFileFlag,
		BootnodesFlag,
		DataDirFlag,
		KeyStoreDirFlag,
		NoKeysFlag,
		UseUSBFlag,
		AquahashCacheDirFlag,
		AquahashCachesInMemoryFlag,
		AquahashCachesOnDiskFlag,
		AquahashDatasetDirFlag,
		AquahashDatasetsInMemoryFlag,
		AquahashDatasetsOnDiskFlag,
		TxPoolNoLocalsFlag,
		TxPoolJournalFlag,
		TxPoolRejournalFlag,
		TxPoolPriceLimitFlag,
		TxPoolPriceBumpFlag,
		TxPoolAccountSlotsFlag,
		TxPoolGlobalSlotsFlag,
		TxPoolAccountQueueFlag,
		TxPoolGlobalQueueFlag,
		TxPoolLifetimeFlag,
		FastSyncFlag,
		SyncModeFlag,
		// GCModeFlag,
		CacheFlag,
		CacheDatabaseFlag,
		CacheGCFlag,
		TrieCacheGenFlag,
		ListenPortFlag,
		ListenAddrFlag,
		MaxPeersFlag,
		MaxPendingPeersFlag,
		AquabaseFlag,
		GasPriceFlag,
		MinerThreadsFlag,
		MiningEnabledFlag,
		TargetGasLimitFlag,
		NATFlag,
		NoDiscoverFlag,
		OfflineFlag,
		NetrestrictFlag,
		NodeKeyFileFlag,
		NodeKeyHexFlag,
		DeveloperFlag,
		DeveloperPeriodFlag,
		NetworkEthFlag,
		VMEnableDebugFlag,

		AquaStatsURLFlag,
		MetricsEnabledFlag,
		FakePoWFlag,
		NoCompactionFlag,
		GpoBlocksFlag,
		GpoPercentileFlag,
		ExtraDataFlag,
		// ConfigFileFlag,
		HF8MainnetFlag,
		// ChainFlag,
	}

	rpcFlags = []cli.Flag{
		RPCEnabledFlag,
		RPCUnlockFlag,
		RPCCORSDomainFlag,
		RPCVirtualHostsFlag,
		RPCListenAddrFlag,
		RPCAllowIPFlag,
		RPCBehindProxyFlag,
		RPCPortFlag,
		RPCApiFlag,
		WSEnabledFlag,
		WSListenAddrFlag,
		WSPortFlag,
		WSApiFlag,
		WSAllowedOriginsFlag,
		IPCDisabledFlag,
		IPCPathFlag,
		AlertModeFlag,
	}
)

var NoEnvFlag = &cli.BoolFlag{Name: "noenv", Usage: "Skip loading existing .env file"}

var (
	SocksClientFlag = &cli.StringFlag{
		Name:  "socks",
		Value: "",
		Usage: "SOCKS5 proxy for outgoing RPC connections (eg: -socks socks5h://localhost:1080)",
	}
	consoleFlags = []cli.Flag{JavascriptDirectoryFlag, ExecFlag, PreloadJSFlag, SocksClientFlag}
	daemonFlags  = append(nodeFlags, rpcFlags...)
)

var ConfigFileFlag = &cli.StringFlag{
	Name:  "config",
	Usage: "TOML configuration file. NEW: In case of multiple instances, use -config=none to disable auto-reading available config files",
}

var (
	NodeFlags    = nodeFlags
	RPCFlags     = rpcFlags
	ConsoleFlags = consoleFlags
	DaemonFlags  = daemonFlags
)
