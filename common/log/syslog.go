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

// +build !windows,!plan9,!nacl

package log

import (
	"log/syslog"
	"strings"
)

// SyslogHandler opens a connection to the system syslog daemon by calling
// syslog.New and writes all records to it.
func SyslogHandler(priority syslog.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := syslog.New(priority, tag)
	return sharedSyslog(fmtr, wr, err)
}

// SyslogNetHandler opens a connection to a log daemon over the network and writes
// all log records to it.
func SyslogNetHandler(net, addr string, priority syslog.Priority, tag string, fmtr Format) (Handler, error) {
	wr, err := syslog.Dial(net, addr, priority, tag)
	return sharedSyslog(fmtr, wr, err)
}

func sharedSyslog(fmtr Format, sysWr *syslog.Writer, err error) (Handler, error) {
	if err != nil {
		return nil, err
	}
	h := FuncHandler(func(r *Record) error {
		var syslogFn = sysWr.Info
		switch r.Lvl {
		case LvlCrit:
			syslogFn = sysWr.Crit
		case LvlError:
			syslogFn = sysWr.Err
		case LvlWarn:
			syslogFn = sysWr.Warning
		case LvlInfo:
			syslogFn = sysWr.Info
		case LvlDebug:
			syslogFn = sysWr.Debug
		case LvlTrace:
			syslogFn = func(m string) error { return nil } // There's no syslog level for trace
		}

		s := strings.TrimSpace(string(fmtr.Format(r)))
		return syslogFn(s)
	})
	return LazyHandler(&closingHandler{sysWr, h}), nil
}

func (m muster) SyslogHandler(priority syslog.Priority, tag string, fmtr Format) Handler {
	return must(SyslogHandler(priority, tag, fmtr))
}

func (m muster) SyslogNetHandler(net, addr string, priority syslog.Priority, tag string, fmtr Format) Handler {
	return must(SyslogNetHandler(net, addr, priority, tag, fmtr))
}
