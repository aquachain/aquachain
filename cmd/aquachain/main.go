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
	"gitlab.com/aquachain/aquachain/cmd/aquachain/aquaflags"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/subcommands"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/params"
)

const (
	clientIdentifier = "aquachain" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash and timestamp of the release (set via linker flags)
	gitCommit, buildDate, gitTag string
)
var this_app *cli.Command

var helpCommand = &cli.Command{
	Name:  "help",
	Usage: "show help",
	Action: func(ctx context.Context, cmd *cli.Command) error {
		cli.ShowAppHelp(cmd)
		os.Exit(1)
		return nil
	},
	UsageText: "aquachain help",
}

func doinit() *cli.Command {
	this_app = &cli.Command{
		Name:    "aquachain",
		Usage:   "the Aquachain command line interface",
		Version: params.VersionWithCommit(gitCommit),
		Flags: append([]cli.Flag{
			aquaflags.NoEnvFlag,
			aquaflags.DoitNowFlag,
			aquaflags.ConfigFileFlag,
			aquaflags.ChainFlag,
			aquaflags.GCModeFlag,
		}, debug.Flags...),
		Suggest: true,
		SuggestCommandFunc: func(commands []*cli.Command, provided string) string {
			s := cli.SuggestCommand(commands, provided)
			// log.Info("running SuggestCommand", "commands", commands, "provided", provided, "suggesting", s)
			if s == provided {
				return s
			}

			println("did you mean:", s)
			os.Exit(1)
			return s
		},
		Before:         beforeFunc,
		After:          afterFunc,
		DefaultCommand: "consoledefault",
		Commands: []*cli.Command{
			// See chaincmd.go:
			helpCommand,
		},
		HideHelpCommand: true,
		HideVersion:     true,
		Copyright:       "Copyright 2018-2025 The Aquachain Authors",
	}
	{
		app := this_app
		// app.Flags = append(app.Flags, debug.Flags...)
		app.Flags = append(app.Flags, aquaflags.NodeFlags...)
		app.Flags = append(app.Flags, aquaflags.RPCFlags...)
		app.Flags = append(app.Flags, aquaflags.ConsoleFlags...)
	}
	sort.Sort((cli.FlagsByName)(this_app.Flags))
	return this_app
}

func afterFunc(context.Context, *cli.Command) error {
	debug.Exit()
	console.Stdin.Close()
	return nil
}

func beforeFunc(ctx context.Context, cmd *cli.Command) (context.Context, error) {
	log.Warn("beforeFunc", "cmd", cmd.Name, "cat", cmd.Category)
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := debug.Setup(ctx, cmd); err != nil {
		return ctx, err
	}
	if x := cmd.Args().First(); x != "" && x != "daemon" || x != "console" { // is subcommand..
		return ctx, nil
	}

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
	if targetGasLimit := cmd.Uint(aquaflags.TargetGasLimitFlag.Name); targetGasLimit > 0 {
		params.TargetGasLimit = targetGasLimit
	}
	_, autoalertmode := os.LookupEnv("ALERT_PLATFORM")
	if autoalertmode {
		cmd.Set(aquaflags.AlertModeFlag.Name, "true")
	}
	return ctx, nil
}

func main() {
	logpkg.SetFlags(logpkg.Lshortfile)
	{
		// check for .env file unless -noenv is in args
		// (before flags are parsed)
		noenv := false
		for _, v := range os.Args {
			if strings.Contains(v, "-noenv") {
				noenv = true
			}
		}
		if !noenv {
			godotenv.Load(".env")
		} else {
			log.Warn("Skipping .env file")
		}
	}
	app := doinit()
	if err := app.Run(mainctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := debug.WaitLoops(time.Second * 2); err != nil {
		log.Warn("waiting for loops", "err", err)
	} else {
		log.Info("graceful shutdown achieved")
	}
}

var startNode = subcommands.StartNodeCommand
var makeFullNode = subcommands.MakeFullNode

var daemonStart = subcommands.DaemonStartCommand
