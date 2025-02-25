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

// Package aquahash implements the aquahash proof-of-work consensus engine.
package aquahash

import (
	"sync"
	"time"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/consensus"
	"gitlab.com/aquachain/aquachain/consensus/aquahash/ethashdag"
	"gitlab.com/aquachain/aquachain/rpc"
)

// Config are the configuration parameters of the aquahash.
type Config = ethashdag.Config

const (
	ModeNormal   = ethashdag.ModeNormal
	ModeShared   = ethashdag.ModeShared
	ModeTest     = ethashdag.ModeTest
	ModeFake     = ethashdag.ModeFake
	ModeFullFake = ethashdag.ModeFullFake
)

// Aquahash is a consensus engine based on proot-of-work implementing the aquahash
// algorithm.
type Aquahash struct {
	config *Config

	ethashdag *ethashdag.EthashDAG // if not-nil, uses same config pointer as config

	// Mining related fields
	threads int           // Number of threads to mine on if mining
	update  chan struct{} // Notification channel to update mining parameters

	// The fields below are hooks for testing
	shared    *Aquahash     // Shared PoW verifier to avoid cache regeneration
	fakeFail  uint64        // Block number which fails PoW check even in fake mode
	fakeDelay time.Duration // Time delay to sleep for before returning from verify

	lock sync.Mutex // Ensures thread safety for the in-memory caches and mining fields
}

func (aquahash *Aquahash) Name() string {
	return "aquahash"
}

// New creates a full sized aquahash PoW scheme.
func New(config *Config) *Aquahash {
	if config.StartVersion > 1 { // no need for caches or datasets
		return &Aquahash{
			config:    config,
			ethashdag: nil,
			update:    make(chan struct{}),
		}
	}
	log.Warn("using ethashdag", "config", common.ToJson(config))
	if config.CachesInMem <= 0 {
		log.Warn("One aquahash cache must always be in memory", "requested", config.CachesInMem)
		config.CachesInMem = 1
	}
	if config.CacheDir != "" && config.CachesOnDisk > 0 {
		log.Warn("Disk storage enabled for aquahash caches", "dir", config.CacheDir, "count", config.CachesOnDisk)
	}
	if config.DatasetDir != "" && config.DatasetsOnDisk > 0 {
		log.Warn("Disk storage enabled for aquahash DAGs", "dir", config.DatasetDir, "count", config.DatasetsOnDisk)
	}
	return &Aquahash{
		config:    config,
		ethashdag: ethashdag.New(config),
		update:    make(chan struct{}),
	}
}

// NewTester creates a small sized aquahash PoW scheme useful only for testing
// purposes.
func NewTester() *Aquahash {
	return New(&Config{CachesInMem: 1, PowMode: ModeTest})
}

// NewFaker creates a aquahash consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Aquachain
// consensus rules.
func NewFaker() *Aquahash {
	return &Aquahash{
		config: &Config{
			PowMode: ModeFake,
		},
	}
}

// NewFakeFailer creates a aquahash consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the Aquachain consensus rules.
func NewFakeFailer(fail uint64) *Aquahash {
	return &Aquahash{
		config: &Config{
			PowMode: ModeFake,
		},
		fakeFail: fail,
	}
}

// NewFakeDelayer creates a aquahash consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the Aquachain consensus rules.
func NewFakeDelayer(delay time.Duration) *Aquahash {
	return &Aquahash{
		config: &Config{
			PowMode: ModeFake,
		},
		fakeDelay: delay,
	}
}

// NewFullFaker creates an aquahash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *Aquahash {
	return &Aquahash{
		config: &Config{
			PowMode: ModeFullFake,
		},
	}
}

// sharedAquahash is a full instance that can be shared between multiple callers.
var sharedAquahash = New(&Config{CachesInMem: 3, DatasetsInMem: 1, PowMode: ModeNormal, StartVersion: 0})

// NewSharedTesting creates a full sized aquahash PoW shared between all requesters running
// in the same process.
func NewSharedTesting() *Aquahash {
	return &Aquahash{shared: sharedAquahash}
}

// Threads returns the number of mining threads currently enabled. This doesn't
// necessarily mean that mining is running!
func (aquahash *Aquahash) Threads() int {
	aquahash.lock.Lock()
	defer aquahash.lock.Unlock()

	return aquahash.threads
}

// SetThreads updates the number of mining threads currently enabled. Calling
// this method does not start mining, only sets the thread count. If zero is
// specified, the miner will use all cores of the machine. Setting a thread
// count below zero is allowed and will cause the miner to idle, without any
// work being done.
func (aquahash *Aquahash) SetThreads(threads int) {
	aquahash.lock.Lock()
	defer aquahash.lock.Unlock()

	// If we're running a shared PoW, set the thread count on that instead
	if aquahash.shared != nil {
		aquahash.shared.SetThreads(threads)
		return
	}
	// Update the threads and ping any running seal to pull in any changes
	aquahash.threads = threads
	select {
	case aquahash.update <- struct{}{}:
	default:
	}
}

// // Hashrate implements PoW, returning the measured rate of the search invocations
// // per second over the last minute.
// func (aquahash *Aquahash) Hashrate() float64 {
// 	return 9999.0
// 	// return aquahash.hashrate.Rate1()
// }

// APIs implements consensus.Engine, returning the user facing RPC APIs. Currently
// that is empty.
func (aquahash *Aquahash) APIs(chain consensus.ChainReader) []rpc.API {
	return nil
}

// SeedHash is the seed to use for generating a verification cache and the mining
// dataset.
func SeedHash(block uint64, version byte) []byte {
	return seedHash(block, version)
}
