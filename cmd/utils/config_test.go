package utils

import (
	"strings"
	"testing"

	"gitlab.com/aquachain/aquachain/common/log"
)

func init() {
	log.ResetForTesting()
}
func TestDefaultNodeConfig(t *testing.T) {
	got := DefaultNodeConfig("abcdefg", "aquachain")
	log.Printf("got: %#v", got.P2P.Name)
	if got.P2P.Name == "" {
		t.Errorf("P2P.Name should not be empty after DefaultNodeConfig")
		return
	}
	if strings.Count(got.P2P.Name, "/") != 3 {
		t.Errorf("P2P.Name should contain / after DefaultNodeConfig got %q", got.P2P.Name)
	}

}
