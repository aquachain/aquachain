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

import (
	"math/big"
	"reflect"
	"testing"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new *ChainConfig
		head        uint64
		wantErr     *ConfigCompatError
	}
	wanthf := AllAquahashProtocolChanges.HF
	tests := []test{
		{stored: AllAquahashProtocolChanges, new: AllAquahashProtocolChanges, head: 0, wantErr: nil},
		{stored: AllAquahashProtocolChanges, new: AllAquahashProtocolChanges, head: 100, wantErr: nil},
		{
			stored: &ChainConfig{HF: ForkMap{1: big.NewInt(10)}},
			new:    &ChainConfig{HF: ForkMap{1: big.NewInt(20)}},
			head:   11,
			wantErr: &ConfigCompatError{
				What:         "Aquachain HF1 block",
				StoredConfig: big.NewInt(10),
				NewConfig:    big.NewInt(20),
				RewindTo:     9,
			},
		},
		{
			stored:  &ChainConfig{EIP150Block: big.NewInt(10), HF: wanthf},
			new:     &ChainConfig{EIP150Block: big.NewInt(20), HF: wanthf},
			head:    9,
			wantErr: nil,
		},
		{
			stored: AllAquahashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: nil, HF: wanthf},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    nil,
				RewindTo:     0,
			},
		},
		{
			stored: AllAquahashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: big.NewInt(1), HF: wanthf},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    big.NewInt(1),
				RewindTo:     0,
			},
		},
		{
			stored: &ChainConfig{HomesteadBlock: big.NewInt(30), EIP150Block: big.NewInt(10)},
			new:    &ChainConfig{HomesteadBlock: big.NewInt(25), EIP150Block: big.NewInt(20)},
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP150 fork block",
				StoredConfig: big.NewInt(10),
				NewConfig:    big.NewInt(20),
				RewindTo:     9,
			},
		},
	}

	for i, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.head)
		if !reflect.DeepEqual(err, test.wantErr) {
			name := "???"
			if test.wantErr != nil {
				name = test.wantErr.What
			}
			t.Errorf("test #%v (%s) error mismatch:\nstored: %v\nnew: %v\nhead: %v\nerr: %v\nwant: %v", i, name, test.stored, test.new, test.head, err, test.wantErr)
		}
	}
}

func TestChainConfigs(t *testing.T) {
	all := allChainConfigs
	for _, v := range all {
		if v == nil {
			t.Error("nil config")
			continue
		}
		if !IsValid(v) {
			t.Error("invalid config")
		}
		if v.ChainId == nil {
			t.Error("nil chain id")
		}
		if GetChainConfig(v.Name()) != v {
			t.Error("GetChainConfig mismatch")
		}
		if GetChainConfigByChainId(v.ChainId) != v {
			t.Error("GetChainConfigByChainId mismatch")
		}
		name := v.Name()
		chainid := v.ChainId.Uint64()
		shouldBeZero := name == "dev" || name == "test"
		if name == "" {
			t.Error("missing name")
		}
		if (v.DefaultBootstrapPort == 0) && !shouldBeZero {
			t.Errorf("missing default bootstrap port for % 10v %d port=%d", name, chainid, v.DefaultBootstrapPort)
		}
		if (v.DefaultPortNumber == 0) && !shouldBeZero {
			t.Errorf("missing default discovery port for % 10v %d port=%d", name, chainid, v.DefaultPortNumber)
		}
	}
}
