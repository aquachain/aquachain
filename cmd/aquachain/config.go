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
	"context"
	"io"
	"os"

	cli "github.com/urfave/cli/v3"

	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/toml"
	"gitlab.com/aquachain/aquachain/node"
)

var (
	dumpConfigCommand = &cli.Command{
		Action:      utils.MigrateFlags(dumpConfig),
		Name:        "dumpconfig",
		Usage:       "Show configuration values",
		ArgsUsage:   "",
		Flags:       append(nodeFlags, rpcFlags...),
		Category:    "MISCELLANEOUS COMMANDS",
		Description: `The dumpconfig command shows configuration values.`,
	}
)

// // These settings ensure that TOML keys use the same names as Go struct fields.
// var tomlSettings = toml.Config{
// 	NormFieldName: func(rt reflect.Type, key string) string {
// 		return key
// 	},
// 	FieldToKey: func(rt reflect.Type, field string) string {
// 		return field
// 	},
// 	MissingField: func(rt reflect.Type, field string) error {
// 		link := ""
// 		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
// 			link = fmt.Sprintf(", see https://pkg.go.dev/%s#%s for available fields", rt.PkgPath(), rt.Name())
// 		}
// 		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
// 	},
// }

func closeMain() {
	log.Warn("got closemain signal")
	maincancel()
}
func makeFullNode(ctx context.Context, cmd *cli.Command) *node.Node {
	stack, cfg := utils.MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, closeMain)
	utils.RegisterAquaService(mainctx, stack, &cfg.Aqua, cfg.Node.Name)

	// Add the Aquachain Stats daemon if requested.
	if cfg.Aquastats.URL != "" {
		utils.RegisterAquaStatsService(stack, cfg.Aquastats.URL)
	}
	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(ctx context.Context, cmd *cli.Command) error {
	_, cfg := utils.MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, closeMain, utils.NoPreviousConfig)
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
