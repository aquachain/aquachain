package subcommands

import (
	"context"
	"fmt"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/crypto/mnemonics"
)

// accountGenerateMnemonic prints a phrase
func accountGenerateMnemonic(_ context.Context, cmd *cli.Command) error {
	phrase := mnemonics.Generate()
	fmt.Printf("WRITE THIS DOWN: %v\n", phrase)
	return nil
}
