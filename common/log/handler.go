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

package log

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/go-stack/stack"
	"gitlab.com/aquachain/aquachain/common/sense"
)

var debuglog = sense.Getenv("DEBUG_LOG") == "1"

func println(a ...any) {
	if debuglog || sense.Getenv("DEBUG_LOG") == "1" {
		caller := Caller(2)
		json.NewEncoder(os.Stderr).Encode(map[string]any{
			"time":    time.Now().Unix(),
			"level":   "debug",
			"caller":  caller,
			"message": fmt.Sprint(a...),
		})
		// fmt.Fprintln(os.Stderr, append([]any{caller}, a...)...)
	}
}

// A Logger prints its log records by writing to a Handler.
// The Handler interface defines where and how log records are written.
// Handlers are composable, providing you great flexibility in combining
// them to achieve the logging structure that suits your applications.
type Handler interface {
	Log(r *Record) error
}

// FuncHandler returns a Handler that logs records with the given
// function.
func FuncHandler(fn func(r *Record) error) Handler {
	return funcHandler(fn)
}

type funcHandler func(r *Record) error

func (h funcHandler) Log(r *Record) error {
	return h(r)
}

// StreamHandler writes log records to an io.Writer
// with the given format. StreamHandler can be used
// to easily begin writing log records to other
// outputs.
//
// StreamHandler wraps itself with LazyHandler and SyncHandler
// to evaluate Lazy objects and perform safe concurrent writes.
func StreamHandler(wr io.Writer, fmtr Format) Handler {
	println("new logging stream-handler")
	h := FuncHandler(func(r *Record) error {
		_, err := wr.Write(fmtr.Format(r))
		return err
	})
	return LazyHandler(SyncHandler(h))
}

// SyncHandler can be wrapped around a handler to guarantee that
// only a single Log operation can proceed at a time. It's necessary
// for thread-safe concurrent writes.
func SyncHandler(h Handler) Handler {
	if NoSync {
		println("skip logging sync-handler (no sync)")
		return h
	}
	println("new logging sync-handler")
	var mu sync.Mutex
	return FuncHandler(func(r *Record) error {
		defer mu.Unlock()
		mu.Lock()
		return h.Log(r)
	})
}

// FileHandler returns a handler which writes log records to the give file
// using the given format. If the path
// already exists, FileHandler will append to the given file. If it does not,
// FileHandler will create the file with mode 0644.
func FileHandler(path string, fmtr Format) (Handler, error) {
	println("new logging file-handler")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return closingHandler{f, StreamHandler(f, fmtr)}, nil
}

// NetHandler opens a socket to the given address and writes records
// over the connection.
func NetHandler(network, addr string, fmtr Format) (Handler, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	return closingHandler{conn, StreamHandler(conn, fmtr)}, nil
}

// XXX: closingHandler is essentially unused at the moment
// it's meant for a future time when the Handler interface supports
// a possible Close() operation
type closingHandler struct {
	io.WriteCloser
	Handler
}

func (h *closingHandler) Close() error {
	return h.WriteCloser.Close()
}

// CallerFileHandler returns a Handler that adds the line number and file of
// the calling function to the context with key "caller".
func CallerFileHandler(h Handler) Handler {
	verb := "%+v" // relative path
	if sense.Getenv("DEBUG_CALLER") == "1" {
		verb = "%#v" // full path, TODO: do once
	}
	return FuncHandler(func(r *Record) error {
		callfmt := formatCall(verb, r.Call)
		for _, prefix := range locationTrims {
			if len(callfmt) > len(prefix) && callfmt[:len(prefix)] == prefix {
				callfmt = callfmt[len(prefix):]
				break
			}
		}
		r.Ctx = append([]any{"caller", callfmt}, r.Ctx...)
		return h.Log(r)
	})
}

// // CallerFuncHandler returns a Handler that adds the calling function name to
// // the context with key "fn".
// func CallerFuncHandler(h Handler) Handler {
// 	return FuncHandler(func(r *Record) error {
// 		r.Ctx = append(r.Ctx, "fn", formatCall("%+n", r.Call))
// 		return h.Log(r)
// 	})
// }

// This function is here to please go vet on Go < 1.8.
func formatCall(format string, c stack.Call) string {
	return fmt.Sprintf(format, c)
}

// // CallerStackHandler returns a Handler that adds a stack trace to the context
// // with key "stack". The stack trace is formated as a space separated list of
// // call sites inside matching []'s. The most recent call site is listed first.
// // Each call site is formatted according to format. See the documentation of
// // package github.com/go-stack/stack for the list of supported formats.
// func CallerStackHandler(format string, h Handler) Handler {
// 	return FuncHandler(func(r *Record) error {
// 		s := stack.Trace().TrimBelow(r.Call).TrimRuntime()
// 		if len(s) > 0 {
// 			r.Ctx = append(r.Ctx, "stack", fmt.Sprintf(format, s))
// 		}
// 		return h.Log(r)
// 	})
// }

// FilterHandler returns a Handler that only writes records to the
// wrapped Handler if the given function evaluates true. For example,
// to only log records where the 'err' key is not nil:
//
//	logger.SetHandler(FilterHandler(func(r *Record) bool {
//	    for i := 0; i < len(r.Ctx); i += 2 {
//	        if r.Ctx[i] == "err" {
//	            return r.Ctx[i+1] != nil
//	        }
//	    }
//	    return false
//	}, h))
func FilterHandler(fn func(r *Record) bool, h Handler) Handler {
	return FuncHandler(func(r *Record) error {
		if fn(r) {
			return h.Log(r)
		}
		return nil
	})
}

// // MatchFilterHandler returns a Handler that only writes records
// // to the wrapped Handler if the given key in the logged
// // context matches the value. For example, to only log records
// // from your ui package:
// //
// //	log.MatchFilterHandler("pkg", "app/ui", log.StdoutHandler)
// func MatchFilterHandler(key string, value interface{}, h Handler) Handler {
// 	return FilterHandler(func(r *Record) (pass bool) {
// 		switch key {
// 		case r.KeyNames.Lvl:
// 			return r.Lvl == value
// 		case r.KeyNames.Time:
// 			return r.Time == value
// 		case r.KeyNames.Msg:
// 			return r.Msg == value
// 		}

// 		for i := 0; i < len(r.Ctx); i += 2 {
// 			if r.Ctx[i] == key {
// 				return r.Ctx[i+1] == value
// 			}
// 		}
// 		return false
// 	}, h)
// }

// LvlFilterHandler returns a Handler that only writes
// records which are less than the given verbosity
// level to the wrapped Handler. For example, to only
// log Error/Crit records:
//
//	log.LvlFilterHandler(log.LvlError, log.StdoutHandler)
func LvlFilterHandler(maxLvl Lvl, h Handler) Handler {
	return FilterHandler(func(r *Record) (pass bool) {
		return r.Lvl <= maxLvl
	}, h)
}

// A MultiHandler dispatches any write to each of its handlers.
// This is useful for writing different types of log information
// to different locations. For example, to log to a file and
// standard error:
//
//	log.MultiHandler(
//	    log.Must.FileHandler("/var/log/app.log", log.LogfmtFormat()),
//	    log.StderrHandler)
func MultiHandler(hs ...Handler) Handler {
	return FuncHandler(func(r *Record) error {
		for _, h := range hs {
			// what to do about failures?
			h.Log(r)
		}
		return nil
	})
}

// A FailoverHandler writes all log records to the first handler
// specified, but will failover and write to the second handler if
// the first handler has failed, and so on for all handlers specified.
// For example you might want to log to a network socket, but failover
// to writing to a file if the network fails, and then to
// standard out if the file write fails:
//
//	log.FailoverHandler(
//	    log.Must.NetHandler("tcp", ":9090", log.JsonFormat()),
//	    log.Must.FileHandler("/var/log/app.log", log.LogfmtFormat()),
//	    log.StdoutHandler)
//
// All writes that do not go to the first handler will add context with keys of
// the form "failover_err_{idx}" which explain the error encountered while
// trying to write to the handlers before them in the list.
func FailoverHandler(hs ...Handler) Handler {
	return FuncHandler(func(r *Record) error {
		var err error
		for i, h := range hs {
			err = h.Log(r)
			if err == nil {
				return nil
			} else {
				r.Ctx = append(r.Ctx, fmt.Sprintf("failover_err_%d", i), err)
			}
		}

		return err
	})
}

// ChannelHandler writes all records to the given channel.
// It blocks if the channel is full. Useful for async processing
// of log messages, it's used by BufferedHandler.
func ChannelHandler(recs chan<- *Record) Handler {
	return FuncHandler(func(r *Record) error {
		recs <- r
		return nil
	})
}

// BufferedHandler writes all records to a buffered
// channel of the given size which flushes into the wrapped
// handler whenever it is available for writing. Since these
// writes happen asynchronously, all writes to a BufferedHandler
// never return an error and any errors from the wrapped handler are ignored.
func BufferedHandler(bufSize int, h Handler) Handler {
	recs := make(chan *Record, bufSize)
	go func() {
		for m := range recs {
			_ = h.Log(m)
		}
	}()
	return ChannelHandler(recs)
}

// LazyHandler writes all values to the wrapped handler after evaluating
// any lazy functions in the record's context. It is already wrapped
// around StreamHandler and SyslogHandler in this library, you'll only need
// it if you write your own Handler.
func LazyHandler(h Handler) Handler {
	return FuncHandler(func(r *Record) error {
		// go through the values (odd indices) and reassign
		// the values of any lazy fn to the result of its execution
		hadErr := false
		for i := 1; i < len(r.Ctx); i += 2 {
			lz, ok := r.Ctx[i].(Lazy)
			if ok {
				v, err := evaluateLazy(lz)
				if err != nil {
					hadErr = true
					r.Ctx[i] = err
				} else {
					if cs, ok := v.(stack.CallStack); ok {
						v = cs.TrimBelow(r.Call).TrimRuntime()
					}
					r.Ctx[i] = v
				}
			}
		}

		if hadErr {
			r.Ctx = append(r.Ctx, errorKey, "bad lazy")
		}

		return h.Log(r)
	})
}

func evaluateLazy(lz Lazy) (interface{}, error) {
	t := reflect.TypeOf(lz.Fn)

	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("INVALID_LAZY, not func: %+v", lz.Fn)
	}

	if t.NumIn() > 0 {
		return nil, fmt.Errorf("INVALID_LAZY, func takes args: %+v", lz.Fn)
	}

	if t.NumOut() == 0 {
		return nil, fmt.Errorf("INVALID_LAZY, no func return val: %+v", lz.Fn)
	}

	value := reflect.ValueOf(lz.Fn)
	results := value.Call([]reflect.Value{})
	if len(results) == 1 {
		return results[0].Interface(), nil
	} else {
		values := make([]interface{}, len(results))
		for i, v := range results {
			values[i] = v.Interface()
		}
		return values, nil
	}
}

// DiscardHandler reports success for all writes but does nothing.
// It is useful for dynamically disabling logging at runtime via
// a Logger's SetHandler method.
func DiscardHandler() Handler {
	return FuncHandler(func(r *Record) error {
		return nil
	})
}

// The Must object provides the following Handler creation functions
// which instead of returning an error parameter only return a Handler
// and panic on failure: FileHandler, NetHandler, SyslogHandler, SyslogNetHandler
var Must muster

func must(h Handler, err error) Handler {
	if err != nil {
		panic(err)
	}
	return h
}

type muster struct{}

func (m muster) FileHandler(path string, fmtr Format) Handler {
	return must(FileHandler(path, fmtr))
}

func (m muster) NetHandler(network, addr string, fmtr Format) Handler {
	return must(NetHandler(network, addr, fmtr))
}
