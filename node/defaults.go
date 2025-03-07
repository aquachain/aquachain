// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package node

import (
	"fmt"
	"os/user"
	"path/filepath"
	"runtime"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/params"
)

const (
	DefaultHTTPHost = "127.0.0.1" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 8543        // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "127.0.0.1" // Default host interface for the websocket RPC server
	DefaultWSPort   = 8544        // Default TCP port for the websocket RPC server
)

// DefaultConfig contains reasonable default settings.
// var DefaultConfig = NewDefaultConfig()

func NewDefaultConfig() *Config {
	datadir := defaultDataDir()
	x := &Config{
		Name:        "", // must be set before GetNodeName
		DataDir:     datadir,
		HTTPPort:    DefaultHTTPPort,
		HTTPModules: []string{"aqua", "eth", "net", "web3"},
		WSPort:      DefaultWSPort,
		WSModules:   []string{"aqua", "eth", "net", "web3"},
		P2P: &p2p.Config{
			ListenAddr: "0.0.0.0:21303", // tcp+udp, ipv4 only
			MaxPeers:   20,
			NAT:        "none", // none
		},
		RPCBehindProxy: sense.EnvBool("RPC_BEHIND_PROXY"),
		UserIdent:      sense.Getenv("AQUA_USERIDENT"),
		HTTPHost:       "",
		WSHost:         "",
		RPCNoSign:      sense.EnvBool("NO_SIGN"), // doesnt do anything here. something needs to read it
		NoKeys:         sense.EnvBool("NO_KEYS"), // doesnt do anything here. something needs to read it
		NoCountdown:    sense.EnvBool("NO_COUNTDOWN"),
		KeyStoreDir:    sense.Getenv("AQUA_KEYSTORE_DIR"),
	}
	return x
}

var _cacheddefaultdatadir string = defaultDataDir()

func DefaultDatadir() string {
	return _cacheddefaultdatadir
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func defaultDataDir() string {
	// first, try AQUA_DATADIR env
	if e := sense.Getenv("AQUA_DATADIR"); e != "" {
		return e
	}
	// Try to place the data folder in the user's home dir
	home := homeDir()
	switch {
	case home == "":
		// As we cannot guess a stable location, return empty and handle later
		log.Error("can't determine home directory to place aquachain data dir")
		log.GracefulShutdown(fmt.Errorf("can't determine home directory to place aquachain data dir"))
		return ""
	case runtime.GOOS == "windows":
		return filepath.Join(home, "AppData", "Roaming", "Aquachain")
	case runtime.GOOS == "darwin":
		return filepath.Join(home, "Library", "Aquachain")
	default:
		return filepath.Join(home, ".aquachain")
	}
}

func DefaultDatadirByChain(cfg *params.ChainConfig) string {
	if cfg == nil {
		log.GracefulShutdownf("selecting default mainnet dir for nil chain config")
	}
	def := defaultDataDir() // eg: ~/.aquachain
	if cfg == params.MainnetChainConfig {
		return def
	}
	name := cfg.Name()
	if name == "" {
		panic("chain config has no name")
	}
	return filepath.Join(def, name) // eg: ~/.aquachain/testnet3
}

func homeDir() string {
	// use HOME first in case user wants to override
	if home := sense.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
