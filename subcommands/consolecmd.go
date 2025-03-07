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

package subcommands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/aerth/tgun"
	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/params"
	rpc "gitlab.com/aquachain/aquachain/rpc/rpcclient"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
)

// // ATM the url is left to the user and deployment to
//
//	aquaflags.JSpathFlag = &cli.aquaflags.StringFlag{
//		Name:  "jspath",
//		Usage: "JavaScript root path for `loadScript`",
//		Value: ".",
//	}

var (
	daemonFlags  = aquaflags.DaemonFlags
	consoleFlags = aquaflags.ConsoleFlags
	nodeFlags    = aquaflags.NodeFlags
	rpcFlags     = aquaflags.RPCFlags
)

var (
	daemonCommand = &cli.Command{
		Action:      MigrateFlags(daemonStart),
		Name:        "daemon",
		Flags:       daemonFlags,
		Usage:       "Start a full node",
		Category:    "CONSOLE COMMANDS",
		Description: "",
	}
	consoleCommand = &cli.Command{
		Action: MigrateFlags(localConsole),
		// Action:   localConsole,
		Name:     "console",
		Usage:    "Start an interactive JavaScript environment",
		Flags:    append(consoleFlags, daemonFlags...),
		Category: "CONSOLE COMMANDS",
		Description: `
The Aquachain console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://gitlab.com/aquachain/aquachain/wiki/JavaScript-Console.`,
	}

	attachCommand = &cli.Command{
		Action:    MigrateFlags(remoteConsole),
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     append(consoleFlags, aquaflags.DataDirFlag),
		Category:  "CONSOLE COMMANDS",
		Description: `
The Aquachain console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://gitlab.com/aquachain/aquachain/wiki/JavaScript-Console.
This command allows to open a console on a running aquachain node.`,
	}

	javascriptCommand = &cli.Command{
		Action:    MigrateFlags(ephemeralConsole),
		Name:      "js",
		Usage:     "Execute the specified JavaScript files",
		ArgsUsage: "<jsfile> [jsfile...]",
		Flags:     append(nodeFlags, consoleFlags...),
		Category:  "CONSOLE COMMANDS",
		Description: `
The JavaScript VM exposes a node admin interface as well as the Ðapp
JavaScript API. See https://gitlab.com/aquachain/aquachain/wiki/JavaScript-Console`,
	}
)

// localConsole starts a new aquachain node, attaching a JavaScript console to it at the
// same time.
func localConsole(ctx context.Context, cmd *cli.Command) error {
	// Create and start the node based on the CLI flags
	if first := cmd.Args().First(); first != "" && first[0] != '-' && first != "console" {
		return fmt.Errorf("uhoh: %q got here", first)
	}
	if args := cmd.Args(); args.Len() != 0 && args.First() != "console" {
		return fmt.Errorf("invalid command: %q", args.First())
	}
	nodeserver := MakeFullNode(ctx, cmd)
	if !node.DefaultConfig.NoCountdown && !cmd.Bool("now") && !cmd.Root().Bool("now") {
		for i := 3; i > 0 && ctx.Err() == nil; i-- {
			log.Info("starting in ...", "seconds", i, "bootnodes", len(nodeserver.Config().P2P.BootstrapNodes),
				"static", len(nodeserver.Config().P2P.StaticNodes), "discovery", !nodeserver.Config().P2P.NoDiscovery)
			for i := 0; i < 10 && ctx.Err() == nil; i++ {
				time.Sleep(time.Second / 10)
			}
		}
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}
	startNode(ctx, cmd, nodeserver)
	defer nodeserver.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := nodeserver.Attach(ctx, "localConsole")
	if err != nil {
		return fmt.Errorf("failed to attach to the inproc aquachain: %v", err)
	}
	config := console.Config{
		DataDir:          MakeDataDir(cmd),
		WorkingDirectory: cmd.String(aquaflags.JavascriptDirectoryFlag.Name),
		Client:           client,
		Preload:          MakeConsolePreloads(cmd),
	}

	console, err := console.New(config)
	if err != nil {
		Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	// If only a short execution was requested, evaluate and return
	if script := cmd.String(aquaflags.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}
	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive(mainctx)
	return nil
}

// assumeEndpoint returns the default IPC endpoint for the given chain.
// for 'attach' with no arg
func assumeEndpoint(_ context.Context, cmd *cli.Command) string {

	chaincfg := params.GetChainConfig(cmd.String(aquaflags.ChainFlag.Name))
	defaultpath := node.DefaultDatadirByChain(chaincfg)
	path := defaultpath
	if cmd.Bool(aquaflags.TestnetFlag.Name) {
		path = filepath.Join(path, "testnet")
	} else if cmd.Bool(aquaflags.Testnet2Flag.Name) {
		path = filepath.Join(path, "testnet2")
	} else if cmd.Bool(aquaflags.Testnet3Flag.Name) {
		path = filepath.Join(path, "testnet3")
	} else if cmd.Bool(aquaflags.NetworkEthFlag.Name) {
		path = filepath.Join(path, "ethereum")
	} else if cmd.Bool(aquaflags.DeveloperFlag.Name) {
		path = filepath.Join(path, "develop")
	}

	if cmd.IsSet(aquaflags.DataDirFlag.Name) {
		got := cmd.String(aquaflags.DataDirFlag.Name)
		// handle case where /var/lib/aquachain is passed with testnet and incompatible genesis block
		if got != "" && got != path && got != defaultpath {
			path = got // will be a subdirectory because Joined above
		}
	}
	if path == "" {
		return ""
	}
	return fmt.Sprintf("%s/aquachain.ipc", path)
}

// remoteConsole will connect to a remote aquachain instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx context.Context, cmd *cli.Command) error {
	// Attach to a remotely running aquachain instance and start the JavaScript console
	endpoint := cmd.Args().First()
	if endpoint == "" {
		endpoint = assumeEndpoint(ctx, cmd)
	}
	if endpoint == "" {
		return fmt.Errorf("no endpoint specified")
	}
	socks := cmd.String(aquaflags.SocksClientFlag.Name) // ignored if IPC endpoint is the endpoint, maybe ignored if 127
	clientIdentifier := cmd.String("clientIdentifier")
	client, err := dialRPC(endpoint, socks, clientIdentifier)
	if err != nil {
		Fatalf("Unable to attach to remote aquachain: %v", err)
	}
	client.Name = "remoteConsole"
	config := console.Config{
		DataDir:          MakeDataDir(cmd),
		WorkingDirectory: cmd.String(aquaflags.JavascriptDirectoryFlag.Name),
		Client:           client,
		Preload:          MakeConsolePreloads(cmd),
	}

	console, err := console.New(config)
	if err != nil {
		Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	if script := cmd.String(aquaflags.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive(ctx)

	return nil
}

// dialRPC returns a RPC client which connects to the given endpoint.
// The check for empty endpoint implements the defaulting logic
// for "aquachain attach" and "aquachain monitor" with no argument.
func dialRPC(endpoint string, socks string, clientIdentifier string) (*rpc.Client, error) {
	/* log.Info("Dialing RPC server", "endpoint", endpoint)
	if socks != "" {
		log.Info("+SOCKS5")
	} */
	if endpoint == "" {
		endpoint = node.DefaultIPCEndpoint(clientIdentifier)
	}
	if strings.HasPrefix(endpoint, "http") {
		client := &tgun.Client{
			Proxy: socks,
		}
		httpclient, err := client.HTTPClient()
		if err == nil {
			return rpc.DialHTTPCustom(endpoint, httpclient, map[string]string{"User-Agent": "Aquachain/" + params.Version})
		}
	}
	return rpc.Dial(endpoint)
}

// ephemeralConsole starts a new aquachain node, attaches an ephemeral JavaScript
// console to it, executes each of the files specified as arguments and tears
// everything down.
func ephemeralConsole(ctx context.Context, cmd *cli.Command) error {
	// Create and start the node based on the CLI flags
	node := MakeFullNode(ctx, cmd)
	StartNodeCommand(ctx, cmd, node)
	defer node.Stop()

	// Attach to the newly started node and start the JavaScript console
	client, err := node.Attach(ctx, "ephemeralConsole")
	if err != nil {
		return fmt.Errorf("failed to attach to the inproc aquachain: %v", err)
	}
	config := console.Config{
		DataDir:          MakeDataDir(cmd),
		WorkingDirectory: cmd.String(aquaflags.JavascriptDirectoryFlag.Name),
		Client:           client,
		Preload:          MakeConsolePreloads(cmd),
	}

	console, err := console.New(config)
	if err != nil {
		Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	files := cmd.Args().Slice()

	// Evaluate each of the specified JavaScript files
	for _, file := range files {
		if err = console.ExecuteFile(file); err != nil {
			Fatalf("Failed to execute %s: %v", file, err)
		}
	}
	// Wait for pending callbacks, but stop for Ctrl-C.
	abort := make(chan os.Signal, 1)
	log.Info("dev console waiting for interrupt")
	signal.Notify(abort, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-abort
		os.Exit(0)
	}()
	console.Stop(true)

	return nil
}
