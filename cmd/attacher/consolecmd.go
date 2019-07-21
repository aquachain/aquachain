// Copyright 2016 The aquachain Authors
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

package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/aerth/tgun"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/params"
	rpc "gitlab.com/aquachain/aquachain/rpc/rpcclient"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	socksFlag = &cli.StringFlag{
		Name:  "socks",
		Value: "",
		Usage: "SOCKS Proxy to use for remote RPC calls (attach subcommand)",
	}

	consoleFlags = []cli.Flag{utils.JSpathFlag, utils.ExecFlag, utils.PreloadJSFlag}

	attachCommand = cli.Command{
		Action:    utils.MigrateFlags(remoteConsole),
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     append([]cli.Flag{utils.DataDirFlag, socksFlag}, consoleFlags...),
		Category:  "CONSOLE COMMANDS",
		Description: `
The AquaChain console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://gitlab.com/aquachain/aquachain/wiki/JavaScript-Console.
This command allows to open a console on a running aquachain node.`,
	}
)

// remoteConsole will connect to a remote aquachain instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running aquachain instance and start the JavaScript console
	endpoint := ctx.Args().First()
	datadir := node.DefaultDataDir()
	if endpoint == "" {
		path := datadir
		if ctx.GlobalIsSet(utils.DataDirFlag.Name) {
			path = ctx.GlobalString(utils.DataDirFlag.Name)
		}
		if path != "" {
			if ctx.GlobalBool(utils.TestnetFlag.Name) {
				path = filepath.Join(path, "testnet")
			} else if ctx.GlobalBool(utils.Testnet2Flag.Name) {
				path = filepath.Join(path, "testnet2")
			} else if ctx.GlobalBool(utils.NetworkEthFlag.Name) {
				path = filepath.Join(path, "ethereum")
			} else if ctx.GlobalBool(utils.DeveloperFlag.Name) {
				path = filepath.Join(path, "develop")
			}
		}
		endpoint = fmt.Sprintf("%s/aquachain.ipc", path)
	}
	socks := ctx.GlobalString("socks")
	client, err := dialRPC(endpoint, socks)
	if err != nil {
		utils.Fatalf("Unable to attach to remote aquachain: %v", err)
	}
	config := console.Config{
		DataDir: datadir,
		DocRoot: ctx.GlobalString(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

// dialRPC returns a RPC client which connects to the given endpoint.
// The check for empty endpoint implements the defaulting logic
// for "aquachain attach" and "aquachain monitor" with no argument.
func dialRPC(endpoint string, socks string) (*rpc.Client, error) {
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
