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

package crypto

import (
	"bytes"

	"gitlab.com/aquachain/aquachain/common"
	oursha3 "gitlab.com/aquachain/aquachain/crypto/sha3"

	"golang.org/x/crypto/argon2"
)

// var (
// 	secp256k1_N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
// 	secp256k1_halfN = new(big.Int).Div(secp256k1_N, big.NewInt(2))
// )

func init() {
	if !secp256k1_N.ProbablyPrime(0) {
		panic("secp256k1_N is not prime")
	}
}

const (
	argonThreads uint8  = 1
	argonTime    uint32 = 1
)

const KnownVersion = 4

// VersionHash switch version, returns digest bytes, v is not hashed.
func VersionHash(v byte, data ...[]byte) []byte {
	switch v {
	//	case 0:
	//		return Keccak256(data...)
	case 1:
		return Keccak256(data...)
	case 2:
		return Argon2idA(data...)
	case 3:
		return Argon2idB(data...)
	case 4:
		return Argon2idC(data...)
	default:
		panic("invalid block version")
	}
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 1kb mem
func Argon2idA(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 1, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 16kb mem
func Argon2idB(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 16, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data, using 32kb
func Argon2idC(data ...[]byte) []byte {
	//fmt.Printf(".")
	buf := &bytes.Buffer{}
	for i := range data {
		buf.Write(data[i])
	}
	return argon2.IDKey(buf.Bytes(), nil, argonTime, 32, argonThreads, common.HashLength)
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idAHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idA(data...))
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idBHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idB(data...))
}

// Argon2id calculates and returns the Argon2id hash of the input data.
func Argon2idCHash(data ...[]byte) (h common.Hash) {
	return common.BytesToHash(Argon2idC(data...))
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	d := oursha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

var Keccak256 = oursha3.Keccak256
