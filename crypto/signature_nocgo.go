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
	if len(sig) != 65 {
		return nil, fmt.Errorf("signature length is not 65 bytes (%d)", len(sig))
	}
	// Convert to btcec input format with 'recovery id' v at the beginning.
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)
	pub, _, err := becdsa.RecoverCompact(btcsig, hash)
	if err != nil {
		return nil, fmt.Errorf("SigToPub failed to recover public key: %v", err)
	}
	if !pub.IsOnCurve() {
		return nil, errors.New("recovered public key is not on the curve")
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
	if v != 0 && v != 1 {
		return nil, fmt.Errorf("invalid signature recovery id: %d", v)
	}
	copy(sig, sig[1:])
	sig[64] = v
	return sig, nil
}

// VerifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func VerifySignature(pubkey, hash, signature []byte) bool {
	if len(signature) != 64 {
		return false
	}
	var r, s btcec.ModNScalar
	// check if truncated
	if r.SetByteSlice(signature[:32]) || s.SetByteSlice(signature[32:]) {
		return false
	}
	key, err := btcec.ParsePubKey(pubkey)
	if err != nil {
		return false
	}
	// Reject malleable signatures. libsecp256k1 does this check but decred doesn't.
	if s.IsOverHalfOrder() {
		return false
	}
	sig := becdsa.NewSignature(&r, &s)
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
