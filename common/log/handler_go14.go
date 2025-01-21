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

//go:build go1.4
// +build go1.4

package log

import "sync/atomic"

// swapHandler wraps another handler that may be swapped out
// dynamically at runtime in a thread-safe fashion.
type swapHandler struct {
	handler atomic.Value
}

func (h *swapHandler) Log(r *Record) error {
	return (*h.handler.Load().(*Handler)).Log(r)
}

func (h *swapHandler) Swap(newHandler Handler) {
	h.handler.Store(&newHandler)
}

func (h *swapHandler) Get() Handler {
	return *h.handler.Load().(*Handler)
}
