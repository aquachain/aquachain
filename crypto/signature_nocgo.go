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
	"crypto/elliptic"
	"errors"
	"fmt"
	"log"

	"github.com/btcsuite/btcd/btcec/v2"
	becdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := SigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := pub.SerializeUncompressed()
	return bytes, err
}

// SigToPub returns the public key that created the given signature.
func SigToPub(hash, sig []byte) (*btcec.PublicKey, error) {
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)
	pub, _, err := becdsa.RecoverCompact(btcsig, hash)
	if err != nil {
		return nil, fmt.Errorf("SigToPub failed to recover public key: %v", err)
	}
	return pub, nil

}

// Sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func Sign(hash []byte, prv0 *btcec.PrivateKey) ([]byte, error) {
	if prv0 == nil {
		return nil, errors.New("private key is nil")
	}
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	// prv := prv0.ToECDSA()
	// if prv.Curve != btcec.S256() {
	// 	return nil, fmt.Errorf("private key curve is not secp256k1")
	// }
	sig := becdsa.SignCompact(prv0, hash, false)
	if len(sig) != 65 {
		return nil, fmt.Errorf("signature length is not 65 bytes (%d)", len(sig))
	}
	// Convert to Ethereum signature format with 'recovery id' v at the end.
	v := sig[0] - 27
	log.Printf("v: %d", v)
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

// Errors returned by canonicalPadding.
var (
	errNegativeValue          = errors.New("value may be interpreted as negative")
	errExcessivelyPaddedValue = errors.New("value is excessively padded")
)

// canonicalPadding checks whether a big-endian encoded integer could
// possibly be misinterpreted as a negative number (even though OpenSSL
// treats all numbers as unsigned), or if there is any unnecessary
// leading zero padding.
func canonicalPadding(b []byte) error {
	switch {
	case b[0]&0x80 == 0x80:
		return errNegativeValue
	case len(b) > 1 && b[0] == 0x00 && b[1]&0x80 != 0x80:
		return errExcessivelyPaddedValue
	default:
		return nil
	}
}

const (
	// MinSigLen is the minimum length of a DER encoded signature and is when both R
	// and S are 1 byte each.
	// 0x30 + <1-byte> + 0x02 + 0x01 + <byte> + 0x2 + 0x01 + <byte>
	MinSigLen = 8

	// MaxSigLen is the maximum length of a DER encoded signature and is
	// when both R and S are 33 bytes each.  It is 33 bytes because a
	// 256-bit integer requires 32 bytes and an additional leading null byte
	// might be required if the high bit is set in the value.
	//
	// 0x30 + <1-byte> + 0x02 + 0x21 + <33 bytes> + 0x2 + 0x21 + <33 bytes>
	MaxSigLen = 72
)

func parseSig(sigStr []byte, der bool) (*becdsa.Signature, error) {
	// Originally this code used encoding/asn1 in order to parse the
	// signature, but a number of problems were found with this approach.
	// Despite the fact that signatures are stored as DER, the difference
	// between go's idea of a bignum (and that they have sign) doesn't agree
	// with the openssl one (where they do not). The above is true as of
	// Go 1.1. In the end it was simpler to rewrite the code to explicitly
	// understand the format which is this:
	// 0x30 <length of whole message> <0x02> <length of R> <R> 0x2
	// <length of S> <S>.

	// The signature must adhere to the minimum and maximum allowed length.
	totalSigLen := len(sigStr)
	if totalSigLen < MinSigLen {
		return nil, errors.New("malformed signature: too short")
	}
	if der && totalSigLen > MaxSigLen {
		return nil, errors.New("malformed signature: too long")
	}

	// 0x30
	index := 0
	if sigStr[index] != 0x30 {
		return nil, fmt.Errorf("malformed signature: no header magic (got %02x want 0x30)", sigStr[index])
	}
	index++
	// length of remaining message
	siglen := sigStr[index]
	index++

	// siglen should be less than the entire message and greater than
	// the minimal message size.
	if int(siglen+2) > len(sigStr) || int(siglen+2) < MinSigLen {
		return nil, errors.New("malformed signature: bad length")
	}
	// trim the slice we're working on so we only look at what matters.
	sigStr = sigStr[:siglen+2]

	// 0x02
	if sigStr[index] != 0x02 {
		return nil,
			errors.New("malformed signature: no 1st int marker")
	}
	index++

	// Length of signature R.
	rLen := int(sigStr[index])
	// must be positive, must be able to fit in another 0x2, <len> <s>
	// hence the -3. We assume that the length must be at least one byte.
	index++
	if rLen <= 0 || rLen > len(sigStr)-index-3 {
		return nil, errors.New("malformed signature: bogus R length")
	}

	// Then R itself.
	rBytes := sigStr[index : index+rLen]
	if der {
		switch err := canonicalPadding(rBytes); err {
		case errNegativeValue:
			return nil, errors.New("signature R is negative")
		case errExcessivelyPaddedValue:
			return nil, errors.New("signature R is excessively padded")
		}
	}

	// Strip leading zeroes from R.
	for len(rBytes) > 0 && rBytes[0] == 0x00 {
		rBytes = rBytes[1:]
	}

	// R must be in the range [1, N-1].  Notice the check for the maximum number
	// of bytes is required because SetByteSlice truncates as noted in its
	// comment so it could otherwise fail to detect the overflow.
	var r btcec.ModNScalar
	if len(rBytes) > 32 {
		str := "invalid signature: R is larger than 256 bits"
		return nil, errors.New(str)
	}
	if overflow := r.SetByteSlice(rBytes); overflow {
		str := "invalid signature: R >= group order"
		return nil, errors.New(str)
	}
	if r.IsZero() {
		str := "invalid signature: R is 0"
		return nil, errors.New(str)
	}
	index += rLen
	// 0x02. length already checked in previous if.
	if sigStr[index] != 0x02 {
		return nil, errors.New("malformed signature: no 2nd int marker")
	}
	index++

	// Length of signature S.
	sLen := int(sigStr[index])
	index++
	// S should be the rest of the string.
	if sLen <= 0 || sLen > len(sigStr)-index {
		return nil, errors.New("malformed signature: bogus S length")
	}

	// Then S itself.
	sBytes := sigStr[index : index+sLen]
	if der {
		switch err := canonicalPadding(sBytes); err {
		case errNegativeValue:
			return nil, errors.New("signature S is negative")
		case errExcessivelyPaddedValue:
			return nil, errors.New("signature S is excessively padded")
		}
	}

	// Strip leading zeroes from S.
	for len(sBytes) > 0 && sBytes[0] == 0x00 {
		sBytes = sBytes[1:]
	}

	// S must be in the range [1, N-1].  Notice the check for the maximum number
	// of bytes is required because SetByteSlice truncates as noted in its
	// comment so it could otherwise fail to detect the overflow.
	var s btcec.ModNScalar
	if len(sBytes) > 32 {
		str := "invalid signature: S is larger than 256 bits"
		return nil, errors.New(str)
	}
	if overflow := s.SetByteSlice(sBytes); overflow {
		str := "invalid signature: S >= group order"
		return nil, errors.New(str)
	}
	if s.IsZero() {
		str := "invalid signature: S is 0"
		return nil, errors.New(str)
	}
	index += sLen

	// sanity check length parsing
	if index != len(sigStr) {
		return nil, fmt.Errorf("malformed signature: bad final length %v != %v",
			index, len(sigStr))
	}

	return becdsa.NewSignature(&r, &s), nil
}

// VerifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func VerifySignature(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}
	sig, err := becdsa.ParseSignature(signature)
	if err != nil {
		log.Printf("signature parse error: %v", err)
		return false
	}

	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return false
	}
	// // Reject malleable signatures. libsecp256k1 does this check but btcec doesn't.
	// if sig.S.Cmp(secp256k1_halfN) > 0 {
	// 	return false
	// }
	return sig.Verify(hash, key)
}

// DecompressPubkey parses a public key in the 33-byte compressed format.
func DecompressPubkey(pubkey []byte) (*btcec.PublicKey, error) {
	if len(pubkey) != 33 {
		return nil, errors.New("invalid compressed public key length")
	}
	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// CompressPubkey encodes a public key to the 33-byte compressed format.
func CompressPubkey(pubkey *btcec.PublicKey) []byte {
	return pubkey.SerializeCompressed()
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return btcec.S256()
}
