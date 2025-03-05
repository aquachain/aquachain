package aqua_test

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/config"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/toml"
)

func init() {
	log.ResetForTesting()
}

type Config = config.Aquaconfig

func TestConfigEmpty(t *testing.T) {
	var cfg Config
	got, err := toml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}
func TestConfigDefault(t *testing.T) {
	var cfg *Config = aqua.NewDefaultConfig()
	got, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}
func TestConfigDefaultMainnet(t *testing.T) {
	var cfg0 *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "aquachain")
	got, err := toml.Marshal(cfg0)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}

// instead of 0x0000000 address, it should be a empty quoted string
func TestConfigDefaultEmptyCoinbase(t *testing.T) {
	var cfg0 *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "nonempty")
	// println("node name:", cfg0.Node.NodeName())
	cfg0.Aqua.Aquabase = common.Address{}
	got, err := toml.Marshal(cfg0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "0x0000000000000000000000000000000000000000") {
		t.Fatal("found zero Aquabase")
	}
	println(string(got))
}

func TestConfigUnmarshalPartial(t *testing.T) {
	tomlStr := `
	[Aqua]
	ChainId = 12345
	[Node]
	UserIdent = "Foo"
	`

	var mainnetcfg *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "aquachain")
	// compare with mainnetcfg after making the exact same changes
	mainnetcfg.Aqua.ChainId = 12345
	mainnetcfg.Node.UserIdent = "Foo"
	var cfg0new *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "aquachain")
	var cfg1copy *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "aquachain").Copy()

	for _, cfg := range []*utils.AquachainConfig{cfg0new, cfg1copy} {
		buf := strings.NewReader(tomlStr)
		if _, err := toml.NewDecoder(buf).Decode(cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Aqua.ChainId != 12345 {
			t.Fatal("ChainId not set from given toml")
		}
		if !cfg.Aqua.NoPruning {
			t.Fatalf("NoPruning not set from default config")
		}

		if l1, l2 := len(cfg.Aqua.ExtraData), len(mainnetcfg.Aqua.ExtraData); l1 != l2 {
			t.Fatalf("len(cfg.Aqua.ExtraData) != len(mainnetcfg.Aqua.ExtraData): %d != %d", l1, l2)
		}

		if !reflect.DeepEqual(&cfg, &mainnetcfg) {
			got1, err := toml.Marshal(&cfg)
			if err != nil {
				t.Fatal(err)
			}
			got2, err := toml.Marshal(mainnetcfg)
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(got1, got2) {
				return // false positive from reflect.DeepEqual
			}
			// try and show exact diff
			fmt.Fprintf(os.Stderr,
				"\n\ncfg1: %#v\n\ncfg0: %#v\n",
				cfg.Aqua, mainnetcfg.Aqua)
			t.Fatalf("cfg != mainnetcfg\n%s\n%s", got1, got2)
		}
	}
}
