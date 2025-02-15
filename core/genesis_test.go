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

package core

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"testing"

	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core/vm"
	"gitlab.com/aquachain/aquachain/params"
)

// for generating delloc array for HF4 (only on mainnet)
func TestDefaultGenesisAlloc(t *testing.T) {
	t.SkipNow()
	m := DefaultGenesisBlock().Alloc
	fmt.Println("Bad Balances:", len(m))
	aquawei := big.NewFloat(params.Aqua)
	_ = aquawei
	mode := 0
	for k, v := range m {
		_ = v
		switch mode {
		case 0: // print balances for humans
			fmt.Printf("%x %.6f\n", k, new(big.Float).Quo(new(big.Float).SetInt(v.Balance), aquawei))
		case 1: // print address array for dealloc (HF4)
			fmt.Printf(`"%x",`+"\n", k)
		}
	}
}
func TestDefaultGenesisBlock(t *testing.T) {
	block := DefaultGenesisBlock().ToBlock(nil)
	if block.Hash() != params.MainnetGenesisHash {
		t.Errorf("wrong mainnet genesis hash, got %x, want %x", block.Hash(), params.MainnetGenesisHash)
	}
	block = DefaultTestnetGenesisBlock().ToBlock(nil)
	if block.Hash() != params.TestnetGenesisHash {
		t.Errorf("wrong testnet genesis hash, got %x, want %x", block.Hash(), params.TestnetGenesisHash)
	}
	block = DefaultTestnet2GenesisBlock().ToBlock(nil)
	if block.Hash() != params.Testnet2GenesisHash {
		t.Errorf("wrong testnet2 genesis hash, got %x, want %x", block.Hash(), params.Testnet2GenesisHash)
	}
	block = DefaultTestnet3GenesisBlock().ToBlock(nil)
	if block.Hash() != params.Testnet3GenesisHash {
		t.Errorf("wrong testnet3 genesis hash, got %x, want %x", block.Hash(), params.Testnet3GenesisHash)
	}
}

func TestSetupGenesis(t *testing.T) {
	var (
		customghash = common.HexToHash("0x73ddc0ce61cfa68e5dc3edb1d5b6d16583d739b9430c1d02ca9981b7dd26b7f2")
		customg     = Genesis{
			Config: &params.ChainConfig{HomesteadBlock: big.NewInt(3), HF: params.TestChainConfig.HF, ChainId: big.NewInt(101)},
			Alloc: GenesisAlloc{
				{1}: {Balance: big.NewInt(1), Storage: map[common.Hash]common.Hash{{1}: {1}}},
			},
		}
		oldcustomg = customg
	)
	if wanthash := customg.ToBlock(nil).Hash(); wanthash != customghash {
		t.Fatalf("bad custom hash, got %x, want %x", wanthash, customghash)
	}
	oldcustomg.Config = &params.ChainConfig{HomesteadBlock: big.NewInt(2), HF: params.TestChainConfig.HF, ChainId: big.NewInt(102)}
	tests := []struct {
		name       string
		fn         func(aquadb.Database) (*params.ChainConfig, common.Hash, error)
		wantConfig *params.ChainConfig
		wantHash   common.Hash
		wantErr    error
	}{
		{
			name: "genesis without ChainConfig",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, new(Genesis))
			},
			wantErr:    errGenesisNoConfig,
			wantConfig: params.AllAquahashProtocolChanges,
		},
		{
			name: "no block in DB, genesis == nil",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "mainnet block in DB, genesis == nil",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				DefaultGenesisBlock().MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   params.MainnetGenesisHash,
			wantConfig: params.MainnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == nil",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, nil)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "custom block in DB, genesis == testnet",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultTestnetGenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.TestnetGenesisHash},
			wantHash:   params.TestnetGenesisHash,
			wantConfig: params.TestnetChainConfig,
		},
		{
			name: "custom block in DB, genesis == testnet2",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultTestnet2GenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.Testnet2GenesisHash},
			wantHash:   params.Testnet2GenesisHash,
			wantConfig: params.Testnet2ChainConfig,
		},
		{
			name: "custom block in DB, genesis == testnet3",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				customg.MustCommit(db)
				return SetupGenesisBlock(db, DefaultTestnet3GenesisBlock())
			},
			wantErr:    &GenesisMismatchError{Stored: customghash, New: params.Testnet3GenesisHash},
			wantHash:   params.Testnet3GenesisHash,
			wantConfig: params.Testnet3ChainConfig,
		},
		{
			name: "compatible config in DB",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				oldcustomg.MustCommit(db)
				return SetupGenesisBlock(db, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
		},
		{
			name: "incompatible config in DB",
			fn: func(db aquadb.Database) (*params.ChainConfig, common.Hash, error) {
				// Commit the 'old' genesis block with Homestead transition at #2.
				// Advance to block #4, past the homestead transition block of customg.
				genesis := oldcustomg.MustCommit(db)

				bc, _ := NewBlockChain(context.TODO(), db, nil, oldcustomg.Config, aquahash.NewFullFaker(), vm.Config{})
				defer bc.Stop()

				blocks, _ := GenerateChain(context.TODO(), oldcustomg.Config, genesis, aquahash.NewFaker(), db, 4, nil)
				bc.InsertChain(blocks)
				bc.CurrentBlock()
				// This should return a compatibility error.
				return SetupGenesisBlock(db, &customg)
			},
			wantHash:   customghash,
			wantConfig: customg.Config,
			wantErr: &params.ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(2),
				NewConfig:    big.NewInt(3),
				RewindTo:     1,
			},
		},
	}

	for _, test := range tests {
		db := aquadb.NewMemDatabase()
		log.Info("Running test", "name", test.name)
		config, hash, err := test.fn(db)
		// Check the return values.
		if (err == nil) != (test.wantErr == nil) {
			t.Fatalf("%s: returned %v, wanted error %v", test.name, err, test.wantErr)
		}
		if test.wantErr != nil && test.wantErr != err { // not exact pointer match, compare fields
			switch x := test.wantErr.(type) {
			case *GenesisMismatchError:
				y, ok := err.(*GenesisMismatchError)
				if !ok {
					t.Fatalf("%s: returned error %v, wanted error %v", test.name, err, test.wantErr)
				}
				if x.Stored != y.Stored || x.New != y.New {
					t.Fatalf("%s: returned error %v, wanted error %v", test.name, err, test.wantErr)
				}
			case *params.ConfigCompatError:
				y, ok := err.(*params.ConfigCompatError)
				if !ok {
					t.Fatalf("%s: returned error %v, wanted error %v", test.name, err, test.wantErr)
				}
				if x.What != y.What || x.RewindTo != y.RewindTo || x.StoredConfig.Cmp(y.StoredConfig) != 0 {
					t.Fatalf("%s: returned error %v, wanted error %v", test.name, err, test.wantErr)
				}
			default:
				t.Fatalf("%s: returned unknonw error %v, wanted error %v", test.name, err, test.wantErr)
			}
		}
		if !reflect.DeepEqual(config, test.wantConfig) {
			t.Fatalf("%s:\nreturned %v\nwant     %v", test.name, config, test.wantConfig)
		}
		if hash != test.wantHash {
			t.Fatalf("%s: returned hash %s, want %s", test.name, hash.Hex(), test.wantHash.Hex())
		} else if err == nil {
			// Check database content.
			stored := GetBlockNoVersion(db, test.wantHash, 0)
			if stored.SetVersion(config.GetBlockVersion(stored.Number())) != test.wantHash {
				t.Errorf("%s: block in DB has hash %s, want %s", test.name, stored.Hash(), test.wantHash)
			}
		}
	}
}
