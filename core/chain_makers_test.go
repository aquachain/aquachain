// Copyright 2019 The aquachain Authors
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
	"testing"

	"gitlab.com/aquachain/aquachain/aquadb"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/consensus"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/core/vm"
	"gitlab.com/aquachain/aquachain/params"
)

// So we can deterministically seed different blockchains
const (
	canonicalSeed = 1
	forkSeed      = 2
)

type HasContext interface {
	GetContext() context.Context
}

// newCanonical creates a chain database, and injects a deterministic canonical
// chain. Depending on the full flag, if creates either a full block chain or a
// header only chain.
func newCanonical(engine consensus.Engine, n int, full bool) (aquadb.Database, *BlockChain, error) {
	// Initialize a fresh chain with only a genesis block
	gspec := new(Genesis)
	db := aquadb.NewMemDatabase()
	genesis := gspec.MustCommit(db)
	ctx := context.TODO()
	if ctxer, ok := engine.(HasContext); ok {
		ctx = ctxer.GetContext()
	} else {
		log.Warn("no context in test chain maker")
	}
	blockchain, _ := NewBlockChain(ctx, db, nil, params.AllAquahashProtocolChanges, engine, vm.Config{
		Tracer: vm.NewStructLogger(nil),
	})
	if blockchain == nil {
		return nil, nil, fmt.Errorf("failed to create blockchain")
	}
	if blockchain.genesisBlock == nil {
		return nil, nil, fmt.Errorf("failed to create blockchain genesis")
	}
	log.Info("genesis block", "hash", blockchain.genesisBlock.Hash())
	// Create and inject the requested chain
	if n == 0 {
		return db, blockchain, nil
	}
	if full {
		// Full block-chain requested
		blocks := makeBlockChain(genesis, n, engine, db, canonicalSeed)
		_, err := blockchain.InsertChain(blocks)
		return db, blockchain, err
	}
	// Header-only chain requested
	headers := makeHeaderChain(genesis.Header(), n, engine, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(headers, 1)
	return db, blockchain, err
}

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.Header, n int, engine consensus.Engine, db aquadb.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, engine, db, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
		headers[i].Version = params.TestChainConfig.GetBlockVersion(headers[i].Number)
	}
	return headers
}

// makeBlockChain creates a deterministic chain of blocks rooted at parent.
func makeBlockChain(parent *types.Block, n int, engine consensus.Engine, db aquadb.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(context.TODO(), params.TestChainConfig, parent, engine, db, n, func(i int, b *BlockGen) {
		b.header.Version = b.config.GetBlockVersion(b.Number())
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}

func TestChainMaker(t *testing.T) {
	genesis := new(Genesis)
	db := aquadb.NewMemDatabase()
	genesis.MustCommit(db)
	engine := aquahash.NewFullFaker()
	blockchain, err := NewBlockChain(context.TODO(), db, nil, params.AllAquahashProtocolChanges, engine, vm.Config{
		Tracer: vm.NewStructLogger(nil),
	})
	if err != nil {
		t.Fatalf("failed to create blockchain: %+v", err)
	}
	if blockchain == nil {
		t.Fatalf("failed to create blockchain")
	}
	defer blockchain.Stop()

	// Create a deterministic chain of 10 blocks
	blocks := makeBlockChain(genesis.ToBlock(db), 10, engine, db, canonicalSeed)
	_, err = blockchain.InsertChain(blocks)
	if err != nil {
		t.Fatalf("failed to insert chain: %v", err)
	}
}
