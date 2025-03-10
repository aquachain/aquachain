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
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/common"
)

var testAddrHex = "970e8128ab834e8eac17ab8e3812f010678cf791"
var testPrivHex = "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.
func TestKeccak256Hash(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256-array", func(in []byte) []byte { h := Keccak256Hash(in); return h[:] }, msg, exp)
}

func TestToECDSAErrors(t *testing.T) {
	if _, err := HexToECDSA("0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Fatal("HexToECDSA 0000 should've returned error")
	}
	if _, err := HexToECDSA("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"); err == nil {
		t.Fatal("HexToECDSA ffff should've returned error")
	}
}

func TestArgon2id(t *testing.T) {
	a := []byte("aquachain")
	fmt.Printf("\nArgonated: 0x%x\n", Argon2idA(a))
	hex2hash := common.HexToHash(fmt.Sprintf("0x%x", Argon2idA(a)))
	if hex2hash.Hex() != "0x1090f3198e771bbe9b3fc9f1291ce5a4744d796e0f099aa02cb681ede75209c0" {
		t.Errorf("argon2 params mismatch")
	}
	bytes2hash := common.BytesToHash(Argon2idA(a))
	if hex2hash.Big().Cmp(bytes2hash.Big()) != 0 {
		t.Errorf("Unequal")
	}
	if !bytes.Equal(hex2hash.Bytes(), bytes2hash.Bytes()) {
		t.Errorf("Unequal")
	}
	fmt.Printf("Argonated: 0x%x\n", Argon2idA(Argon2idA(a)))
	fmt.Printf("Argonated: 0x%x\n", Argon2idA(Argon2idA(Argon2idA(a))))
	fmt.Printf("Argonated: 0x%x\n", Argon2idA(Argon2idA(Argon2idA(Argon2idA(a)))))
	fmt.Printf("Argonated: 0x%x\n", Argon2idA(Argon2idA(Argon2idA(Argon2idA(Argon2idA(a))))))
	fmt.Printf("Argonated: 0x%x\n", Argon2idB(a))
	fmt.Printf("Argonated: 0x%x\n", Argon2idC(a))
}
func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Keccak256(a)
	}
}
func BenchmarkArgon2idA(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Argon2idA(a)
	}
}
func BenchmarkArgon2idB(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Argon2idB(a)
	}
}
func BenchmarkArgon2idC(b *testing.B) {
	a := []byte("hello world")
	for i := 0; i < b.N; i++ {
		Argon2idC(a)
	}
}

func TestSign(t *testing.T) {
	key, err := HexToBtcec(testPrivHex)
	if err != nil {
		t.Errorf("HexToECDSA error: %s", err)
		return
	}
	addr := common.HexToAddress(testAddrHex)
	addr2 := PubkeyToAddress(key.PubKey())
	if addr != addr2 {
		t.Errorf("Address mismatch: want: %s have: %s", addr.Hex(), addr2.Hex())
	}
	if strings.ToLower(addr.Hex()[2:]) != testAddrHex {
		t.Errorf("Address mismatch: want: %s have: %s", addr.Hex()[2:], testAddrHex)
	}
	msg := Keccak256([]byte("foo"))
	sig, err := Sign(msg, key)
	if err != nil {
		t.Errorf("Sign error: %s", err)
		return
	}
	recoveredPub, err := Ecrecover(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	pubKey := ToECDSAPub(recoveredPub)
	recoveredAddr := PubkeyToAddress(pubKey)
	if addr != recoveredAddr {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr)
	}

	// should be equal to SigToPub
	recoveredPub2, err := SigToPub(msg, sig)
	if err != nil {
		t.Errorf("ECRecover error: %s", err)
	}
	recoveredAddr2 := PubkeyToAddress(recoveredPub2)
	if addr != recoveredAddr2 {
		t.Errorf("Address mismatch: want: %x have: %x", addr, recoveredAddr2)
	}
}

func TestInvalidSign(t *testing.T) {
	if _, err := Sign(make([]byte, 1), nil); err == nil {
		t.Errorf("expected sign with hash 1 byte to error")
	}
	if _, err := Sign(make([]byte, 33), nil); err == nil {
		t.Errorf("expected sign with hash 33 byte to error")
	}
}

func TestNewContractAddress(t *testing.T) {
	key, _ := HexToBtcec(testPrivHex)
	addr := common.HexToAddress(testAddrHex)
	genAddr := PubkeyToAddress(key.PubKey())
	// sanity check before using addr to create contract address
	checkAddr(t, genAddr, addr)

	caddr0 := CreateAddress(addr, 0)
	caddr1 := CreateAddress(addr, 1)
	caddr2 := CreateAddress(addr, 2)
	checkAddr(t, common.HexToAddress("333c3310824b7c685133f2bedb2ca4b8b4df633d"), caddr0)
	checkAddr(t, common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"), caddr1)
	checkAddr(t, common.HexToAddress("c9ddedf451bc62ce88bf9292afb13df35b670699"), caddr2)
}

func TestLoadECDSAFile(t *testing.T) {
	keyBytes := common.FromHex(testPrivHex)
	fileName0 := "test_key0"
	fileName1 := "test_key1"
	checkKey := func(k *btcec.PrivateKey) {
		checkAddr(t, PubkeyToAddress(k.PubKey()), common.HexToAddress(testAddrHex))
		loadedKeyBytes := FromECDSA(k)
		if !bytes.Equal(loadedKeyBytes, keyBytes) {
			t.Fatalf("private key mismatch: want: %x have: %x", keyBytes, loadedKeyBytes)
		}
	}

	ioutil.WriteFile(fileName0, []byte(testPrivHex), 0600)
	defer os.Remove(fileName0)

	key0, err := LoadECDSA(fileName0)
	if err != nil {
		t.Fatal(err)
	}
	checkKey(key0)

	// again, this time with SaveECDSA instead of manual save:
	err = SaveECDSA(fileName1, key0)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(fileName1)

	key1, err := LoadECDSA(fileName1)
	if err != nil {
		t.Fatal(err)
	}
	checkKey(key1)
}

func TestValidateSignatureValues(t *testing.T) {
	check := func(expected bool, v byte, r, s *big.Int) {
		if ValidateSignatureValues(v, r, s, false) != expected {
			t.Errorf("mismatch for v: %d r: %d s: %d want: %v", v, r, s, expected)
		}
	}
	minusOne := big.NewInt(-1)
	one := common.Big1
	zero := common.Big0
	secp256k1nMinus1 := new(big.Int).Sub(secp256k1_N, common.Big1)

	// correct v,r,s
	check(true, 0, one, one)
	check(true, 1, one, one)
	// incorrect v, correct r,s,
	check(false, 2, one, one)
	check(false, 3, one, one)

	// incorrect v, combinations of incorrect/correct r,s at lower limit
	check(false, 2, zero, zero)
	check(false, 2, zero, one)
	check(false, 2, one, zero)
	check(false, 2, one, one)

	// correct v for any combination of incorrect r,s
	check(false, 0, zero, zero)
	check(false, 0, zero, one)
	check(false, 0, one, zero)

	check(false, 1, zero, zero)
	check(false, 1, zero, one)
	check(false, 1, one, zero)

	// correct sig with max r,s
	check(true, 0, secp256k1nMinus1, secp256k1nMinus1)
	// correct v, combinations of incorrect r,s at upper limit
	check(false, 0, secp256k1_N, secp256k1nMinus1)
	check(false, 0, secp256k1nMinus1, secp256k1_N)
	check(false, 0, secp256k1_N, secp256k1_N)

	// current callers ensures r,s cannot be negative, but let's test for that too
	// as crypto package could be used stand-alone
	check(false, 0, minusOne, one)
	check(false, 0, one, minusOne)
}

func checkhash(t *testing.T, name string, f func([]byte) []byte, msg, exp []byte) {
	sum := f(msg)
	if !bytes.Equal(exp, sum) {
		t.Fatalf("hash %s mismatch: want: %x have: %x", name, exp, sum)
	}
}

func checkAddr(t *testing.T, addr0, addr1 common.Address) {
	if addr0 != addr1 {
		t.Fatalf("address mismatch: want: %x have: %x", addr0, addr1)
	}
}

// test to help Python team with integration of libsecp256k1
// skip but keep it after they are done
func TestPythonIntegration(t *testing.T) {
	kh := "289c2857d4598e37fb9647507e47a309d6133539bf21a8b9cb6df88fd5232032"
	k0, _ := HexToBtcec(kh)

	msg0 := Keccak256([]byte("foo"))
	sig0, _ := Sign(msg0, k0)

	msg1 := common.FromHex("00000000000000000000000000000000")
	sig1, _ := Sign(msg0, k0)

	t.Logf("msg: %x, privkey: %s sig: %x\n", msg0, kh, sig0)
	t.Logf("msg: %x, privkey: %s sig: %x\n", msg1, kh, sig1)
}
