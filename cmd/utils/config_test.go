package utils

import (
	"log"
	"strings"
	"testing"
)

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
