package aqua_test

import (
	"strings"
	"testing"

	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/toml"
)

type Config = aqua.Config

var DefaultConfig = aqua.DefaultConfig

func TestConfigEmpty(t *testing.T) {
	var cfg Config
	got, err := toml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}
func TestConfigDefault(t *testing.T) {
	var cfg *Config = DefaultConfig
	got, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}
func TestConfigDefaultMainnet(t *testing.T) {
	var cfg0 *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "aquachain")
	cfg := cfg0.Aqua
	got, err := toml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	println(string(got))
}

func TestConfigDefaultEmptyCoinbase(t *testing.T) {
	var cfg0 *utils.AquachainConfig = utils.Mkconfig("aqua", "", false, "100aa3", "")
	cfg := cfg0.Aqua
	cfg.Aquabase = common.Address{}
	got, err := toml.Marshal(&cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "0x0000000000000000000000000000000000000000") {
		t.Fatal("found zero Aquabase")
	}
	println(string(got))
}
