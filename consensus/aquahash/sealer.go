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
	"encoding/binary"
	"math/big"
	mrand "math/rand"
	"runtime"
	"sync"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/consensus"
	"gitlab.com/aquachain/aquachain/consensus/aquahash/ethashdag"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/params"
)

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (aquahash *Aquahash) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// If we're running a fake PoW, simply return a 0 nonce immediately
	chaincfg := params.TestChainConfig
	if chain != nil {
		chaincfg = chain.Config()
		log.Trace("[sealer] Using aquahash engine", "chaincfg", chaincfg.String())
	}
	if aquahash.config.PowMode == ModeFake || aquahash.config.PowMode == ModeFullFake {
		log.Trace("[sealer] aquahash engine using fake POW")
		header := block.Header()
		header.Version = chaincfg.GetBlockVersion(header.Number)
		header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
		return block.WithSeal(header), nil
	}
	// If we're running a shared PoW, delegate sealing to it
	if aquahash.shared != nil {
		return aquahash.shared.Seal(chain, block, stop)
	}
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})
	found := make(chan *types.Block)

	aquahash.lock.Lock()
	threads := aquahash.threads
	aquahash.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	var pend sync.WaitGroup
	version := chaincfg.GetBlockVersion(block.Number())
	if version == 0 {
		return nil, errUnknownGrandparent
	}
	if version == 1 && aquahash.ethashdag == nil {
		// make sure we have a dataset for ethash version 1
		aquahash.ethashdag = ethashdag.New(aquahash.config)
		aquahash.ethashdag.Dataset(block.NumberU64())
	}
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			log.Trace("launching miner", "algoVersion", version)
			aquahash.mine(version, block, id, nonce, abort, found)
		}(i, uint64(mrand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	var result *types.Block
	select {
	case <-stop:
		// Outside abort, stop all miner threads
		close(abort)
	case result = <-found:
		// One of the threads found a block, abort all others
		close(abort)
	case <-aquahash.update:
		// Thread count was changed on user request, restart
		close(abort)
		pend.Wait()
		return aquahash.Seal(chain, block, stop)
	}
	// Wait for all miners to terminate and return the block
	pend.Wait()
	return result, nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final block difficulty.
func (aquahash *Aquahash) mine(version params.HeaderVersion, block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	// Extract some data from the header
	var (
		header  = block.Header()
		hash    = header.HashNoNonce().Bytes()
		target  = new(big.Int).Div(maxUint256, header.Difficulty)
		number  = header.Number.Uint64()
		dataset *ethashdag.Dataset
	)
	header.Version = version
	if header.Version == 0 || header.Version > crypto.KnownVersion {
		common.Report("Mining incorrect version")
		return
	}
	if header.Version == 1 {
		if aquahash.ethashdag == nil {
			log.Warn("Mining with ethash version 1 is disabled")
			return
		}
		log.Warn("Mining with ethash version 1")
		dataset = aquahash.ethashdag.Dataset(number)
	}

	// Start generating random nonces until we abort or find a good one
	var (
		// attempts = int64(0)
		nonce = seed
	)
	logger := log.New("miner", id)
	logger.Trace("Started aquahash search for new nonces", "seed", seed, "algo", version, "number", number, "difficulty", header.Difficulty, "target", target)
search:
	for {

		select {
		case <-abort:
			// Mining terminated, update stats and abort
			logger.Trace("Aquahash nonce search aborted", "attempts", nonce-seed)
			// aquahash.hashrate.Mark(attempts)
			break search

		default:
			// // We don't have to update hash rate on every nonce, so update after after 2^X nonces
			// attempts++
			// if (attempts % (1 << 15)) == 0 {
			// 	// aquahash.hashrate.Mark(attempts)
			// 	attempts = 0
			// }

			// Compute the PoW value of this nonce
			var (
				digest []byte
				result []byte
			)

			switch header.Version {
			case 1:
				digest, result = ethashdag.HashimotoFull(dataset.GetDataset(), hash, nonce)
			default:
				seed := make([]byte, 40)
				copy(seed, hash)
				binary.LittleEndian.PutUint64(seed[32:], nonce)
				result = crypto.VersionHash(byte(header.Version), seed)
				digest = make([]byte, common.HashLength)
			}

			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				header = types.CopyHeader(header)
				header.Nonce = types.EncodeNonce(nonce)
				header.MixDigest = common.BytesToHash(digest)

				// Seal and return a block (if still needed)
				select {
				case found <- block.WithSeal(header):
					logger.Trace("Aquahash nonce found and reported", "attempts", nonce-seed, "nonce", nonce)
				case <-abort:
					logger.Trace("Aquahash nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				break search
			}
			nonce++
		}
	}
	// Datasets are unmapped in a finalizer. Ensure that the dataset stays live
	// during sealing so it's not unmapped while being read.
	if dataset != nil {
		runtime.KeepAlive(dataset)
	}
}
