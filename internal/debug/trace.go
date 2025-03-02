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

//go:build go1.5
// +build go1.5

package debug

import (
	"errors"
	"os"
	"runtime/trace"
	"sync"
	"time"

	"github.com/go-stack/stack"
	"gitlab.com/aquachain/aquachain/common/log"
)

// StartGoTrace turns on tracing, writing to the given file.
func (h *HandlerT) StartGoTrace(file string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.traceW != nil {
		return errors.New("trace already in progress")
	}
	f, err := os.Create(expandHome(file))
	if err != nil {
		return err
	}
	if err := trace.Start(f); err != nil {
		f.Close()
		return err
	}
	h.traceW = f
	h.traceFile = file
	log.Info("Go tracing started", "dump", h.traceFile)
	return nil
}

// StopTrace stops an ongoing trace.
func (h *HandlerT) StopGoTrace() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	trace.Stop()
	if h.traceW == nil {
		return errors.New("trace not in progress")
	}
	log.Info("Done writing Go trace", "dump", h.traceFile)
	h.traceW.Close()
	h.traceW = nil
	h.traceFile = ""
	return nil
}

var loops = []loopinfo{}

var loopwg sync.WaitGroup

type loopinfo struct {
	caller stack.Call // TODO: remove stack package
}

// AddLoop starts tracking a loop and must be closed
func AddLoop() func() {
	callerinfo := log.Caller(2)
	loops = append(loops, loopinfo{caller: callerinfo})
	loopwg.Add(1)
	return loopwg.Done
}

// Loops returns a list of all loops for logging purposes
func Loops() []string {
	var out []string
	for _, l := range loops {
		out = append(out, l.caller.String())
	}
	return out
}

// WaitLoops waits for all loops to finish, should be called only once
func WaitLoops(d time.Duration) error {
	ch := make(chan struct{})
	go func() {
		loopwg.Wait()
		close(ch)
	}()
	select {
	case <-time.After(d):
		return errors.New("timeout waiting for loops")
	case <-ch:
		return nil
	}
}
func init() {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			log.Info("Loops", "loops", Loops())
		}
	}()
}
