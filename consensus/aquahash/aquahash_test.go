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

package aquahash

import (
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"testing"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/params"
)

// Tests that aquahash works correctly in test mode.
func TestTestMode(t *testing.T) {
	head := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}
	head.Version = types.H_KECCAK256
	aquahash := NewTester()
	block, err := aquahash.Seal(nil, types.NewBlockWithHeader(head), nil)
	if err != nil {
		t.Fatalf("failed to seal block: %v", err)
	}
	head.Nonce = types.EncodeNonce(block.Nonce())
	head.MixDigest = block.MixDigest()
	if err := aquahash.VerifySeal(nil, head); err != nil {
		t.Fatalf("unexpected verification error: %+v", err)
	}
}

// This test checks that cache lru logic doesn't crash under load.
// It reproduces https://gitlab.com/aquachain/aquachain/issues/14943
func TestCacheFileEvict(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "aquahash-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	e := New(&Config{CachesInMem: 3, CachesOnDisk: 10, CacheDir: tmpdir, PowMode: ModeTest})

	workers := 8
	epochs := 100
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go verifyTest(&wg, e, i, epochs)
	}
	wg.Wait()
}

func verifyTest(wg *sync.WaitGroup, e *Aquahash, workerIndex, epochs int) {
	defer wg.Done()

	const wiggle = 4 * epochLength
	r := rand.New(rand.NewSource(int64(workerIndex)))
	for epoch := 0; epoch < epochs; epoch++ {
		block := int64(epoch)*epochLength - wiggle/2 + r.Int63n(wiggle)
		if block < 0 {
			block = 0
		}
		head := &types.Header{Number: big.NewInt(block), Difficulty: big.NewInt(100)}
		head.Version = params.AllAquahashProtocolChanges.GetBlockVersion(big.NewInt(block))
		e.VerifySeal(nil, head)
	}
}

// Tests that caches generated on disk may be done concurrently.
func TestConcurrentDiskCacheGeneration(t *testing.T) {
	// Create a temp folder to generate the caches into
	cachedir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary cache dir: %v", err)
	}
	defer os.RemoveAll(cachedir)
	log.Printf("Using cache dir: %s", cachedir)
	// Define a heavy enough block, one from mainnet should do
	block := types.NewBlockWithHeader(&types.Header{
		Number:      big.NewInt(3311058),
		ParentHash:  common.HexToHash("0xd783efa4d392943503f28438ad5830b2d5964696ffc285f338585e9fe0a37a05"),
		UncleHash:   common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
		Coinbase:    common.HexToAddress("0xc0ea08a2d404d3172d2add29a45be56da40e2949"),
		Root:        common.HexToHash("0x77d14e10470b5850332524f8cd6f69ad21f070ce92dca33ab2858300242ef2f1"),
		TxHash:      common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		ReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
		Difficulty:  big.NewInt(167925187834220),
		GasLimit:    4015682,
		GasUsed:     0,
		Time:        big.NewInt(1488928920),
		Extra:       []byte("www.bw.com"),
		MixDigest:   common.HexToHash("0x3e140b0784516af5e5ec6730f2fb20cca22f32be399b9e4ad77d32541f798cd0"),
		Nonce:       types.EncodeNonce(0xf400cd0006070c49),
		Version:     types.H_KECCAK256,
	})
	// Simulate multiple processes sharing the same datadir
	var pend sync.WaitGroup

	for i := 0; i < 3; i++ {
		pend.Add(1)

		go func(idx int) {
			defer pend.Done()
			aquahash := New(&Config{CacheDir: cachedir, CachesOnDisk: 1, PowMode: ModeTest, StartVersion: 1})
			if err := aquahash.VerifySeal(nil, block.Header()); err != nil {
				t.Errorf("proc %d: block verification failed: %+v", idx, err)
			}
		}(i)
	}
	pend.Wait()
}
