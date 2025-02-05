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

package debug

import (
	"context"
	"fmt"
	"io"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"

	colorable "github.com/mattn/go-colorable"
	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/log/term"
	"gitlab.com/aquachain/aquachain/common/metrics"
	"gitlab.com/aquachain/aquachain/common/metrics/exp"
)

var (
	verbosityFlag = &cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	vmoduleFlag = &cli.StringFlag{
		Name:  "vmodule",
		Usage: "Per-module verbosity: comma-separated list of <pattern>=<level> (e.g. aqua/*=5,p2p=4), try \"good\" or \"great\" for predefined verbose logging",
		Value: "",
	}
	backtraceAtFlag = &cli.StringFlag{
		Name:  "backtrace",
		Usage: "Request a stack trace at a specific logging statement (e.g. \"block.go:271\")",
		Value: "",
	}
	debugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	pprofFlag = &cli.BoolFlag{
		Name:  "pprof",
		Usage: "Enable the pprof HTTP server",
	}
	pprofPortFlag = &cli.IntFlag{
		Name:  "pprofport",
		Usage: "pprof HTTP server listening port",
		Value: 6060,
	}
	pprofAddrFlag = &cli.StringFlag{
		Name:  "pprofaddr",
		Usage: "pprof HTTP server listening interface",
		Value: "127.0.0.1",
	}
	memprofilerateFlag = &cli.IntFlag{
		Name:  "memprofilerate",
		Usage: "Turn on memory profiling with the given rate",
		Value: int64(runtime.MemProfileRate),
	}
	blockprofilerateFlag = &cli.IntFlag{
		Name:  "blockprofilerate",
		Usage: "Turn on block profiling with the given rate",
	}
	cpuprofileFlag = &cli.StringFlag{
		Name:  "cpuprofile",
		Usage: "Write CPU profile to the given file",
	}
	traceFlag = &cli.StringFlag{
		Name:  "trace",
		Usage: "Write execution trace to the given file",
	}
)

// Flags holds all command-line flags required for debugging.
var Flags = []cli.Flag{
	verbosityFlag, vmoduleFlag, backtraceAtFlag, debugFlag,
	pprofFlag, pprofAddrFlag, pprofPortFlag,
	memprofilerateFlag, blockprofilerateFlag, cpuprofileFlag, traceFlag,
}

var glogger *log.GlogHandler = initglogger()

func initglogger() *log.GlogHandler {
	usecolor := os.Getenv("COLOR") == "1" || (term.IsTty(os.Stderr.Fd()) && os.Getenv("TERM") != "dumb")
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	x := log.NewGlogHandler(log.StreamHandler(output, log.TerminalFormat(usecolor)))
	x.Verbosity(log.LvlInfo)
	return x
}

// Setup initializes profiling and logging based on the CLI flags.
// It should be called as early as possible in the program.
func Setup(ctx_ context.Context, cmd *cli.Command) error {
	// logging
	log.PrintOrigins(cmd.Bool(debugFlag.Name))
	glogger.Verbosity(log.Lvl(cmd.Int(verbosityFlag.Name)))
	glogger.Vmodule(wrapVmodule(cmd.String(vmoduleFlag.Name)))
	glogger.BacktraceAt(cmd.String(backtraceAtFlag.Name))
	log.Root().SetHandler(glogger)

	// profiling, tracing
	runtime.MemProfileRate = int(cmd.Int(memprofilerateFlag.Name))
	Handler.SetBlockProfileRate(int(cmd.Int(blockprofilerateFlag.Name)))
	if traceFile := cmd.String(traceFlag.Name); traceFile != "" {
		if err := Handler.StartGoTrace(traceFile); err != nil {
			return err
		}
	}
	if cpuFile := cmd.String(cpuprofileFlag.Name); cpuFile != "" {
		if err := Handler.StartCPUProfile(cpuFile); err != nil {
			return err
		}
	}

	// pprof server
	if cmd.Bool(pprofFlag.Name) {
		// Hook go-metrics into expvar on any /debug/metrics request, load all vars
		// from the registry into expvar, and execute regular expvar handler.
		exp.Exp(metrics.DefaultRegistry)

		runtime.SetMutexProfileFraction(10)
		address := fmt.Sprintf("%s:%d", cmd.String(pprofAddrFlag.Name), cmd.Int(pprofPortFlag.Name))
		go func() {
			log.Info("Starting pprof server", "addr", fmt.Sprintf("http://%s/debug/pprof", address))
			if err := http.ListenAndServe(address, nil); err != nil {
				log.Error("Failure in running pprof server", "err", err)
			}
		}()
	}
	return nil
}

// Exit stops all running profiles, flushing their output to the
// respective file.
func Exit() {
	if Handler == nil {
		return
	}
	Handler.StopCPUProfile()
	Handler.StopGoTrace()
}
