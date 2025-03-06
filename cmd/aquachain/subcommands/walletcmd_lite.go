// Copyright 2018 The aquachain Authors
// This file is part of aquachain.
//
// aquachain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// aquachain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with aquachain. If not, see <http://www.gnu.org/licenses/>.

package subcommands

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/cmd/aquachain/aquaflags"
	"gitlab.com/aquachain/aquachain/crypto"
)

var (
	paperCommand = &cli.Command{
		Name:      "paper",
		Usage:     `Generate paper wallet keypair`,
		Flags:     []cli.Flag{aquaflags.JsonFlag, aquaflags.VanityFlag, aquaflags.VanityEndFlag, paperthreadsFlag},
		ArgsUsage: "[number of wallets]",
		Category:  "ACCOUNT COMMANDS",
		Action:    paper,
		Description: `
Generate a number of wallets.`,
	}
	paperthreadsFlag = &cli.IntFlag{
		Name:  "threads",
		Usage: "number of threads to use for paper wallet generation",
	}
)

type paperWallet struct {
	Private string `json:"private_key"`
	Public  string `json:"address"`
}

func paper(ctx context.Context, c *cli.Command) error {
	log.SetFlags(0)
	if c.NArg() > 1 {
		return fmt.Errorf("too many arguments")
	}
	var (
		count = 1
		err   error
	)
	if c.NArg() == 1 {
		count, err = strconv.Atoi(c.Args().First())
		if err != nil {
			return err
		}
	}
	vanity := strings.ToLower(c.String("vanity"))
	vanityend := strings.ToLower(c.String("vanityend"))
	if !strings.HasPrefix(vanity, "0x") {
		vanity = "0x" + vanity
	}
	// check input
	combined := vanity[2:] + vanityend
	if len(combined)%2 != 0 {
		combined += "0"
	}
	_, err = hex.DecodeString(combined)
	if err != nil {
		return fmt.Errorf("fatal: must use hex characters: %v", err)
	}
	ch := make(chan paperWallet, 100)
	var found atomic.Int32
	limit := int32(count)
	threads := int(c.Int("threads"))
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	log.Printf("threads: %d", threads)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for thread := 0; thread < threads; thread++ {
		go func() {
			for found.Load() < limit && ctx.Err() == nil {
				var wallet paperWallet
				for {
					key, err := crypto.GenerateKey()
					if err != nil {
						panic(err.Error())
					}
					addr := crypto.PubkeyToAddress(key.PubKey()).Hex()
					wallet = paperWallet{
						Private: hex.EncodeToString(crypto.FromECDSA(key)),
						Public:  addr,
					}
					addr = strings.ToLower(addr)
					if strings.HasPrefix(addr, vanity) && strings.HasSuffix(addr, vanityend) {
						select {
						case <-ctx.Done():
							return
						case ch <- wallet:
						}
					}
				}
			}
			if ctx.Err() == nil {
				cancel()
				close(ch)
			}
		}()
	}
	dojson := c.Bool("json")
	for wallet := range ch {
		found.Add(1)
		if dojson {
			json.NewEncoder(os.Stdout).Encode(wallet)
		} else {
			fmt.Println(wallet.Private, wallet.Public)
		}
		if found.Load() >= limit {
			break
		}
	}
	return nil
}
