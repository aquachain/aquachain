package sha3

import (
	"hash"

	"golang.org/x/crypto/sha3"
)

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
func NewKeccak256() hash.Hash {
	return sha3.NewLegacyKeccak256()
}

// Keccak512 calculates and returns the Keccak512 hash of the input data.
//
// only used for ethash
func Keccak512(data ...[]byte) []byte {
	d := NewKeccak512Hasher()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

func NewKeccak512Hasher() hash.Hash {
	return sha3.NewLegacyKeccak512()
}
