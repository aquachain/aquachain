package utils

import (
	"bufio"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/toml"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/params"
)

func LoadConfigFromFile(file string, cfg *AquachainConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = toml.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// after toml decode, lets expand DataDir (tilde, environmental variables)
	// this keeps config file tidy and sharable.
	// TODO re-tilde on save? or make replacer func
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
// 		err := fmt.Errorf("field '%s' is not defined in %s%s", field, rt.String(), link)
// 		if os.Getenv("TOML_MISSING_FIELD") == "OK" {
// 			log.Warn(err.Error())
// 			return nil
// 		}
// 		// wrong config file, or outdated config file
// 		return err
// 	},
// }

// MakeConfigNode created a Node and Config, and is called by a subcommand at startup.
func MakeConfigNode(ctx context.Context, cmd *cli.Command, gitCommit string, clientIdentifier string, closemain func(error), s ...cfgopt) (*node.Node, *AquachainConfig) {
	// Load defaults.
	useprev := true
	for _, v := range s {
		if v == NoPreviousConfig { // so dumpconfig doesn't read auto-config without '-config <name>' flag
			useprev = false
		}
	}
	// log.Info("Calling MkConfig", "chain", cmd.String(ChainFlag.Name),
	// 	"config", cmd.String(ConfigFileFlag.Name), "useprev", fmt.Sprint(useprev), "gitCommit", gitCommit, "id", clientIdentifier)
	cfgptr := Mkconfig(cmd.String(ChainFlag.Name), cmd.String(ConfigFileFlag.Name), useprev, gitCommit, clientIdentifier)
	// Apply flags.
	if err := SetNodeConfig(cmd, cfgptr.Node); err != nil {
		Fatalf("Fatal: could not set node config %+v", err)
	}
	cfgptr.Node.Context = ctx
	cfgptr.Node.NoInProc = cmd.Name != "" && cmd.Name != "console" && os.Getenv("NO_INPROC") == "1"
	cfgptr.Node.CloseMain = closemain
	stack, err := node.New(cfgptr.Node)
	if err != nil {
		Fatalf("Failed to create the protocol stack: %v", err)
	}

	SetAquaConfig(cmd, stack, cfgptr.Aqua)
	if cmd.IsSet(AquaStatsURLFlag.Name) {
		cfgptr.Aquastats.URL = cmd.String(AquaStatsURLFlag.Name)
	}

	return stack, cfgptr
}

func DefaultNodeConfig(gitCommit, clientIdentifier string) *node.Config {
	if clientIdentifier == "" {
		panic("clientIdentifier must be set")
	}
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	cfg.Version = params.VersionWithCommit(gitCommit)
	cfg.HTTPModules = append(cfg.HTTPModules, "aqua")
	cfg.WSModules = append(cfg.WSModules, "aqua")
	cfg.IPCPath = "aquachain.ipc"
	cfg.P2P.Name = node.GetNodeName(cfg) // cached
	return cfg
}

var ConfigFileFlag = &cli.StringFlag{
	Name:  "config",
	Usage: "TOML configuration file. NEW: In case of multiple instances, use -config=none to disable auto-reading available config files",
}

func Mkconfig(chainName string, configFileOptional string, checkDefaultConfigFiles bool, gitCommit, clientIdentifier string) *AquachainConfig {
	cfgptr := &AquachainConfig{
		Aqua: aqua.DefaultConfig,
		Node: DefaultNodeConfig(gitCommit, clientIdentifier),
	}
	// Load config file.
	file := configFileOptional
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
		chainName := chainName
		chainCfg := params.GetChainConfig(chainName)
		if chainCfg == nil {
			Fatalf("invalid chain name: %q, try one of %q", chainName, params.ValidChainNames())
		}
		var slug string // eg: "_testnet" for testnet, "" for mainnet
		if params.MainnetChainConfig == chainCfg {
			userdatadir = node.DefaultConfig.DataDir
		} else {
			userdatadir = node.DefaultDatadirByChain(chainCfg)
		}
		if chainName != "aquachain" && chainName != "mainnet" && chainName != "aqua" {
			slug = "_" + chainName
		}

		fn := "aquachain" + slug + ".toml" // eg: aquachain_testnet.toml or aquachain.toml
		for _, file := range []string{"./" + fn, filepath.Join(userdatadir, fn), "/etc/aquachain/" + fn} {
			log.Debug("Looking for autoconfig file", "file", file)
			if err := LoadConfigFromFile(file, cfgptr); err == nil {
				log.Info("Loaded autoconfig", "file", file)
				break
			} else if !errors.Is(err, fs.ErrNotExist) { // error loading an existing config file
				Fatalf("error loading autoconfig file: %v", err)
			}
		}
	}

	// set defaults asap
	node.DefaultConfig = cfgptr.Node
	return cfgptr
}

type AquachainConfig = config.AquachainConfigFull
