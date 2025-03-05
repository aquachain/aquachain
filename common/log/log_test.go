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
	limit := 100000
	wg.Add(limit)
	for i := range limit {
		go func() {
			log.Warn("mic test 1 2", "NoSync", log.NoSync, "t", i)
			wg.Done()
		}()
	}
	wg.Wait()
}
