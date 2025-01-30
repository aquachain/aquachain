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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	cli "github.com/urfave/cli/v3"

	"github.com/naoina/toml"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/params"
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

	configFileFlag = &cli.StringFlag{
		Name:  "config",
		Usage: "TOML configuration file (in case of multiple instances, use -config=none to disable auto-reading available config files)",
	}
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
	MissingField: func(rt reflect.Type, field string) error {
		link := ""
		if unicode.IsUpper(rune(rt.Name()[0])) && rt.PkgPath() != "main" {
			link = fmt.Sprintf(", see https://pkg.go.dev/%s#%s for available fields", rt.PkgPath(), rt.Name())
		}
		return fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
	},
}

type ethstatsConfig struct {
	URL string `toml:",omitempty"`
}

type gethConfig struct {
	Info      any `toml:",omitempty"`
	Aqua      aqua.Config
	Node      node.Config
	Aquastats ethstatsConfig
}

func loadConfig(file string, cfg *gethConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	// after toml decode, lets expand DataDir (tilde, environmental variables)
	// this keeps config file tidy and sharable.
	if err == nil {
		cfg.Node.DataDir = strings.Replace(cfg.Node.DataDir, "~/", "$HOME/", 1)
		cfg.Node.DataDir = os.ExpandEnv(cfg.Node.DataDir)
		cfg.Aqua.Aquahash.DatasetDir = strings.Replace(cfg.Aqua.Aquahash.DatasetDir, "~/", "$HOME/", 1)
		cfg.Aqua.Aquahash.DatasetDir = os.ExpandEnv(cfg.Aqua.Aquahash.DatasetDir)
		cfg.Aqua.Aquahash.CacheDir = strings.Replace(cfg.Aqua.Aquahash.CacheDir, "~/", "$HOME/", 1)
		cfg.Aqua.Aquahash.CacheDir = os.ExpandEnv(cfg.Aqua.Aquahash.CacheDir)
	}
	return err
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "aqua")
	cfg.WSModules = append(cfg.WSModules, "aqua")
	cfg.IPCPath = "aquachain.ipc"
	return cfg
}

func mkconfig(cmd *cli.Command, checkDefaultConfigFiles bool) *gethConfig {
	cfgptr := &gethConfig{
		Aqua: aqua.DefaultConfig,
		Node: defaultNodeConfig(),
	}
	// Load config file.
	file := cmd.String(configFileFlag.Name)
	switch {
	default:
		if err := loadConfig(file, cfgptr); err != nil {
			utils.Fatalf("error loading config file: %v", err)
		}
		log.Info("Loaded config", "file", file)
	case file == "" && !checkDefaultConfigFiles || file == "none":
		// default config, flags only
	case file == "" && checkDefaultConfigFiles: // find config if exists in working-directory, ~/.aquachain/aquachain.toml, or /etc/aquachain/aquachain.toml
		var userdatadir string
		chainName := cmd.String(utils.ChainFlag.Name)
		chainCfg := params.GetChainConfig(chainName)
		if chainCfg == nil {
			utils.Fatalf("invalid chain name: %q, try one of %q", chainName, params.ValidChainNames())
		}
		var slug string // eg: "_testnet" for testnet, "" for mainnet
		if params.MainnetChainConfig == chainCfg {
			userdatadir = node.DefaultDataDir()
		} else {
			userdatadir = filepath.Join(node.DefaultDataDir(), chainName)
		}
		if chainName != "aquachain" && chainName != "mainnet" && chainName != "aqua" {
			slug = "_" + chainName
		}

		fn := "aquachain" + slug + ".toml" // eg: aquachain_testnet.toml or aquachain.toml
		for _, file := range []string{fn, filepath.Join(userdatadir, fn), "/etc/aquachain/" + fn} {
			if err := loadConfig(file, cfgptr); err == nil {
				log.Info("Loaded config", "file", file)
				break
			} else if !errors.Is(err, fs.ErrNotExist) { // error loading an existing config file
				utils.Fatalf("error loading config file: %v", err)
			}
		}
	}

	return cfgptr
}

type cfgopt int

const (
	NoPreviousConfig cfgopt = 1
)

func makeConfigNode(cmd *cli.Command, s ...cfgopt) (*node.Node, *gethConfig) {
	// Load defaults.
	useprev := true
	for _, v := range s {
		if v == NoPreviousConfig {
			useprev = false
		}
	}
	cfgptr := mkconfig(cmd, useprev)
	// Apply flags.
	if err := utils.SetNodeConfig(cmd, &cfgptr.Node); err != nil {
		utils.Fatalf("Fatal: could not set node config %+v", err)
	}
	stack, err := node.New(&cfgptr.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}

	utils.SetAquaConfig(cmd, stack, &cfgptr.Aqua)
	if cmd.IsSet(utils.AquaStatsURLFlag.Name) {
		cfgptr.Aquastats.URL = cmd.String(utils.AquaStatsURLFlag.Name)
	}

	return stack, cfgptr

}

func makeFullNode(cmd *cli.Command) *node.Node {
	stack, cfg := makeConfigNode(cmd)

	utils.RegisterAquaService(stack, &cfg.Aqua)

	// Add the Aquachain Stats daemon if requested.
	if cfg.Aquastats.URL != "" {
		utils.RegisterAquaStatsService(stack, cfg.Aquastats.URL)
	}
	return stack
}

// dumpConfig is the dumpconfig command.
func dumpConfig(_ context.Context, cmd *cli.Command) error {
	_, cfg := makeConfigNode(cmd, NoPreviousConfig)
	comment := ""

	if cfg.Aqua.Genesis != nil {
		cfg.Aqua.Genesis = nil
		comment += "# Note: this config doesn't contain the genesis block.\n\n"
	}

	out, err := tomlSettings.Marshal(&cfg)
	if err != nil {
		return err
	}
	io.WriteString(os.Stdout, comment)
	os.Stdout.Write(out)
	return nil
}
