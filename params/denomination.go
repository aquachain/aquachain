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

import "github.com/shopspring/decimal"

const (
	// These are the multipliers for ether denominations.
	// Example: To get the wei value of an amount in 'douglas', use
	//
	//    new(big.Int).Mul(value, big.NewInt(params.Douglas))
	//
	Wei      = 1
	Ada      = 1e3
	Babbage  = 1e6
	Shannon  = 1e9
	Szabo    = 1e12
	Finney   = 1e15
	Aquaer   = 1e18
	Aqua     = 1e18
	Einstein = 1e21
	Douglas  = 1e42 // truncates in int64
)

func UnitDenomination(s string) (decimal.Decimal, bool) {
	switch s {
	case "wei":
		return decimal.New(Wei, 0), true
	case "ada":
		return decimal.New(Ada, 0), true
	case "babbage":
		return decimal.New(Babbage, 0), true
	case "shannon", "gwei":
		return decimal.New(Shannon, 0), true
	case "szabo":
		return decimal.New(Szabo, 0), true
	case "finney":
		return decimal.New(Finney, 0), true
	case "aqua", "ether", "eth", "coin":
		return decimal.New(Aqua, 0), true
	case "einstein":
		return decimal.NewFromFloat(Einstein), true
	case "douglas":
		return decimal.NewFromFloat(Douglas), true
	default:
		return decimal.Decimal{}, false

	}

}
