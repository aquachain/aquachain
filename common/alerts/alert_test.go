package alerts

import (
	"os"
	"sync"
	"testing"

	"github.com/joho/godotenv"
)

func TestAlert(t *testing.T) {
	if _, err := os.Stat("../../Makefile"); err != nil {
		t.Fatal("bad test; only run in root directory")
	}
	err := godotenv.Load("../../.env") // we are in ./common/alerts/
	if err != nil {
		t.Skipf("expected no error, got %v", err)
		return
	}
	ac, err := ParseAlertConfig()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !ac.Enabled() {
		t.Fatalf("expected enabled, got disabled")
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	err = ac.Send("(test-alert-1) chain is on fire! send help! [aaaaahhhhh!!!] [  ! .* <b>wow!</b>", wg.Done)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	wg.Wait()
	println("Alert Sent?")
	// if fails with new token, make sure you "/start" the "bot" first, or Channel is correct
}
