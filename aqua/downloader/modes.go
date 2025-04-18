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

package downloader

import "fmt"

// SyncMode represents the synchronisation mode of the downloader.
type SyncMode int

const (
	FullSync    SyncMode = iota // Synchronise the entire blockchain history from full blocks
	FastSync                    // Quickly download the headers, full sync only at the chain head
	OfflineSync                 // no p2p
)

func (mode SyncMode) IsValid() bool {
	return mode >= FullSync && mode <= OfflineSync
}

// String implements the stringer interface.
func (mode SyncMode) String() string {
	switch mode {
	case FullSync:
		return "full"
	case FastSync:
		return "fast"
	case OfflineSync:
		return "offline"
	default:
		return "unknown"
	}
}

func (mode SyncMode) MarshalText() ([]byte, error) {
	switch mode {
	case FullSync:
		return []byte("full"), nil
	case FastSync:
		return []byte("fast"), nil
	case OfflineSync:
		return []byte("offline"), nil
	default:
		return nil, fmt.Errorf("unknown sync mode %d", mode)
	}
}

func (mode *SyncMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "full":
		*mode = FullSync
	case "fast":
		*mode = FastSync
	case "none", "offline":
		*mode = OfflineSync
	default:
		return fmt.Errorf(`unknown sync mode %q, want "full", "fast"`, text)
	}
	return nil
}
