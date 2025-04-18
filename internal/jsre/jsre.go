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

// Package jsre provides execution environment for JavaScript.
package jsre

import (
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	"math/rand"
	"time"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/internal/jsre/deps"
	"gitlab.com/aquachain/aquachain/opt/console/jsruntime"
)

var (
	BigNumber_JS = deps.MustAsset("bignumber.js")
	Web3_JS      = deps.MustAsset("web3.js")
)

/*
JSRE is a generic JS runtime environment embedding the jsruntime JS interpreter.
It provides some helper functions to
- load code from files
- run code snippets
- require libraries
- bind native go objects
*/
type JSRE struct {
	assetPath     string
	output        io.Writer
	evalQueue     chan *evalReq
	stopEventLoop chan bool
	closed        chan struct{}
}

// jsTimer is a single timer instance with a callback function
type jsTimer struct {
	timer    *time.Timer
	duration time.Duration
	interval bool
	call     jsruntime.FunctionCall
}

// evalReq is a serialized vm execution request processed by runEventLoop.
type evalReq struct {
	fn   func(vm *jsruntime.Otto)
	done chan bool
}

// runtime must be stopped with Stop() after use and cannot be used after stopping
func New(assetPath string, output io.Writer) *JSRE {
	re := &JSRE{
		assetPath:     assetPath,
		output:        output,
		closed:        make(chan struct{}),
		evalQueue:     make(chan *evalReq),
		stopEventLoop: make(chan bool),
	}
	go re.runEventLoop()
	re.Set("loadScript", re.loadScript)
	re.Set("inspect", re.prettyPrintJS)
	return re
}

// randomSource returns a pseudo random value generator.
func randomSource() *rand.Rand {
	bytes := make([]byte, 8)
	seed := time.Now().UnixNano()
	if _, err := crand.Read(bytes); err == nil {
		seed = int64(binary.LittleEndian.Uint64(bytes))
	}

	src := rand.NewSource(seed)
	return rand.New(src)
}

// This function runs the main event loop from a goroutine that is started
// when JSRE is created. Use Stop() before exiting to properly stop it.
// The event loop processes vm access requests from the evalQueue in a
// serialized way and calls timer callback functions at the appropriate time.

// Exported functions always access the vm through the event queue. You can
// call the functions of the jsruntime vm directly to circumvent the queue. These
// functions should be used if and only if running a routine that was already
// called from JS through an RPC call.
func (self *JSRE) runEventLoop() {
	defer close(self.closed)

	vm := jsruntime.New()
	r := randomSource()
	vm.SetRandomSource(r.Float64)

	registry := map[*jsTimer]*jsTimer{}
	ready := make(chan *jsTimer)

	newTimer := func(call jsruntime.FunctionCall, interval bool) (*jsTimer, jsruntime.Value) {
		delay, _ := call.Argument(1).ToInteger()
		if 0 >= delay {
			delay = 1
		}
		timer := &jsTimer{
			duration: time.Duration(delay) * time.Millisecond,
			call:     call,
			interval: interval,
		}
		registry[timer] = timer

		timer.timer = time.AfterFunc(timer.duration, func() {
			ready <- timer
		})

		value, err := call.Otto.ToValue(timer)
		if err != nil {
			panic(err)
		}
		return timer, value
	}

	setTimeout := func(call jsruntime.FunctionCall) jsruntime.Value {
		_, value := newTimer(call, false)
		return value
	}

	setInterval := func(call jsruntime.FunctionCall) jsruntime.Value {
		_, value := newTimer(call, true)
		return value
	}

	clearTimeout := func(call jsruntime.FunctionCall) jsruntime.Value {
		timer, _ := call.Argument(0).Export()
		if timer, ok := timer.(*jsTimer); ok {
			timer.timer.Stop()
			delete(registry, timer)
		}
		return jsruntime.UndefinedValue()
	}

	vm.Set("_setTimeout", setTimeout)
	vm.Set("_setInterval", setInterval)
	vm.Run(`var setTimeout = function(args) {
		if (arguments.length < 1) {
			throw TypeError("Failed to execute 'setTimeout': 1 argument required, but only 0 present.");
		}
		return _setTimeout.apply(this, arguments);
	}`)
	vm.Run(`var setInterval = function(args) {
		if (arguments.length < 1) {
			throw TypeError("Failed to execute 'setInterval': 1 argument required, but only 0 present.");
		}
		return _setInterval.apply(this, arguments);
	}`)
	vm.Set("clearTimeout", clearTimeout)
	vm.Set("clearInterval", clearTimeout)

	var waitForCallbacks bool

loop:
	for {
		select {
		case timer := <-ready:
			// execute callback, remove/reschedule the timer
			var arguments []interface{}
			if len(timer.call.ArgumentList) > 2 {
				tmp := timer.call.ArgumentList[2:]
				arguments = make([]interface{}, 2+len(tmp))
				for i, value := range tmp {
					arguments[i+2] = value
				}
			} else {
				arguments = make([]interface{}, 1)
			}
			arguments[0] = timer.call.ArgumentList[0]
			_, err := vm.Call(`Function.call.call`, nil, arguments...)
			if err != nil {
				fmt.Println("js error:", err, arguments)
			}

			_, inreg := registry[timer] // when clearInterval is called from within the callback don't reset it
			if timer.interval && inreg {
				timer.timer.Reset(timer.duration)
			} else {
				delete(registry, timer)
				if waitForCallbacks && (len(registry) == 0) {
					break loop
				}
			}
		case req := <-self.evalQueue:
			// run the code, send the result back
			req.fn(vm)
			close(req.done)
			if waitForCallbacks && (len(registry) == 0) {
				break loop
			}
		case waitForCallbacks = <-self.stopEventLoop:
			if !waitForCallbacks || (len(registry) == 0) {
				break loop
			}
		}
	}

	for _, timer := range registry {
		timer.timer.Stop()
		delete(registry, timer)
	}
}

// Do executes the given function on the JS event loop.
func (self *JSRE) Do(fn func(*jsruntime.Otto)) {
	done := make(chan bool)
	req := &evalReq{fn, done}
	self.evalQueue <- req
	<-done
}

// stops the event loop before exit, optionally waits for all timers to expire
func (self *JSRE) Stop(waitForCallbacks bool) {
	select {
	case <-self.closed:
	case self.stopEventLoop <- waitForCallbacks:
		<-self.closed
	}
}

// Exec(file) loads and runs the contents of a file
// if a relative path is given, the jsre's assetPath is used
func (self *JSRE) ExecFile(file string) error {
	code, err := ioutil.ReadFile(common.AbsolutePath(self.assetPath, file))
	if err != nil {
		return err
	}
	var script *jsruntime.Script
	self.Do(func(vm *jsruntime.Otto) {
		script, err = vm.Compile(file, code)
		if err != nil {
			log.Error("failed to compile js", err, fmt.Sprintf("%+v", err))
			return
		}
		_, err = vm.Run(script)
	})
	return err
}

// Bind assigns value v to a variable in the JS environment
// This method is deprecated, use Set.
func (self *JSRE) Bind(name string, v interface{}) error {
	return self.Set(name, v)
}

// Run runs a piece of JS code.
func (self *JSRE) Run(code string) (v jsruntime.Value, err error) {
	if debugJsre {
		log.Info("JSRE.Run", "code", code)
	}
	self.Do(func(vm *jsruntime.Otto) { v, err = vm.Run(code) })
	return v, err
}

// Get returns the value of a variable in the JS environment.
func (self *JSRE) Get(ns string) (v jsruntime.Value, err error) {
	self.Do(func(vm *jsruntime.Otto) { v, err = vm.Get(ns) })
	return v, err
}

// Set assigns value v to a variable in the JS environment.
func (self *JSRE) Set(ns string, v interface{}) (err error) {
	self.Do(func(vm *jsruntime.Otto) { err = vm.Set(ns, v) })
	return err
}

// loadScript executes a JS script from inside the currently executing JS code.
func (self *JSRE) loadScript(call jsruntime.FunctionCall) jsruntime.Value {
	file, err := call.Argument(0).ToString()
	if err != nil {
		// TODO: throw exception
		return jsruntime.FalseValue()
	}
	file = common.AbsolutePath(self.assetPath, file)
	source, err := ioutil.ReadFile(file)
	if err != nil {
		// TODO: throw exception
		return jsruntime.FalseValue()
	}
	if _, err := compileAndRun(call.Otto, file, source); err != nil {
		// TODO: throw exception
		fmt.Println("err:", err)
		return jsruntime.FalseValue()
	}
	// TODO: return evaluation result
	return jsruntime.TrueValue()
}

// Evaluate executes code and pretty prints the result to the specified output
// stream.
func (self *JSRE) Evaluate(code string, w io.Writer) error {
	if code == "" {
		return fmt.Errorf("empty js input")
	}
	var fail error

	self.Do(func(vm *jsruntime.Otto) {
		defer func() {
			if err := recover(); err != nil {
				fmt.Printf("recovered JS panic: %+v", err)
			}
		}()

		val, err := vm.Run(code)

		if err != nil {
			prettyError(vm, err, w)
			fmt.Fprintln(w)
		} else if val.IsDefined() {
			prettyPrint(vm, val, w)
			fmt.Fprintln(w)
		}
	})
	return fail
}

// Compile compiles and then runs a piece of JS code.
func (self *JSRE) Compile(filename string, src interface{}) (err error) {
	self.Do(func(vm *jsruntime.Otto) { _, err = compileAndRun(vm, filename, src) })
	return err
}

func compileAndRun(vm *jsruntime.Otto, filename string, src interface{}) (jsruntime.Value, error) {
	script, err := vm.Compile(filename, src)
	if err != nil {
		return jsruntime.Value{}, err
	}
	return vm.Run(script)
}
