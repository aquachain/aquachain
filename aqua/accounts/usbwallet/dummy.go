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

// +build !usb

package usbwallet

import (
	"errors"

	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/event"
)

// Hub is a accounts.Backend that can find and handle generic USB hardware wallets.
type Hub struct{}

// NewLedgerHub creates a new hardware wallet manager for Ledger devices.
func NewLedgerHub() (*Hub, error) {
	return nil, accounts.ErrNotSupported
}

// ErrTrezorPINNeeded is returned if opening the trezor requires a PIN code. In
// this case, the calling application should display a pinpad and send back the
// encoded passphrase.
var ErrTrezorPINNeeded = errors.New("trezor: pin needed")

// NewTrezorHub creates a new hardware wallet manager for Trezor devices.
func NewTrezorHub() (*Hub, error) {
	return nil, accounts.ErrNotSupported
}

// Wallets implements accounts.Backend, returning all the currently tracked USB
// devices that appear to be hardware wallets.
func (hub *Hub) Wallets() []accounts.Wallet {
	return nil
}

// Subscribe implements accounts.Backend, creating an async subscription to
// receive notifications on the addition or removal of USB wallets.
func (hub *Hub) Subscribe(sink chan<- accounts.WalletEvent) event.Subscription {
	return nil
}
