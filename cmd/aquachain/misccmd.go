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

package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/params"
	"github.com/urfave/cli"
)

var (
	makecacheCommand = cli.Command{
		Action:    utils.MigrateFlags(makecache),
		Name:      "makecache",
		Usage:     "Generate aquahash verification cache (for testing)",
		ArgsUsage: "<blockNum> <outputDir>",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The makecache command generates an aquahash cache in <outputDir>.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
	}
	makedagCommand = cli.Command{
		Action:    utils.MigrateFlags(makedag),
		Name:      "makedag",
		Usage:     "Generate aquahash mining DAG (for testing)",
		ArgsUsage: "<blockNum> <outputDir>",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The makedag command generates an aquahash DAG in <outputDir>.

This command exists to support the system testing project.
Regular users do not need to execute it.
`,
	}
	versionCommand = cli.Command{
		Action:    utils.MigrateFlags(version),
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
	licenseCommand = cli.Command{
		Action:    utils.MigrateFlags(license),
		Name:      "license",
		Usage:     "Display license information",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
	}
)

// makecache generates an aquahash verification cache into the provided folder.
func makecache(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		utils.Fatalf(`Usage: aquachain makecache <block number> <outputdir>`)
	}
	block, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		utils.Fatalf("Invalid block number: %v", err)
	}
	aquahash.MakeCache(block, args[1])

	return nil
}

// makedag generates an aquahash mining DAG into the provided folder.
func makedag(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 2 {
		utils.Fatalf(`Usage: aquachain makedag <block number> <outputdir>`)
	}
	block, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		utils.Fatalf("Invalid block number: %v", err)
	}
	aquahash.MakeDataset(block, args[1])

	return nil
}

func version(ctx *cli.Context) error {
	fmt.Println(strings.Title(clientIdentifier), params.Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if buildDate != "" {
		ts, err := strconv.Atoi(buildDate)
		if err == nil {
			fmt.Printf("Build Date: %s UTC\n", time.Unix(int64(ts), 0).UTC().Format(time.ANSIC))
		} else {
			fmt.Printf("WARN: tried to get date, but got err: %v\n", err)
		}
	}

	chaincfg := params.MainnetChainConfig
	if ctx.GlobalBool(utils.TestnetFlag.Name) {
		chaincfg = params.TestnetChainConfig
	} else if ctx.GlobalBool(utils.Testnet2Flag.Name) {
		chaincfg = params.Testnet2ChainConfig
	} else if ctx.GlobalBool(utils.NetworkEthFlag.Name) {
		chaincfg = params.EthnetChainConfig
	} else if ctx.GlobalBool(utils.DeveloperFlag.Name) {
		chaincfg = params.TestChainConfig
	}

	// set hardfork params for printing
	utils.SetHardforkParams(ctx, chaincfg)

	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Println("Protocol Versions:", aqua.ProtocolVersions)
	fmt.Println("Network Id:", aqua.DefaultConfig.NetworkId)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("CGO_ENABLED=%v\n", CGO)
	fmt.Printf("GOPATH=%s\n", os.Getenv("GOPATH"))
	fmt.Printf("GOROOT=%s\n", runtime.GOROOT())
	fmt.Printf("AQUA Fork Map: %s\n", chaincfg.HF.String())
	fmt.Printf("Chain Config: %s\n", chaincfg)

	return nil
}

func license(_ *cli.Context) error {
	fmt.Println(`AquaChain is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

AquaChain is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with aquachain. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
