// Copyright 2015 The aquachain Authors
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

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main AquaChain network.
var MainnetBootnodes = []string{
	"enode://7f636b8198a41abb10c1a571992335b8cb760d6ef973efc5f3ff613dda7acbe9e6d6b27254e076ef7b684ac7ea09a27bd05a37844cd8ad242199593bdd8cec21@107.161.24.142:21000",  // aquachain-1
	"enode://6227ff2948ff51ee4f09e5f1df2c1270c47b753718d406605787326341de6ff8e7cb6a5f01a4deed5437dcdd7b9fb8e656f0ad6a08c1f677c2ca5d0e666a92fc@168.235.78.103:21000",  // aquachain-2
	"enode://1a6b78cf626540d1eecfeba1f364e72bf92847561b9344403ac7010b2be184cfc760b5bcd21402b19713deebef256dcdfc5af67487554650bf07807737a36203@23.94.123.137:21303",   // aerthnode
	"enode://a341920437d7141e4355ed1f298fd2415cbec781c8b4fedd943eac37fd0c835375718085b1a65208c0a06af10c388d452a4148b8430da93bd0b75100b2315f3c@107.161.24.142:21303",  // aerthpool
	"enode://fa2ad4aba184e5c7cbcdc613dcfb70ecd7ebd77fcef1b9b1b468476a7f043f7d0fe8befb7c2ace319b0c2316a94d743c3a5ca401ef5d35c897c5a8af65d1badd@140.82.42.96:21000",    // GribblyRulez
	"enode://f413c22f83972c7736668fc5f3b6b1eaccb9a59da9ed75538ff3ea85a842b76f680aac72fcd3549d153f2886e6231595ab0d2f1c5cc657318f2752ad46bd0cca@93.171.216.173:21000",  // artemka
	"enode://f6a307137829a32941249d8bf2bfa9e365735a120b3acaeff339af6ee4111f81bae48812671bfff26e98c851d8c3a2e51c32a32b264b94f29715c619b098053c@173.212.244.109:21000", // abudfv
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network. (port 21001)
var DiscoveryV5Bootnodes = []string{
	"enode://7f636b8198a41abb10c1a571992335b8cb760d6ef973efc5f3ff613dda7acbe9e6d6b27254e076ef7b684ac7ea09a27bd05a37844cd8ad242199593bdd8cec21@107.161.24.142:21001", // aquachain-1 new protocol
	"enode://6227ff2948ff51ee4f09e5f1df2c1270c47b753718d406605787326341de6ff8e7cb6a5f01a4deed5437dcdd7b9fb8e656f0ad6a08c1f677c2ca5d0e666a92fc@168.235.78.103:21001", // aquachain-2 new protocol
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// test network. (port 21002)
var TestnetBootnodes = []string{
	"enode://6227ff2948ff51ee4f09e5f1df2c1270c47b753718d406605787326341de6ff8e7cb6a5f01a4deed5437dcdd7b9fb8e656f0ad6a08c1f677c2ca5d0e666a92fc@168.235.78.103:21002", // aquachain-2 testnet new protocol
}

// Testnet2Bootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Testnet2 test network.
var Testnet2Bootnodes = []string{}

var EthnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://d860a01f9722d78051619d1e2351aba3f43f943f6f00718d1b9baa4101932a1f5011f16bb2b1bb35db20d6fe28fa0bf09636d26a87d31de9ec6203eeedb1f666@18.138.108.67:30303",   // bootnode-aws-ap-southeast-1-001
	"enode://22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de@3.209.45.79:30303",     // bootnode-aws-us-east-1-001
	"enode://ca6de62fce278f96aea6ec5a2daadb877e51651247cb96ee310a318def462913b653963c155a0ef6c7d50048bba6e6cea881130857413d9f50a621546b590758@34.255.23.113:30303",   // bootnode-aws-eu-west-1-001
	"enode://279944d8dcd428dffaa7436f25ca0ca43ae19e7bcf94a8fb7d1641651f92d121e972ac2e8f381414b80cc8e5555811c2ec6e1a99bb009b3f53c4c69923e11bd8@35.158.244.151:30303",  // bootnode-aws-eu-central-1-001
	"enode://8499da03c47d637b20eee24eec3c356c9a2e6148d6fe25ca195c7949ab8ec2c03e3556126b0d7ed644675e78c4318b08691b7b57de10e5f0d40d05b09238fa0a@52.187.207.27:30303",   // bootnode-azure-australiaeast-001
	"enode://103858bdb88756c71f15e9b5e09b56dc1be52f0a5021d46301dbbfb7e130029cc9d0d6f73f693bc29b665770fff7da4d34f3c6379fe12721b5d7a0bcb5ca1fc1@191.234.162.198:30303", // bootnode-azure-brazilsouth-001
	"enode://715171f50508aba88aecd1250af392a45a330af91d7b90701c436b618c86aaa1589c9184561907bebbb56439b8f8787bc01f49a7c77276c58c1b09822d75e8e8@52.231.165.108:30303",  // bootnode-azure-koreasouth-001
	"enode://5d6d7cd20d6da4bb83a1d28cadb5d409b64edf314c0335df658c1a54e32c7c4a7ab7823d57c39b6a757556e68ff1df17c748b698544a55cb488b52479a92b60f@104.42.217.25:30303",   // bootnode-azure-westus-001

	// Ethereum Foundation Go Bootnodes
	"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// Ethereum Foundation C++ Bootnodes
	"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE
}
