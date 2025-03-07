package mainctxs

import (
	"context"
	"fmt"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/common/sense"
)

func Main() context.Context {
	return mainctx
}
func MainCancelCause() context.CancelCauseFunc {
	return maincancel
}

var mainctx, maincancelreal = mkmainctx()

func parseTypicalDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	// if all digits
	if strings.Trim(s, "0123456789") == "" {
		s += "s"
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Error("failed to parse duration", "duration", s, "error", err)
	}
	return d // 0 if error

}

func mkmainctx() (context.Context, context.CancelCauseFunc) {
	c := context.Background()
	tm := parseTypicalDuration(sense.Getenv("SCHEDULE_TIMEOUT"))
	var maybenoop, stopSignals context.CancelFunc
	var cancelCause context.CancelCauseFunc

	// first, timeout
	if tm != 0 {
		log.Warn("scheduling timeout", "timeout", tm, "at", time.Now().Add(tm).Format(time.RFC3339))
		c, maybenoop = context.WithTimeoutCause(c, tm, fmt.Errorf("on schedule"))
	}

	// then, various function callers (eg utils.Fatalf or common/log.Fatal)
	c, cancelCause = context.WithCancelCause(c)

	// finally signals which does not set cause
	c, stopSignals = signal.NotifyContext(c, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	multi := multicancelcause{
		cancel1: cancelCause,
		cancels: []context.CancelFunc{maybenoop, stopSignals},
	}
	log.RegisterCancelCause(multi.CancelCause) // when common/log.Fatal is called, this will be called
	return c, cancelCause
}

// helper to free all the resources attached to contexts
type multicancelcause struct {
	cancel1 context.CancelCauseFunc // the only one exposed to callers, the first one cancelled
	cancels []context.CancelFunc
}

func (x multicancelcause) CancelCause(err error) {
	log.Warn("shutting down everything: interrupted", "err", err)
	x.cancel1(err)
	for _, c := range x.cancels {
		if c != nil {
			c()
		}
	}
}

func maincancel(err error) {
	log.Trace("calling main cancel: interrupted", "err", err)
	maincancelreal(err)
}
