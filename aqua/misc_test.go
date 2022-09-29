// Copyright 2018,2022 The aquachain Authors
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

package aqua

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeExtra(t *testing.T) {
	b := makeExtraData(nil)
	t.Logf("extra: %s\n", gohex(b)) // print if -test.v flag
	version, o, extra, err := DecodeExtraData(b)
	if err != nil {
		t.Log("got err:", err)
		t.Fail()
	}
	t.Log("version:", version)
	t.Log("OS:", o)
	t.Log("raw extra string:", string(extra[6:]))
	t.Logf("raw first6: %#02x", extra[:6])
}

func TestDecodeExtra2(t *testing.T) {
	var (
		wants = [][3]uint8{
			{1, 7, 12},
			{1, 7, 7},
			{1, 7, 12},
		}

		bufs = [][]byte{
			{0xd9, 0x83, 0x01, 0x07, 0x0c, 0x84, 0x61, 0x71,
				0x75, 0x61, 0x85, 0x6c, 0x69, 0x6e, 0x75, 0x78,
				0x89, 0x67, 0x6f, 0x64, 0x65, 0x76, 0x31, 0x2e, 0x32, 0x30},
			{0xd4, 0x83, 0x1, 0x7, 0x7, 0x89, 0x61, 0x71, 0x75,
				0x61, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x85, 0x6c, 0x69,
				0x6e, 0x75, 0x7},
			{0xd6, 0x83, 0x01, 0x07, 0x0c, 0x84, 0x61, 0x71,
				0x75, 0x61, 0x85, 0x6c, 0x69, 0x6e, 0x75, 0x78,
				0x86, 0x67, 0x6f, 0x31, 0x2e, 0x32, 0x30}}
	)
	for i := 0; i < len(bufs); i++ {
		var b []byte = bufs[i]
		var wantVersion [3]uint8 = wants[i]
		version, osname, extra, err := DecodeExtraData(b)
		if err != nil {
			t.Log("err non-nil", err)
			t.FailNow()
		}
		t.Log("Detected OS:", osname)
		for i := 0; i < 3; i++ {

			if version[i] != wantVersion[i] {
				t.Log("version mismatch digit", i, ":", version, "wanted:", wantVersion)
				t.Fail()
			}
		}
		t.Log("Detected Version:", version[0], version[1], version[2])
		if 0 != bytes.Compare(extra, b) {
			t.Log("extra mismatch:", gohex(extra), "wanted:", gohex(b))
			t.Fail()
		}
		t.Log("extra:", string(b[6:]))
	}

}

func gohex(b []byte) string {
	return strings.Replace(fmt.Sprintf("%# 02x", b), " ", ", ", -1)
}
