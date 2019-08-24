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

package abi

import (
	"math/big"
	"reflect"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/math"
)

var (
	big_t      = reflect.TypeOf(&big.Int{})
	derefbig_t = reflect.TypeOf(big.Int{})
	uint8_t    = reflect.TypeOf(uint8(0))
	uint16_t   = reflect.TypeOf(uint16(0))
	uint32_t   = reflect.TypeOf(uint32(0))
	uint64_t   = reflect.TypeOf(uint64(0))
	int8_t     = reflect.TypeOf(int8(0))
	int16_t    = reflect.TypeOf(int16(0))
	int32_t    = reflect.TypeOf(int32(0))
	int64_t    = reflect.TypeOf(int64(0))
	address_t  = reflect.TypeOf(common.Address{})
)

// U256 converts a big Int into a 256bit EVM number.
func U256(n *big.Int) []byte {
	return math.PaddedBigBytes(math.U256(n), 32)
}
