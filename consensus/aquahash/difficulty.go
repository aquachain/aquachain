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
	"math/big"
	"os"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/math"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/params"
)

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big10         = big.NewInt(10)
	big240        = big.NewInt(240)
	bigMinus99    = big.NewInt(-99)
	big10000      = big.NewInt(10000)
)

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyStarting(time uint64, parent *types.Header, chainID uint64) *big.Int {
	// https://github.com/aquanetwork/EIPs/blob/master/EIPS/eip-2.md
	// algorithm:
	// diff = (parent_diff +
	//         (parent_diff / 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	//        ) + 2^(periodCount - 2)
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// testnet no minimum
	if chainID == params.MainnetChainConfig.ChainId.Uint64() {
		x = math.BigMax(x, params.MinimumDifficultyGenesis)
	}
	return x
}

// calcDifficultyHF1 is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses modified Homestead rules.
// It is flawed, target 10 seconds
func calcDifficultyHF1(time uint64, parent *types.Header, chainID uint64) *big.Int {
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big10)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 10, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}
	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(parent.Difficulty, params.DifficultyBoundDivisor)
	x.Mul(y, x)
	x.Add(parent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if chainID == params.MainnetChainConfig.ChainId.Uint64() {
		x = math.BigMax(x, params.MinimumDifficultyHF1)
	}
	return x
}

var _, fakedifficultymode = os.LookupEnv("FAKEPOWTEST") // will generate wrong blocks

// calcDifficultyHFX combines all difficulty algorithms
func calcDifficultyHFX(config *params.ChainConfig, time uint64, parent, grandparent *types.Header) *big.Int {
	if config == nil {
		panic("difficulty: chainConfig is nil")
	}
	var (
		next                   = new(big.Int).Add(parent.Number, big1)
		chainID                = config.ChainId.Uint64()
		adjust        *big.Int = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
		bigTime                = new(big.Int).SetUint64(time)
		bigParentTime          = new(big.Int).Set(parent.Time)
		limit                  = params.DurationLimit // not accurate, fixed in HF6
		min                    = params.MinimumDifficultyGenesis
		// mainnet       = params.MainnetChainConfig.ChainId.Uint64() == chainID // TODO: allow others
	)
	if fakedifficultymode {
		log.Warn("Fake difficulty mode!!!", "static-difficulty", params.MinimumDifficultyHF5)
		return params.MinimumDifficultyHF5
	}

	// fix min
	if config.IsHF(5, next) {
		min = params.MinimumDifficultyHF5
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF5)
	} else if config.IsHF(3, next) {
		min = params.MinimumDifficultyHF3
	} else if config.IsHF(1, next) {
		min = params.MinimumDifficultyHF1
	}
	if config.IsHF(6, next) {
		limit = params.DurationLimitHF6
	}

	// adjust
	if config.IsHF(8, next) {
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF8)
	} else if config.IsHF(6, next) {
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF6)
	}

	switch {
	case config.IsHF(10, next): // HF10: use grandparent
		return calcDifficultyGrandparent(time, parent, grandparent, config, chainID)
	case config.IsHF(8, next) && next.Cmp(config.GetHF(8)) == 0: // is this the HF8 fork block?
		log.Info("Activating Hardfork", "HF", 8, "BlockNumber", next.String())
		return params.MinimumDifficultyHF5 // difficulty reset for fork block
	case config.IsHF(6, next) && next.Cmp(config.GetHF(6)) == 0: // is this the HF6 fork block?
		log.Info("Activating Hardfork", "HF", 6, "BlockNumber", next.String())
	case config.IsHF(7, next) && next.Cmp(config.GetHF(7)) == 0: // is this the HF7 fork block?
		log.Info("Activating Hardfork", "HF", 7, "BlockNumber", next.String())
	case config.IsHF(5, next) && next.Cmp(config.GetHF(5)) == 0: // is this the HF5 fork block?
		// log.Info("Activating Hardfork", "HF", 5, "BlockNumber", next.String()) // already notified in StateProcesser
		return params.MinimumDifficultyHF5 // difficulty reset for fork block
	case config.IsHF(3, next) && next.Cmp(config.GetHF(3)) == 0: // is this the HF3 fork block?
		log.Info("Activating Hardfork", "HF", 3, "BlockNumber", next.String())
		return params.MinimumDifficultyHF3 // difficulty reset for fork block
	case config.IsHF(2, next) && next.Cmp(config.GetHF(2)) == 0: // is this the HF2 fork block?
		log.Info("Activating Hardfork", "HF", 2, "BlockNumber", next.String())
		// continue below
	case config.IsHF(2, next):
		// continue below
	case config.IsHF(1, next) && next.Cmp(config.GetHF(1)) == 0: // is this the HF1 fork block?
		log.Info("Activating Hardfork", "HF", 1, "BlockNumber", next.String())
		return params.MinimumDifficultyHF1 // difficulty reset for fork block
	case config.IsHF(1, next):
		return calcDifficultyHF1(time, parent, chainID)
	default: // no HF... first 30k blocks.
		return calcDifficultyStarting(time, parent, chainID)

	}
	// calculate difficulty using [adjust,min,limit]
	var diff = new(big.Int).Set(parent.Difficulty)
	if bigTime.Sub(bigTime, bigParentTime).Cmp(limit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}
	if diff.Cmp(min) < 0 {
		diff.Set(min)
	}
	return diff
}

// calcDifficultyGrandparent experimental
func calcDifficultyGrandparent(time uint64, parent, grandparent *types.Header, chaincfg *params.ChainConfig, chainID uint64) *big.Int {
	if grandparent == nil {
		log.Warn("calcDifficultyGrandparent: grandparent is nil, using parent difficulty")
		return new(big.Int).Set(parent.Difficulty)
	}
	bigGrandparentTime := new(big.Int).Set(grandparent.Time)
	bigParentTime := new(big.Int).Set(parent.Time)
	if bigParentTime.Cmp(bigGrandparentTime) <= 0 {
		panic("invalid code")
	}
	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	divisor := params.DifficultyBoundDivisorHF5
	if chaincfg.IsHF(8, parent.Number) {
		divisor = params.DifficultyBoundDivisorHF8
	}
	// 1 - (block_timestamp - parent_timestamp) // 240
	x.Sub(bigParentTime, bigGrandparentTime)
	x.Div(x, big240)
	x.Sub(big1, x)

	// max(1 - (block_timestamp - parent_timestamp) // 240, -99)
	if x.Cmp(bigMinus99) < 0 {
		x.Set(bigMinus99)
	}

	// (parent_diff + parent_diff // 2048 * max(1 - (block_timestamp - parent_timestamp) // 10, -99))
	y.Div(grandparent.Difficulty, divisor)
	x.Mul(y, x)
	x.Add(grandparent.Difficulty, x)

	// minimum difficulty can ever be (before exponential factor)
	if chainID == params.MainnetChainConfig.ChainId.Uint64() {
		x = math.BigMax(params.MinimumDifficultyHF5, x)
	} else {
		x = math.BigMax(params.MinimumDifficultyHF5Testnet, x)
	}
	return x
}

var big100 = big.NewInt(100)
var big1000 = big.NewInt(1000)
var big20 = big.NewInt(20)

func calcDifficultyTestnet3(time uint64, parent, grandparent *types.Header) *big.Int {
	if grandparent == nil {
		return new(big.Int).Set(parent.Difficulty)
	}
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)
	bigGParentTime := new(big.Int).Set(grandparent.Time)
	difference := new(big.Int).Sub(bigTime, bigParentTime)
	gdifference := new(big.Int).Sub(bigGParentTime, bigParentTime)
	if difference.Cmp(big10) < 0 && gdifference.Cmp(big10) < 0 {
		return new(big.Int).Add(parent.Difficulty, big1000)
	}
	if difference.Cmp(big20) > 0 && gdifference.Cmp(big20) > 0 {
		return new(big.Int).Sub(parent.Difficulty, big1000)
	}
	if difference.Cmp(big100) > 0 && gdifference.Cmp(big100) > 0 {
		return new(big.Int).Quo(parent.Difficulty, big2)
	}
	return new(big.Int).Set(parent.Difficulty)
}
