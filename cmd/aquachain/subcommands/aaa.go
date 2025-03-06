package subcommands

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/aquaflags"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/buildinfo"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/mainctxs"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/toml"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/aquaclient"
)

var mainctx, maincancel = mainctxs.Main(), mainctxs.MainCancelCause()

var gitCommit, buildDate, gitTag, clientIdentifier string

func SetBuildInfo(commit, date, tag string, clientIdentifier0 string) {
	gitCommit = commit
	buildDate = date
	gitTag = tag
	clientIdentifier = clientIdentifier0
	buildinfo.SetBuildInfo(buildinfo.BuildInfo{
		GitCommit: commit,
		BuildDate: date,
		GitTag:    tag,
		BuildTags: "",
	})
}

func Subcommands() []*cli.Command {
	return []*cli.Command{
		echoCommand,
		initCommand,
		importCommand,
		exportCommand,
		copydbCommand,
		removedbCommand,
		dumpCommand,
		// See monitorcmd.go:
		//monitorCommand,
		// See accountcmd.go:
		accountCommand,

		// See walletcmd_lite.go
		paperCommand,
		// See consolecmd.go:
		consoleCommand,
		daemonCommand, // previously default
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		makecacheCommand,
		makedagCommand,
		versionCommand,
		bugCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
	}
}

func SubcommandByName(s string) *cli.Command {
	for _, c := range Subcommands() {
		if c.Name == s {
			return c
		}
	}
	return nil
}

var dumpConfigCommand = &cli.Command{
	Action:      MigrateFlags(dumpConfig),
	Name:        "dumpconfig",
	Usage:       "Show configuration values",
	ArgsUsage:   "",
	Flags:       append(nodeFlags, rpcFlags...),
	Category:    "MISCELLANEOUS COMMANDS",
	Description: `The dumpconfig command shows configuration values.`,
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx context.Context, cmd *cli.Command) error {
	var opts []Cfgopt
	if cmd.String("config") == "none" {
		opts = append(opts, NoPreviousConfig)
	}
	_, cfg := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, maincancel, opts...)
	comment := ""

	if cfg.Aqua.Genesis != nil {
		cfg.Aqua.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := toml.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}

var StartNodeCommand = startNode

func MakeFullNode(ctx context.Context, cmd *cli.Command) *node.Node {
	stack, cfg := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, maincancel)
	RegisterAquaService(mainctx, stack, cfg.Aqua, cfg.Node.NodeName())

	// Add the Aquachain Stats daemon if requested.
	if cfg.Aquastats.URL != "" {
		RegisterAquaStatsService(stack, cfg.Aquastats.URL)
	}
	return stack
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx context.Context, cmd *cli.Command, stack *node.Node) {
	unlocks := strings.Split(strings.TrimSpace(cmd.String(aquaflags.UnlockedAccountFlag.Name)), ",")
	if len(unlocks) == 1 && unlocks[0] == "" {
		unlocks = []string{} // TODO?
	}
	if len(unlocks) > 0 && stack.Config().NoKeys {
		Fatalf("Unlocking accounts is not supported with NO_KEYS mode")
	}
	if !stack.Config().NoKeys {
		for _, v := range unlocks {
			log.Info("Unlocking account", "account", v)
		}
		if len(unlocks) > 0 && unlocks[0] != "" {
			log.Warn("Unlocking account", "unlocks", unlocks)
			passwords := MakePasswordList(cmd)
			if len(passwords) == 0 && cmd.IsSet(aquaflags.PasswordFileFlag.Name) && cmd.String(aquaflags.PasswordFileFlag.Name) == "" {
				// empty password "" means no password
				passwords = append(passwords, "")
			}
			// Unlock any account specifically requested
			ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			for i, account := range unlocks {
				if trimmed := strings.TrimSpace(account); trimmed != "" {
					unlockAccount(cmd, ks, trimmed, i, passwords)
				}
			}
		}
	}
	node.DefaultConfig.NoCountdown = node.DefaultConfig.NoCountdown || cmd.Bool(aquaflags.DoitNowFlag.Name)
	// Start up the node itself
	StartNode(ctx, stack)

	// Register wallet event handlers to open and auto-derive wallets
	if !stack.Config().NoKeys && !stack.Config().NoInProc {
		events := make(chan accounts.WalletEvent, 16)
		stack.AccountManager().Subscribe(events)
		log.Info("Starting Account Manager")
		go func() {
			// Create an chain state reader for self-derivation
			rpcClient, err := stack.Attach(ctx, "accountManager")
			if err != nil {
				log.GracefulShutdown(log.Errorf("failed to attach to self: %v", err))
				return
			}
			stateReader := aquaclient.NewClient(rpcClient)
			defer rpcClient.Close()
			// Open any wallets already attached
			for _, wallet := range stack.AccountManager().Wallets() {
				if err := wallet.Open(""); err != nil {
					log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
				}
			}
			// Listen for wallet event till termination
			for event := range events {
				switch event.Kind {
				case accounts.WalletArrived:
					if err := event.Wallet.Open(""); err != nil {
						log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
					}
				case accounts.WalletOpened:
					status, _ := event.Wallet.Status()
					log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

					if event.Wallet.URL().Scheme == "ledger" {
						event.Wallet.SelfDerive(accounts.DefaultLedgerBaseDerivationPath, stateReader)
					} else {
						event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
					}

				case accounts.WalletDropped:
					log.Info("Old wallet dropped", "url", event.Wallet.URL())
					event.Wallet.Close()
				}
			}
		}()
	}
	// Start auxiliary services if enabled
	if cmd.Bool(aquaflags.MiningEnabledFlag.Name) || cmd.Bool(aquaflags.DeveloperFlag.Name) {
		var aquachain *aqua.Aquachain
		if err := stack.Service(&aquachain); err != nil {
			Fatalf("Aquachain service not running: %v", err)
		}
		// Use a reduced number of threads if requested
		if threads := cmd.Int(aquaflags.MinerThreadsFlag.Name); threads > 0 {
			type threaded interface {
				SetThreads(threads int)
			}
			if th, ok := aquachain.Engine().(threaded); ok {
				th.SetThreads(int(threads))
			}
		}
		// Set the gas price to the limits from the CLI and start mining
		if cmd.IsSet(aquaflags.GasPriceFlag.Name) {
			if x := aquaflags.GlobalBig(cmd, aquaflags.GasPriceFlag.Name); x != nil {
				aquachain.TxPool().SetGasPrice(x)
			}
		}
		log.Info("gas price", "min", aquachain.TxPool().GasPrice())
		if err := aquachain.StartMining(true); err != nil {
			Fatalf("Failed to start mining: %v", err)
		}
	}
}
