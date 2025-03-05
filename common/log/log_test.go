package log_test

import (
	"sync"
	"testing"

	"gitlab.com/aquachain/aquachain/common/log"
)

// with with NO_LOGSYNC=off or NO_LOGSYNC= to see the difference
func TestXxx(t *testing.T) {
	logger := log.New("module", "log_test")
	logger.Warn("mic test 1 2", "NoSync", log.NoSync)
	log.Warn("mic test 1 2", "NoSync", log.NoSync)
	var wg sync.WaitGroup
	// limit := 100000
	limit := 10
	wg.Add(limit)
	for i := range limit {
		go func() {
			log.Warn("mic test 1 2", "NoSync", log.NoSync, "t", i)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestPrintf(t *testing.T) {
	log.Printf("test %d %d", 1, 2)

	oldlogger := log.Root()
	oldhandler := oldlogger.GetHandler()
	// should go to text logging >= WARN level only, depending on TESTLOGLVL env var
	// eg: TESTLOGLVL=9 go test  -count=1  -v ./common/log
	//     TESTLOGLVL=3 go test  -count=1  -v ./common/log
	log.ResetForTesting()
	log.Error("hello error level")
	log.Warn("hello warn level")
	log.Info("hello info level")
	log.Debug("hello debug level")
	log.Trace("hello trace level")

	// should go back to json logging with all levels
	oldlogger.SetHandler(oldhandler)
	log.SetRoot(oldlogger)

	log.Error("hello2 error level")
	log.Warn("hello2 warn level")
	log.Info("hello2 info level")
	log.Debug("hello2 debug level")
	log.Trace("hello2 trace level")

	// log.Crit("hello crit level") // ok to run at the end of this function, but fails the test
}
