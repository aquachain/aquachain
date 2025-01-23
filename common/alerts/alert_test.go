package alerts

import (
	"testing"

	"github.com/joho/godotenv"
)

func TestAlert(t *testing.T) {
	godotenv.Load("../../.env") // we are in ./common/alerts/
	ac, err := ParseAlertConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if !ac.Enabled() {
		t.Errorf("expected enabled, got disabled")
	}
	err = ac.Send("(test-alert) chain is on fire! send help! [aaaaahhhhh!!!] [  !")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// if fails with new token, make sure you "/start" the "bot" first, or Channel is correct
}
