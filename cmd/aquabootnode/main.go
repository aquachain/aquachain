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

// bootnode runs a bootstrap node for the Aquachain Discovery Protocol.
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/p2p/discover"
	"gitlab.com/aquachain/aquachain/p2p/nat"
	"gitlab.com/aquachain/aquachain/p2p/netutil"
	"gitlab.com/aquachain/aquachain/params"
)

type BootstrapConfig struct {
	ChainId      uint64
	ChainName    string
	Bootnodes    []*discover.Node
	AnnounceAddr *net.UDPAddr
	NetRestrict  netutil.Netlist
	Addr         net.UDPAddr       // listen address, ipv4
	nodekey      *btcec.PrivateKey // (p2p key, not wallet)
	Nat          string            // 'none' or 'upnp' or 'pmp' or 'extip:<IP>'

}

var bootstrapcfg = BootstrapConfig{
	ChainId:      params.MainnetChainConfig.ChainId.Uint64(),
	ChainName:    "aqua",
	Bootnodes:    (discover.BootnodeStringList)(params.MainnetBootnodes).ToDiscoverNodes(),
	AnnounceAddr: nil,
	NetRestrict:  nil,
	Addr:         net.UDPAddr{IP: net.IPv4zero, Port: 21000},
}

func main() {
	flag.Func("addr", "listen address, port implies chainid and chainname unless set explicitly", func(s string) error {
		println("parsing:", s)
		if s == "" {
			return nil
		}
		if strings.HasPrefix(s, ":") {
			s = "0.0.0.0" + s // ipv4 only
		}
		if ip := net.ParseIP(s); ip != nil {
			bootstrapcfg.Addr.IP = ip
		}
		if bootstrapcfg.Addr.IP == nil || bootstrapcfg.Addr.IP.To4() == nil {
			return fmt.Errorf("invalid IP address: %v", s)
		}
		if _, port, err := net.SplitHostPort(s); err == nil {
			p, err := strconv.Atoi(port)
			if err != nil {
				return fmt.Errorf("invalid port: %v", port)
			}
			bootstrapcfg.Addr.Port = p
		}
		switch bootstrapcfg.Addr.Port {
		default:
			return fmt.Errorf("invalid port: %v", bootstrapcfg.Addr.Port)
		case params.MainnetChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.MainnetChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "aqua"
			bootstrapcfg.Bootnodes = (discover.BootnodeStringList)(params.MainnetBootnodes).ToDiscoverNodes()
		case params.TestnetChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.TestnetChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet"
			bootstrapcfg.Bootnodes = (discover.BootnodeStringList)(params.TestnetBootnodes).ToDiscoverNodes()
		case params.Testnet2ChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.Testnet2ChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet2"
			bootstrapcfg.Bootnodes = (discover.BootnodeStringList)(params.Testnet2Bootnodes).ToDiscoverNodes()
		case params.Testnet3ChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.Testnet3ChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet3"
			bootstrapcfg.Bootnodes = (discover.BootnodeStringList)(params.Testnet3Bootnodes).ToDiscoverNodes()
		}
		return nil
	})
	flag.Func("nodekey", "private key filename, or use default location. generate one with -genkey [filename]", func(s string) error {
		var err error
		bootstrapcfg.nodekey, err = crypto.LoadECDSA(s)
		if err != nil {
			return err
		}
		return nil
	})
	flag.StringVar(&bootstrapcfg.Nat, "nat", "none", "port mapping mechanism (any|none|upnp|pmp|extip:<IP>)")
	flag.Func("netrestrict", "restrict network communication to the given IP networks (CIDR masks)", func(s string) error {
		// bootstrapcfg.NetRestrict = netutil.NewNetlist(strings.Split(s, ","))
		var err error
		bootstrapcfg.NetRestrict, err = netutil.ParseNetlist(s)
		if err != nil {
			return err
		}

		return nil
	})

	var (
		genKey    string
		writeAddr bool
		verbosity int
		vmodule   string
		debug     bool = os.Getenv("DEBUG") != ""
	)
	flag.StringVar(&genKey, "genkey", "", "generate a new private key and write to file (file must not exist)")
	flag.BoolVar(&writeAddr, "writeaddress", false, "write out the node's pubkey hash and quit")
	flag.Func("announceIP", "announce IP address (default is listen address)", func(s string) error {
		var err error
		bootstrapcfg.AnnounceAddr, err = net.ResolveUDPAddr("udp4", s)
		return err
	})
	// runv5       = flag.Bool("v5", false, "run a v5 topic discovery bootnode")

	flag.IntVar(&verbosity, "verbosity", int(log.LvlInfo), "log verbosity (0-9)")
	flag.StringVar(&vmodule, "vmodule", "", "log verbosity pattern (eg 'p2p=5,aqua=3' or try 'good' or 'great' aliases)")
	flag.Uint64Var(&bootstrapcfg.ChainId, "chainid", bootstrapcfg.ChainId, "chain id for p2p communication (deprecated, use chain flag)")
	flag.StringVar(&bootstrapcfg.ChainName, "chain", bootstrapcfg.ChainName, "chain name to get chainid from config (aqua, mainnet, testnet, testnet2 ...)")
	flag.BoolVar(&debug, "debug", debug, "debug mode (only shows line numbers, see -verbosity 9 for full debug)")
	var colormode bool = true
	flag.BoolVar(&colormode, "color", colormode, "colorize log output (default on when attached to terminal)")
	var (
		// nodeKey *btcec.PrivateKey
		err error
	)
	flag.Parse()
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(colormode)))
	glogger.Verbosity(log.Lvl(verbosity))
	glogger.Vmodule(vmodule)
	log.Root().SetHandler(glogger)
	if debug {
		log.PrintOrigins(true) // show line numbers
	}

	nodeKey := bootstrapcfg.nodekey
	if nodeKey == nil {
		log.Error("no private key set (-nodekey flag), for example, use -nodekey ~/.aquachain/aquachain/nodekey")
		os.Exit(1)
	}

	// subcommands
	switch {
	case genKey != "":
		nodeKey, err = crypto.GenerateKey()
		if err != nil {
			utils.Fatalf("could not generate key: %v", err)
		}
		if err = crypto.SaveECDSA(genKey, nodeKey); err != nil {
			utils.Fatalf("%v", err)
		}
		return
	case writeAddr:
		fmt.Printf("%v\n", discover.PubkeyID(nodeKey.PubKey().ToECDSA()))
		os.Exit(0)
	}

	// serving...
	servebootstrap(bootstrapcfg)
}
func servebootstrap(bootstrapcfg BootstrapConfig) {
	addr := bootstrapcfg.Addr
	if bootstrapcfg.ChainName != "" { // get port from chain name
		cfg := params.GetChainConfig(bootstrapcfg.ChainName)
		if cfg == nil {
			utils.Fatalf("chain not found: %v", bootstrapcfg.ChainName)
		}
		addr.Port = cfg.DefaultBootstrapPort
		bootstrapcfg.ChainId = cfg.ChainId.Uint64()
	} else if bootstrapcfg.ChainId != 0 {
		// set port from chainid
		switch bootstrapcfg.ChainId {
		case params.MainnetChainConfig.ChainId.Uint64():
			bootstrapcfg.ChainName = "mainnet"
			addr.Port = params.MainnetChainConfig.DefaultBootstrapPort
		case params.TestnetChainConfig.ChainId.Uint64():
			bootstrapcfg.ChainName = "testnet"
			addr.Port = params.TestnetChainConfig.DefaultBootstrapPort
		case params.Testnet2ChainConfig.ChainId.Uint64():
			bootstrapcfg.ChainName = "testnet2"
			addr.Port = params.Testnet2ChainConfig.DefaultBootstrapPort
		case params.Testnet3ChainConfig.ChainId.Uint64():
			bootstrapcfg.ChainName = "testnet3"
			addr.Port = params.Testnet3ChainConfig.DefaultBootstrapPort
		default:
			// no port change
		}
	} else {
		switch addr.Port { // no custom chain, use port number to guess
		case params.MainnetChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.MainnetChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "mainnet"
		case params.TestnetChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.TestnetChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet"
		case params.Testnet2ChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.Testnet2ChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet2"
		case params.Testnet3ChainConfig.DefaultBootstrapPort:
			bootstrapcfg.ChainId = params.Testnet3ChainConfig.ChainId.Uint64()
			bootstrapcfg.ChainName = "testnet3"
		default:
			bootstrapcfg.ChainName = "unknown"
			// keep chainid
		}
		log.Info("chain set", "chain", bootstrapcfg.ChainName, "port", addr.Port, "chainid", bootstrapcfg.ChainId)
	}

	natm, err := nat.Parse(bootstrapcfg.Nat)
	if err != nil {
		utils.Fatalf("-nat: %v", err)
	}

	conn, err := net.ListenUDP("udp4", &addr)
	if err != nil {
		utils.Fatalf("-ListenUDP: %v", err)
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	// if *announceIP != "" {
	// 	announceAddr, err := net.ResolveUDPAddr("udp", *announceIP)
	// 	if err != nil {
	// 		utils.Fatalf("-announce: %v", err)
	// 	}
	// 	realaddr.IP = announceAddr.IP
	// }
	if natm != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(natm, nil, "udp", realaddr.Port, realaddr.Port, "aquachain discovery")
		}
		// TODO: react to external IP changes over time.
		if ext, err := natm.ExternalIP(); err == nil {
			realaddr = &net.UDPAddr{IP: ext, Port: realaddr.Port}
		}
	}

	// switch {
	// case bootstrapcfg.ChainId != 0:
	// 	// ok
	// case bootstrapcfg.ChainName != "":
	// 	cfg := params.GetChainConfig(bootstrapcfg.ChainName)
	// 	if cfg == nil {
	// 		utils.Fatalf("chain not found: %v", bootstrapcfg.ChainName)
	// 	}
	// 	bootstrapcfg.ChainId = cfg.ChainId.Uint64()
	// default:
	// 	// from listen address
	// 	switch realaddr.Port {
	// 	case params.MainnetChainConfig.DefaultBootstrapPort:
	// 		bootstrapcfg.ChainId = params.MainnetChainConfig.ChainId.Uint64()
	// 	case params.TestnetChainConfig.DefaultBootstrapPort:
	// 		bootstrapcfg.ChainId = params.TestnetChainConfig.ChainId.Uint64()
	// 	case params.Testnet2ChainConfig.DefaultBootstrapPort:
	// 		bootstrapcfg.ChainId = params.Testnet2ChainConfig.ChainId.Uint64()
	// 	case params.Testnet3ChainConfig.DefaultBootstrapPort:
	// 		bootstrapcfg.ChainId = params.Testnet3ChainConfig.ChainId.Uint64()
	// 	default:
	// 		utils.Fatalf("could not determine chainid from port, use -chainid or -chainname to hint")
	// 	}
	// }

	var bootstraps []*discover.Node = Chain2Bootstraps(bootstrapcfg.ChainName)
	cfg := discover.Config{
		PrivateKey:   bootstrapcfg.nodekey,
		AnnounceAddr: realaddr,
		NetRestrict:  bootstrapcfg.NetRestrict,
		ChainId:      bootstrapcfg.ChainId,
		Bootnodes:    bootstraps,
	}

	log.Info("serving bootstrap", "network", bootstrapcfg.ChainName, "port", realaddr.Port, "chainid", bootstrapcfg.ChainId)
	time.Sleep(time.Second) // wait for nat mapping

	if _, err := discover.ListenUDP(conn, cfg); err != nil {
		utils.Fatalf("could not listen udp: %v", err)
	}
	select {}
}

func Chain2Bootstraps(s string) []*discover.Node {
	switch s {
	case "mainnet", "aqua":
		return (discover.BootnodeStringList)(params.MainnetBootnodes).ToDiscoverNodes()
	case "testnet":
		return (discover.BootnodeStringList)(params.TestnetBootnodes).ToDiscoverNodes()
	case "testnet2":
		return (discover.BootnodeStringList)(params.Testnet2Bootnodes).ToDiscoverNodes()
	}
	return nil
}
