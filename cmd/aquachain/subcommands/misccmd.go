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
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/aquaflags"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/buildinfo"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/consensus/aquahash/ethashdag"
	"gitlab.com/aquachain/aquachain/params"
)

var (
	makecacheCommand = &cli.Command{
		Action:    MigrateFlags(makecache),
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
	makedagCommand = &cli.Command{
		Action:    MigrateFlags(makedag),
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
	versionCommand = &cli.Command{
		Action:    MigrateFlags(printVersion),
		Name:      "version",
		Usage:     "Print version numbers",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
		Description: `
The output of this command is supposed to be machine-readable.
`,
	}
	licenseCommand = &cli.Command{
		Action:    MigrateFlags(license),
		Name:      "license",
		Usage:     "Display license information",
		ArgsUsage: " ",
		Category:  "MISCELLANEOUS COMMANDS",
	}
)

// makecache generates an aquahash verification cache into the provided folder.
func makecache(_ context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		utils.Fatalf(`Usage: aquachain makecache <block number> <outputdir>`)
	}
	block, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		utils.Fatalf("Invalid block number: %v", err)
	}
	ethashdag.MakeCache(block, args[1])
	return nil
}

// makedag generates an aquahash mining DAG into the provided folder.
func makedag(_ context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) != 2 {
		utils.Fatalf(`Usage: aquachain makedag <block number> <outputdir>`)
	}
	block, err := strconv.ParseUint(args[0], 0, 64)
	if err != nil {
		utils.Fatalf("Invalid block number: %v", err)
	}
	ethashdag.MakeDataset(block, args[1])

	return nil
}

func printVersion(_ context.Context, cmd *cli.Command) error {
	fmt.Println(strings.Title(clientIdentifier), params.Version)
	if gitCommit != "" {
		fmt.Println("Git Commit:", gitCommit)
	}
	if gitTag != "" {
		fmt.Println("Git Tag:", gitTag)
	}
	if buildDate != "" {
		ts, err := strconv.Atoi(buildDate)
		if err == nil {
			fmt.Printf("Build Date: %s UTC\n", time.Unix(int64(ts), 0).UTC().Format(time.ANSIC))
		} else {
			fmt.Printf("WARN: tried to get date, but got err: %v\n", err)
		}
	}

	// chaincfg := params.MainnetChainConfig

	chainName := cmd.String(aquaflags.ChainFlag.Name)
	chaincfg := params.GetChainConfig(chainName)
	if chaincfg == nil {
		utils.Fatalf("invalid chain name: %q, try one of %q", chainName, params.ValidChainNames())
	}

	// set hardfork params for printing
	// utils.SetFParams(ctx, chaincfg)

	fmt.Println("Architecture:", runtime.GOARCH)
	fmt.Printf("Pure Go: %v\n", !buildinfo.CGO)
	fmt.Println("Go Version:", runtime.Version())
	fmt.Println("Operating System:", runtime.GOOS)
	fmt.Printf("Build Tags: %q\n", params.BuildTags())
	fmt.Println("Protocol Versions:", aqua.ProtocolVersions)
	//
	fmt.Printf("Chain Selected: %s\n", chaincfg.Name())
	fmt.Printf("Chain Id:   %d\n", chaincfg.ChainId.Uint64())
	fmt.Printf("Chain Config: {%s}\n", chaincfg.StringNoChainId())
	fmt.Printf("Chain HF Map: {%s}\n", chaincfg.HF.String())
	fmt.Printf("Consensus Engine: %s\n", chaincfg.EngineName())
	return nil
}

func license(context.Context, *cli.Command) error {
	fmt.Println(`Aquachain is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Aquachain is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with aquachain. If not, see <http://www.gnu.org/licenses/>.`)
	return nil
}
