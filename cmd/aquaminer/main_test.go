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

package main

import (
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/crypto"
)

func TestMiner(t *testing.T) {
	// dummy work load
	workHash := common.HexToHash("0xd3b5f1b47f52fdc72b1dab0b02ab352442487a1d3a43211bc4f0eb5f092403fc")
	target := new(big.Int).SetBytes(common.HexToHash("0x08637bd05af6c69b5a63f9a49c2c1b10fd7e45803cd141a6937d1fe64f54").Bytes())

	// good nonce
	nonce := uint64(14649775584697213406)

	seed := make([]byte, 40)
	copy(seed, workHash.Bytes())
	fmt.Printf("hashing work: %x\nless than target:  %s\nnonce: %v\n", workHash, target, nonce)

	// debug
	fmt.Printf("seednononc: %x\n", seed)

	// little endian
	binary.LittleEndian.PutUint64(seed[32:], nonce)

	// pre hash
	fmt.Printf("beforehash: %x\n", seed)

	// hash
	result := crypto.VersionHash(2, seed)

	// difficulty
	out := new(big.Int).SetBytes(result)
	fmt.Printf("result difficulty: %s\n", out)
	fmt.Printf("result difficulty: %x\n", out)

	// test against target difficulty
	testresult := out.Cmp(target) <= 0
	fmt.Printf("%x: %v\n", out, testresult)
	if !testresult {
		t.FailNow()
	}
}

func TestZeros(t *testing.T) {

	seed := make([]byte, 40)
	result := crypto.VersionHash(2, seed)
	fmt.Printf("%02x -> %02x\n", seed, result)
}
