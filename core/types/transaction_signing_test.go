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

package types

import (
	"errors"
	"log"
	"math/big"
	"testing"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rlp"
)

func TestEIP155Signing(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PubKey())

	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatal(err)
	}
	if from != addr {
		t.Errorf("exected from and address to be equal. Got %x want %x", from, addr)
	}
}

func TestEIP155ChainIdNil(t *testing.T) {
	if !func() (ok bool) {
		defer func() {
			if x := recover(); x != nil {
				ok = true
			}
			return
		}()
		signer := NewEIP155Signer(nil)
		_ = signer
		return

	}() {
		t.Error("expected panic for nil chainId, but did not panic")
	}

}
func TestEIP155ChainId(t *testing.T) {
	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PubKey())
	log.Printf("addr: %s", addr.Hex())
	signer := NewEIP155Signer(big.NewInt(18))
	tx, err := SignTx(NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil), signer, key)
	if err != nil {
		t.Fatal(err)
	}
	if !tx.Protected() {
		t.Fatal("expected tx to be protected")
	}

	if tx.ChainId().Cmp(signer.chainId) != 0 {
		t.Error("expected chainId to be", signer.chainId, "got", tx.ChainId())
	}

	from, err := signer.Sender(tx)
	if err != nil {
		t.Fatalf("unexpected error retrieving signer: %v", err)
	}
	if from != addr {
		t.Error("expected from to be", addr, "got", from)
	}

	tx = NewTransaction(0, addr, new(big.Int), 0, new(big.Int), nil)
	tx, err = SignTx(tx, HomesteadSigner{}, key)
	if err != nil {
		t.Fatal(err)
	}

	if tx.Protected() {
		t.Error("didn't expect tx to be protected")
	}

	if tx.ChainId().Sign() != 0 {
		t.Error("expected chain id to be 0 got", tx.ChainId())
	}
	from, err = HomesteadSigner{}.Sender(tx)
	if err != nil {
		t.Fatalf("unexpected error retrieving signer: %v", err)
	}
	if from != addr {
		t.Error("expected from to be", addr, "got", from)
	}
}

func TestEIP155SigningVitalik(t *testing.T) {
	// Test vectors come from http://vitalik.ca/files/eip155_testvec.txt
	for i, test := range []struct {
		txRlp, addr string
	}{
		{"f864808504a817c800825208943535353535353535353535353535353535353535808025a0044852b2a670ade5407e78fb2863c51de9fcb96542a07186fe3aeda6bb8a116da0044852b2a670ade5407e78fb2863c51de9fcb96542a07186fe3aeda6bb8a116d", "0xf0f6f18bca1b28cd68e4357452947e021241e9ce"},
		{"f864018504a817c80182a410943535353535353535353535353535353535353535018025a0489efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bcaa0489efdaa54c0f20c7adf612882df0950f5a951637e0307cdcb4c672f298b8bc6", "0x23ef145a395ea3fa3deb533b8a9e1b4c6c25d112"},
		{"f864028504a817c80282f618943535353535353535353535353535353535353535088025a02d7c5bef027816a800da1736444fb58a807ef4c9603b7848673f7e3a68eb14a5a02d7c5bef027816a800da1736444fb58a807ef4c9603b7848673f7e3a68eb14a5", "0x2e485e0c23b4c3c542628a5f672eeab0ad4888be"},
		{"f865038504a817c803830148209435353535353535353535353535353535353535351b8025a02a80e1ef1d7842f27f2e6be0972bb708b9a135c38860dbe73c27c3486c34f4e0a02a80e1ef1d7842f27f2e6be0972bb708b9a135c38860dbe73c27c3486c34f4de", "0x82a88539669a3fd524d669e858935de5e5410cf0"},
		{"f865048504a817c80483019a28943535353535353535353535353535353535353535408025a013600b294191fc92924bb3ce4b969c1e7e2bab8f4c93c3fc6d0a51733df3c063a013600b294191fc92924bb3ce4b969c1e7e2bab8f4c93c3fc6d0a51733df3c060", "0xf9358f2538fd5ccfeb848b64a96b743fcc930554"},
		{"f865058504a817c8058301ec309435353535353535353535353535353535353535357d8025a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c1a04eebf77a833b30520287ddd9478ff51abbdffa30aa90a8d655dba0e8a79ce0c1", "0xa8f7aba377317440bc5b26198a363ad22af1f3a4"},
		{"f866068504a817c80683023e3894353535353535353535353535353535353535353581d88025a06455bf8ea6e7463a1046a0b52804526e119b4bf5136279614e0b1e8e296a4e2fa06455bf8ea6e7463a1046a0b52804526e119b4bf5136279614e0b1e8e296a4e2d", "0xf1f571dc362a0e5b2696b8e775f8491d3e50de35"},
		{"f867078504a817c807830290409435353535353535353535353535353535353535358201578025a052f1a9b320cab38e5da8a8f97989383aab0a49165fc91c737310e4f7e9821021a052f1a9b320cab38e5da8a8f97989383aab0a49165fc91c737310e4f7e9821021", "0xd37922162ab7cea97c97a87551ed02c9a38b7332"},
		{"f867088504a817c8088302e2489435353535353535353535353535353535353535358202008025a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c12a064b1702d9298fee62dfeccc57d322a463ad55ca201256d01f62b45b2e1c21c10", "0x9bddad43f934d313c2b79ca28a432dd2b7281029"},
		{"f867098504a817c809830334509435353535353535353535353535353535353535358202d98025a052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afba052f8f61201b2b11a78d6e866abc9c3db2ae8631fa656bfe5cb53668255367afb", "0x3c24d7329e92f84f08556ceb6df1cdb0104ca49f"},
	} {
		signer := NewEIP155Signer(big.NewInt(1))

		var tx *Transaction
		err := rlp.DecodeBytes(common.Hex2Bytes(test.txRlp), &tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		from, err := Sender(signer, tx)
		if err != nil {
			t.Errorf("%d: %v", i, err)
			continue
		}

		addr := common.HexToAddress(test.addr)
		if from != addr {
			t.Errorf("%d: expected %x got %x", i, addr, from)
		}

	}
}

func TestChainId(t *testing.T) {
	key, _ := defaultTestKey()

	tx := NewTransaction(0, common.Address{}, new(big.Int), 0, new(big.Int), nil)

	var err error
	tx, err = SignTx(tx, NewEIP155Signer(big.NewInt(1)), key)
	if err != nil {
		t.Fatal(err)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(2)), tx)
	if !errors.Is(err, ErrInvalidChainId) {
		t.Errorf("expected error: %v got %v", ErrInvalidChainId, err)
	}

	_, err = Sender(NewEIP155Signer(big.NewInt(1)), tx)
	if err != nil {
		t.Error("expected no error")
	}
}

func TestRecoverPaper(t *testing.T) {
	//e6d3285a7082f22916b290f7d1afbead14358e3ffdcacc7a72435fafb16c22c3 0x20116804405Ae3bbb05aDe6Bb57E7f97bE546b82
	key, err := crypto.BytesToKey(common.Hex2Bytes("e6d3285a7082f22916b290f7d1afbead14358e3ffdcacc7a72435fafb16c22c3"))
	if err != nil {
		t.Fatal(err)
	}
	addr := crypto.PubkeyToAddress(key.PubKey())
	if addr.Hex() != "0x20116804405Ae3bbb05aDe6Bb57E7f97bE546b82" {
		t.Fatalf("expected 0x20116804405Ae3bbb05aDe6Bb57E7f97bE546b82 got %s", addr.Hex())
	}

	tx := newTransaction(1, &addr, big.NewInt(1000000000000000000), 21000, big.NewInt(1000000000), nil)
	signer := NewEIP155Signer(big.NewInt(61717561))
	tx, err = SignTx(tx, signer, key)
	if err != nil {
		t.Fatal(err)
	}

	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatal(err)
	}

	if from != addr {
		t.Errorf("expected %x got %x", addr, from)
	}

}
func TestOtherInValid(t *testing.T) {
	t.SkipNow()
	/*
		{
			from: "0xbbc1595b297e6330d91561bd6a3d23901822ff4b",
			gas: 21000,
			gasPrice: 1000000000,
			nonce: 1,
			to: "0xbbc1595b297e6330d91561bd6a3d23901822ff4b",
			value: "1000000000000000000"
		  }
		 {
			raw: "0xf86f01843b9aca0082520894bbc1595b297e6330d91561bd6a3d23901822ff4b880de0b6b3a76400008084075b7896a0661499471a5eaed8e3517199afc4c30743df4352c52f1862ea6e65e5a5e3a7e9a056bcd9fb01b03bc7c918e352c4ce0c56d97cfa4206e4196852e11a4767cb9aa8",
			tx: {
			  gas: "0x5208",
			  gasPrice: "0x3b9aca00",
			  hash: "0x7feaff7ef852f6b1dae12b847b087732e33f572f8ec84a3e17cc76aab5e377be",
			  input: "0x",
			  nonce: "0x1",
			  r: "0x661499471a5eaed8e3517199afc4c30743df4352c52f1862ea6e65e5a5e3a7e9",
			  s: "0x56bcd9fb01b03bc7c918e352c4ce0c56d97cfa4206e4196852e11a4767cb9aa8",
			  to: "0xbbc1595b297e6330d91561bd6a3d23901822ff4b",
			  v: "0x75b7896",
			  value: "0xde0b6b3a7640000"
			}
		  }
	*/

	raw := "f86f01843b9aca0082520894bbc1595b297e6330d91561bd6a3d23901822ff4b880de0b6b3a76400008084075b7896a0661499471a5eaed8e3517199afc4c30743df4352c52f1862ea6e65e5a5e3a7e9a056bcd9fb01b03bc7c918e352c4ce0c56d97cfa4206e4196852e11a4767cb9aa8"
	tx := new(Transaction)
	if err := rlp.DecodeBytes(common.Hex2Bytes(raw), tx); err != nil {
		t.Fatalf("decoding tx: %v", err)
	}

	log.Printf("parsed tx: %s", tx.String()) // should cache the sender
	if tx.Hash().Hex() != "0x7feaff7ef852f6b1dae12b847b087732e33f572f8ec84a3e17cc76aab5e377be" {
		t.Fatalf("hash mismatch: want: %s have: %s", "0x7feaff7ef852f6b1dae12b847b087732e33f572f8ec84a3e17cc76aab5e377be", tx.Hash().Hex())
	}
	signer := NewEIP155Signer(params.MainnetChainConfig.ChainId)
	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatalf("unexpected sender error: %v", err)
	}
	if from.Hex() != "0xBBC1595b297E6330d91561bD6A3d23901822ff4b" {
		t.Fatalf("expected sender %s got %s", "0xBBC1595b297E6330d91561bD6A3d23901822ff4b", from.Hex())
	}

	if from != *tx.To() {
		t.Errorf("expected sender %s got %s", tx.To().Hex(), from.Hex())
	}
	// tx := NewTransaction(1, common.HexToAddress("0xbbc1595b297e6330d91561bd6a3d23901822ff4b"), big.NewInt(1000000000000000000), 21000, big.NewInt(1000000000), nil)
	// tx.WithSignature(signer, sig)
	// tx, err = SignTx(tx, NewEIP155Signer(big.NewInt(1)), common.HexToECDSA("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	// if hash := tx.Hash(); hash.Hex() != "0x7feaff7ef852f6b1dae12b847b087732e33f572f8ec84a3e17cc76aab5e377be" {
	// 	t.Errorf("hash mismatch: want: %s have: %s", "0x7feaff7ef852f6b1dae12b847b087732e33f572f8ec84a3e17cc76aab5e377be", hash.Hex())
	// }
}

func TestOtherValid(t *testing.T) {
	// chainId:  61717561,
	// from:     "0x29f3f6bd8d7d37400a138a9f9fab7f7c2ca847d0",
	// gas:      21000,
	// gasPrice: 1000000000,
	// nonce:    1,
	// to:       "0x29f3f6bd8d7d37400a138a9f9fab7f7c2ca847d0",
	// value:    "1000000000000000000"
	// {
	// 	raw: "0xf86f01843b9aca008252089429f3f6bd8d7d37400a138a9f9fab7f7c2ca847d0880de0b6b3a76400008084075b7896a0cfb22a5ed7e2a1ef0a8d768823c00d845f6bd4df06b56de4ac9c306b9861bf03a04b76a486e0934b07b21868b364038a884c174bbcbbbe181f726837637909ef88",
	// 	tx: {
	// 	  gas: "0x5208",
	// 	  gasPrice: "0x3b9aca00",
	// 	  hash: "0x1abd584dace47f1809c2d42f0c14f5cd94e56c6dfee7899f83a5763b5d10974e",
	// 	  input: "0x",
	// 	  nonce: "0x1",
	// 	  r: "0xcfb22a5ed7e2a1ef0a8d768823c00d845f6bd4df06b56de4ac9c306b9861bf03",
	// 	  s: "0x4b76a486e0934b07b21868b364038a884c174bbcbbbe181f726837637909ef88",
	// 	  to: "0x29f3f6bd8d7d37400a138a9f9fab7f7c2ca847d0",
	// 	  v: "0x75b7896",
	// 	  value: "0xde0b6b3a7640000"
	// 	}
	//   }

	raw := "f86f01843b9aca008252089429f3f6bd8d7d37400a138a9f9fab7f7c2ca847d0880de0b6b3a76400008084075b7896a0cfb22a5ed7e2a1ef0a8d768823c00d845f6bd4df06b56de4ac9c306b9861bf03a04b76a486e0934b07b21868b364038a884c174bbcbbbe181f726837637909ef88"
	tx := new(Transaction)
	if err := rlp.DecodeBytes(common.Hex2Bytes(raw), tx); err != nil {
		t.Fatalf("decoding tx: %v", err)
	}
	signer := NewEIP155Signer(big.NewInt(61717561))
	from, err := Sender(signer, tx)
	if err != nil {
		t.Fatalf("unexpected sender error: %v", err)
	}

	if from != *tx.To() {
		t.Errorf("expected sender %s got %s", tx.To().Hex(), from.Hex())
	}
}
