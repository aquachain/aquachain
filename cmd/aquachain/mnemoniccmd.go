package main

import (
	"fmt"

	cli "github.com/urfave/cli"
	"gitlab.com/aquachain/aquachain/crypto/mnemonics"
)

// accountGenerateMnemonic prints a phrase
func accountGenerateMnemonic(ctx *cli.Context) error {
	phrase := mnemonics.Generate()
	fmt.Printf("WRITE THIS DOWN: %v\n", phrase)
	return nil
}
