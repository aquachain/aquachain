package aquahash

import (
	"math/big"

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
)

// calcDifficultyHomestead is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses the Homestead rules.
func calcDifficultyHomestead(time uint64, parent *types.Header, chainID uint64) *big.Int {
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

// calcDifficultyX is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time given the
// parent block's time and difficulty. The calculation uses modified Homestead rules.
// It is flawed, target 10 seconds
func calcDifficultyX(time uint64, parent *types.Header, hf int, chainID uint64) *big.Int {
	bigTime := new(big.Int).SetUint64(time)
	bigParentTime := new(big.Int).Set(parent.Time)

	// holds intermediate values to make the algo easier to read & audit
	x := new(big.Int)
	y := new(big.Int)

	// 1 - (block_timestamp - parent_timestamp) // 10
	x.Sub(bigTime, bigParentTime)
	x.Div(x, big240)
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
func calcDifficultyHF6(time uint64, parent *types.Header, hf int, chainID uint64) *big.Int {
	return calcDifficultyHFX(time, parent, hf, chainID)
}

func calcDifficultyHFX(time uint64, parent *types.Header, hf int, chainID uint64) *big.Int {
	var (
		diff          = new(big.Int)
		adjust        *big.Int
		bigTime       = new(big.Int)
		bigParentTime = new(big.Int)
		limit         = params.DurationLimitHF6 // target 240 seconds
		min           = params.MinimumDifficultyHF5
		mainnet       = params.MainnetChainConfig.ChainId.Uint64() == chainID // bool
	)

	switch hf {
	case 10:
		return calcDifficultyX(time, parent, hf, chainID)
	case 9:
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF9)
	case 8:
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF8)
	case 6, 7:
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF6)
	case 5:
		limit = params.DurationLimit // not accurate, fixed in hf6
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisorHF5)
	case 3:
		limit = params.DurationLimit // not accurate, fixed in hf6
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
		min = params.MinimumDifficultyHF3
	case 2:
		limit = params.DurationLimit // not accurate, fixed in hf6
		adjust = new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
		min = params.MinimumDifficultyHF1
	case 1:
		return calcDifficultyHF1(time, parent, chainID)
	case 0:
		return calcDifficultyHomestead(time, parent, chainID)
	default:
		panic("calculating difficulty fail")
	}

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	// calculate difficulty
	if bigTime.Sub(bigTime, bigParentTime).Cmp(limit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}

	if mainnet && diff.Cmp(min) < 0 {
		diff.Set(min)
	}

	return diff
}
