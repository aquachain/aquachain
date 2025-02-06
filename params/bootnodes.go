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

package params

type BootnodeStringList []string

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Aquachain network.
var MainnetBootnodes = []string{
	"enode://ef5a7f89789f9150e282c4a37d317596c1f29e0b51748269472395c45790784d585b253089ef87e579408de88961f1e43f15d5dbc271d612b676f1961792814f@168.235.107.40:21000", // c.onical.org (2020/12/05)
	"enode://5b152d555ecd59d225a9cec9c002554fe1afc75a96ec6c4b021524029b3c683df31981f4e460cf02da8740dbae97b213d899a2c5464e21dfe0223633d00e2dd5@168.235.111.37:21000", // tf1
	"enode://61cf6974b566caefc7238c3095692a3cac266a0eb76cb4c950034427a3112388d28db410412bc5d59d54789c6418b4aa59af797f34b39c1c08c2642fa60ddc90@168.235.85.77:21000",  // tf2
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// test network. (port 21001)
var TestnetBootnodes = []string{
	"enode://ef5a7f89789f9150e282c4a37d317596c1f29e0b51748269472395c45790784d585b253089ef87e579408de88961f1e43f15d5dbc271d612b676f1961792814f@168.235.107.40:21001", // c.onical.org
	"enode://5b152d555ecd59d225a9cec9c002554fe1afc75a96ec6c4b021524029b3c683df31981f4e460cf02da8740dbae97b213d899a2c5464e21dfe0223633d00e2dd5@168.235.111.37:21001", // tf1
	"enode://61cf6974b566caefc7238c3095692a3cac266a0eb76cb4c950034427a3112388d28db410412bc5d59d54789c6418b4aa59af797f34b39c1c08c2642fa60ddc90@168.235.85.77:21001",  // tf2
}

// Testnet2Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Testnet2 test network. (port 21002)
var Testnet2Bootnodes = []string{
	"enode://ef5a7f89789f9150e282c4a37d317596c1f29e0b51748269472395c45790784d585b253089ef87e579408de88961f1e43f15d5dbc271d612b676f1961792814f@168.235.107.40:21002", // c.onical.org
	"enode://5b152d555ecd59d225a9cec9c002554fe1afc75a96ec6c4b021524029b3c683df31981f4e460cf02da8740dbae97b213d899a2c5464e21dfe0223633d00e2dd5@168.235.111.37:21002", // tf1
	"enode://61cf6974b566caefc7238c3095692a3cac266a0eb76cb4c950034427a3112388d28db410412bc5d59d54789c6418b4aa59af797f34b39c1c08c2642fa60ddc90@168.235.85.77:21002",  // tf2
}

// Testnet3Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Testnet3 test network. (port 21003)
var Testnet3Bootnodes = []string{
	"enode://ef5a7f89789f9150e282c4a37d317596c1f29e0b51748269472395c45790784d585b253089ef87e579408de88961f1e43f15d5dbc271d612b676f1961792814f@168.235.107.40:21003", // c.onical.org
	"enode://5b152d555ecd59d225a9cec9c002554fe1afc75a96ec6c4b021524029b3c683df31981f4e460cf02da8740dbae97b213d899a2c5464e21dfe0223633d00e2dd5@168.235.111.37:21003", // tf1
	"enode://61cf6974b566caefc7238c3095692a3cac266a0eb76cb4c950034427a3112388d28db410412bc5d59d54789c6418b4aa59af797f34b39c1c08c2642fa60ddc90@168.235.85.77:21003",  // tf2
}

var EthnetBootnodes = []string{}
