// Copyright 2019 The aquachain Authors
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
	"fmt"
	"os"
	"sort"

	cli "github.com/urfave/cli"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/opt/console"
)

const (
	clientIdentifier = "aquachain-lite" // Client identifier to advertise over the network
)

var (
	// Git SHA1 commit hash and timestamp of the release (set via linker flags)
	gitCommit, buildDate string
	// The app that holds all commands and flags.
	app = utils.NewApp(gitCommit, "the aquachain command line interface")
)

func init() {
	// Initialize the CLI app and start Aquachain
	//app.Action = app.Usage // default command is 'console'

	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2018-2019 The Aquachain Authors"
	app.Commands = []cli.Command{
		// See walletcmd.go
		paperCommand,
		// See consolecmd.go:
		attachCommand,
		// See misccmd.go:
		versionCommand,
		licenseCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, consoleFlags...)

	app.Before = func(ctx *cli.Context) error {
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
