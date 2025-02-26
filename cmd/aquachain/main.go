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

// aquachain is the official command-line client for Aquachain.
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	logpkg "log"

	"github.com/joho/godotenv"
	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/aquaclient"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/params"
)

const (
	clientIdentifier = "aquachain" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash and timestamp of the release (set via linker flags)
	gitCommit, buildDate, gitTag string
	// The app that holds all commands and flags.
	// app = utils.NewApp(gitCommit, "the aquachain command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
		utils.DoitNowFlag,
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,
		utils.BootnodesFlag,
		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.NoKeysFlag,
		utils.UseUSBFlag,
		utils.AquahashCacheDirFlag,
		utils.AquahashCachesInMemoryFlag,
		utils.AquahashCachesOnDiskFlag,
		utils.AquahashDatasetDirFlag,
		utils.AquahashDatasetsInMemoryFlag,
		utils.AquahashDatasetsOnDiskFlag,
		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,
		utils.FastSyncFlag,
		utils.SyncModeFlag,
		utils.GCModeFlag,
		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheGCFlag,
		utils.TrieCacheGenFlag,
		utils.ListenPortFlag,
		utils.ListenAddrFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,
		utils.AquabaseFlag,
		utils.GasPriceFlag,
		utils.MinerThreadsFlag,
		utils.MiningEnabledFlag,
		utils.TargetGasLimitFlag,
		utils.NATFlag,
		utils.NoDiscoverFlag,
		utils.OfflineFlag,
		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,
		utils.NodeKeyHexFlag,
		utils.DeveloperFlag,
		utils.DeveloperPeriodFlag,
		utils.NetworkEthFlag,
		utils.VMEnableDebugFlag,

		utils.AquaStatsURLFlag,
		utils.MetricsEnabledFlag,
		utils.FakePoWFlag,
		utils.NoCompactionFlag,
		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,
		utils.ExtraDataFlag,
		utils.ConfigFileFlag,
		utils.HF8MainnetFlag,
		utils.ChainFlag,
	}

	rpcFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCUnlockFlag,
		utils.RPCCORSDomainFlag,
		utils.RPCVirtualHostsFlag,
		utils.RPCListenAddrFlag,
		utils.RPCAllowIPFlag,
		utils.RPCBehindProxyFlag,
		utils.RPCPortFlag,
		utils.RPCApiFlag,
		utils.WSEnabledFlag,
		utils.WSListenAddrFlag,
		utils.WSPortFlag,
		utils.WSApiFlag,
		utils.WSAllowedOriginsFlag,
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.AlertModeFlag,
	}
)

func doinit() *cli.Command {
	app := &cli.Command{
		Name:    "aquachain",
		Usage:   "the Aquachain command line interface",
		Version: params.VersionWithCommit(gitCommit),
		Flags:   []cli.Flag{&cli.BoolFlag{Name: "noenv", Usage: "Skip loading existing .env file"}},
		// UsageText: ,
	}
	// Initialize the CLI app and start Aquachain
	app.Action = localConsole // default command is 'console'
	app.HideVersion = true    // we have a command to print the version
	app.Copyright = "Copyright 2018-2025 The Aquachain Authors"

	app.Commands = []*cli.Command{
		// See chaincmd.go:
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

	sort.Sort(cli.FlagsByName(app.Flags))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)

	// func(context.Context, *Command) (context.Context, error)
	app.Before = beforeFunc
	app.After = afterFunc
	return app
}

func afterFunc(context.Context, *cli.Command) error {
	debug.Exit()
	console.Stdin.Close() // Resets terminal mode.
	return nil
}

func beforeFunc(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := debug.Setup(ctx, cmd); err != nil {
		return ctx, err
	}
	if x := cmd.Args().First(); x != "" && x != "daemon" || x != "console" { // is subcommand..
		return ctx, nil
	}
	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)

	utils.SetupNetworkGasLimit(cmd)
	_, autoalertmode := os.LookupEnv("ALERT_PLATFORM")
	if autoalertmode {
		cmd.Set(utils.AlertModeFlag.Name, "true")
	}
	return ctx, nil
}

func main() {
	logpkg.SetFlags(logpkg.Lshortfile)
	noenv := false
	for _, v := range os.Args {
		if strings.Contains(v, "-noenv") {
			noenv = true
		}
	}
	if !noenv {
		godotenv.Load(".env")
	}
	app := doinit()
	if err := app.Run(mainctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// daemonCommand is the main entry point into the system if the 'daemon' subcommand
// is ran. It creates a default node based on the command line arguments
// and runs it in blocking mode, waiting for it to be shut down.
func daemonStart(ctx context.Context, cmd *cli.Command) error {
	node := makeFullNode(ctx, cmd)
	startNode(ctx, cmd, node)
	node.Wait()
	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx context.Context, cmd *cli.Command, stack *node.Node) {
	unlocks := strings.Split(cmd.String(utils.UnlockedAccountFlag.Name), ",")
	if len(unlocks) > 0 && stack.Config().NoKeys {
		utils.Fatalf("Unlocking accounts is not supported with --%s", utils.NoKeysFlag.Name)
	}
	if !stack.Config().NoKeys {
		if len(unlocks) > 0 && unlocks[0] != "" {
			log.Warn("Unlocking account", "unlocks", unlocks)
			passwords := utils.MakePasswordList(cmd)
			if len(passwords) == 0 && cmd.IsSet(utils.PasswordFileFlag.Name) && cmd.String(utils.PasswordFileFlag.Name) == "" {
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
	ctx = context.WithValue(ctx, "doitnow", cmd.Bool(utils.DoitNowFlag.Name)) // TODO
	// Start up the node itself
	utils.StartNode(ctx, stack)

	// Register wallet event handlers to open and auto-derive wallets
	if !stack.Config().NoKeys {
		events := make(chan accounts.WalletEvent, 16)
		stack.AccountManager().Subscribe(events)
		log.Info("Starting Account Manager")
		go func() {
			// Create an chain state reader for self-derivation
			rpcClient, err := stack.Attach("accountManager")
			if err != nil {
				utils.Fatalf("Failed to attach to self: %v", err)
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
	if cmd.Bool(utils.MiningEnabledFlag.Name) || cmd.Bool(utils.DeveloperFlag.Name) {
		var aquachain *aqua.Aquachain
		if err := stack.Service(&aquachain); err != nil {
			utils.Fatalf("Aquachain service not running: %v", err)
		}
		// Use a reduced number of threads if requested
		if threads := cmd.Int(utils.MinerThreadsFlag.Name); threads > 0 {
			type threaded interface {
				SetThreads(threads int)
			}
			if th, ok := aquachain.Engine().(threaded); ok {
				th.SetThreads(int(threads))
			}
		}
		// Set the gas price to the limits from the CLI and start mining
		if cmd.IsSet(utils.GasPriceFlag.Name) {
			if x := utils.GlobalBig(cmd, utils.GasPriceFlag.Name); x != nil {
				aquachain.TxPool().SetGasPrice(x)
			}
		}
		log.Info("gas price", "min", aquachain.TxPool().GasPrice())
		if err := aquachain.StartMining(true); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}
}
