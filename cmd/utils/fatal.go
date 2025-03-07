package utils

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var start_time = time.Now()

// Fatalf formats a message to standard error and exits the program.
// The message is also printed to standard output if standard error
// is redirected to a different file.
func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		// The SameFile check below doesn't work on Windows.
		// stdout is unlikely to get redirected though, so just print there.
		w = os.Stdout
	} else {
		outf, _ := os.Stdout.Stat()
		errf, _ := os.Stderr.Stat()
		if outf != nil && errf != nil && os.SameFile(outf, errf) {
			w = os.Stderr
		}
	}

	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)

	// small traceback
	if debug := sense.Getenv("DEBUG"); debug != "" || time.Since(start_time) > 10*time.Second {
		pc := make([]uintptr, 8)
		n := runtime.Callers(1, pc)
		if n != 0 {
			pc = pc[:n]
			frames := runtime.CallersFrames(pc)
			for {
				frame, more := frames.Next()
				fmt.Fprintf(w, "\t >%s:%d %s\n", frame.File, frame.Line,
					frame.Function)
				if !more {
					break
				}
			}
		}

	}
	os.Exit(111)
}
