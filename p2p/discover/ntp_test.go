package discover

import (
	"testing"
	"time"

	"gitlab.com/aquachain/aquachain/common/sense"
)

func TestNtp(t *testing.T) {
	if !sense.EnvBool("TEST_NTP") {
		t.Skipf("skipping NTP test, run manually with\n\tTEST_NTP=1 go test -v -run TestNtp ./p2p/discover")
	}
	offset, err := sntpDrift(ntpChecks)
	if err != nil {
		t.Fatalf("sntpDrift failed: %v", err)
	}
	t.Logf("NTP offset: %v", offset)
	checkClockDrift()
	time.Sleep(1 * time.Second)
	println("OK")
}
