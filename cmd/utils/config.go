package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/naoina/toml"
	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/params"
)

func LoadConfigFromFile(file string, cfg *AquachainConfig) error {
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
	// TODO re-tilde on save
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

type cfgopt int

const (
	NoPreviousConfig cfgopt = 1
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

func MakeConfigNode(cmd *cli.Command, gitCommit string, clientIdentifier string, s ...cfgopt) (*node.Node, *AquachainConfig) {
	// Load defaults.
	useprev := true
	for _, v := range s {
		if v == NoPreviousConfig { // so dumpconfig doesn't read auto-config without '-config <name>' flag
			useprev = false
		}
	}
	cfgptr := Mkconfig(cmd, useprev, gitCommit, clientIdentifier)
	// Apply flags.
	if err := SetNodeConfig(cmd, &cfgptr.Node); err != nil {
		Fatalf("Fatal: could not set node config %+v", err)
	}
	stack, err := node.New(&cfgptr.Node)
	if err != nil {
		Fatalf("Failed to create the protocol stack: %v", err)
	}

	SetAquaConfig(cmd, stack, &cfgptr.Aqua)
	if cmd.IsSet(AquaStatsURLFlag.Name) {
		cfgptr.Aquastats.URL = cmd.String(AquaStatsURLFlag.Name)
	}

	return stack, cfgptr
}

func DefaultNodeConfig(gitCommit, clientIdentifier string) node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "aqua")
	cfg.WSModules = append(cfg.WSModules, "aqua")
	cfg.IPCPath = "aquachain.ipc"
	return cfg
}

var ConfigFileFlag = &cli.StringFlag{
	Name:  "config",
	Usage: "TOML configuration file. NEW: In case of multiple instances, use -config=none to disable auto-reading available config files",
}

func Mkconfig(cmd *cli.Command, checkDefaultConfigFiles bool, gitCommit, clientIdentifier string) *AquachainConfig {
	cfgptr := &AquachainConfig{
		Aqua: aqua.DefaultConfig,
		Node: DefaultNodeConfig(gitCommit, clientIdentifier),
	}
	// Load config file.
	file := cmd.String(ConfigFileFlag.Name)
	switch {
	default:
		if err := LoadConfigFromFile(file, cfgptr); err != nil {
			Fatalf("error loading config file: %v", err)
		}
		log.Info("Loaded config", "file", file)
	case file == "" && !checkDefaultConfigFiles || file == "none":
		// default config, flags only
	case file == "" && checkDefaultConfigFiles: // find config if exists in working-directory, ~/.aquachain/aquachain.toml, or /etc/aquachain/aquachain.toml
		var userdatadir string
		chainName := cmd.String(ChainFlag.Name)
		chainCfg := params.GetChainConfig(chainName)
		if chainCfg == nil {
			Fatalf("invalid chain name: %q, try one of %q", chainName, params.ValidChainNames())
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
			if err := LoadConfigFromFile(file, cfgptr); err == nil {
				log.Info("Loaded config", "file", file)
				break
			} else if !errors.Is(err, fs.ErrNotExist) { // error loading an existing config file
				Fatalf("error loading config file: %v", err)
			}
		}
	}

	return cfgptr
}
