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

// +build nousb

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	cli "github.com/urfave/cli"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/crypto"
)

var (
	walletCommand = cli.Command{
		Name:     "wallet",
		Usage:    `Launch MyAquaWallet, an offline wallet that connects to this aquachain program via JSON-RPC/HTTP`,
		Category: "ACCOUNT COMMANDS",
		Action:   launchmaw,
		Description: `
  aquachain wallet

Disabled in this version of Aquachain`,
	}
	paperCommand = cli.Command{
		Name:      "paper",
		Usage:     `Generate paper wallet keypair`,
		Flags:     []cli.Flag{utils.JsonFlag, utils.VanityFlag},
		ArgsUsage: "[number of wallets]",
		Category:  "ACCOUNT COMMANDS",
		Action:    paper,
		Description: `
Generate a number of wallets.`,
	}
)

func launchmaw(c *cli.Context) error {
	return fmt.Errorf("MyAquaWallet is not included in this version of Aquachain")
}

type paperWallet struct{ Private, Public string }

func paper(c *cli.Context) error {

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
	wallets := []paperWallet{}
	vanity := c.String("vanity")
	for i := 0; i < count; i++ {
		var wallet paperWallet
		for {
			key, err := crypto.GenerateKey()
			if err != nil {
				return err
			}

			addr := crypto.PubkeyToAddress(key.PublicKey)
			wallet = paperWallet{
				Private: hex.EncodeToString(crypto.FromECDSA(key)),
				Public:  "0x" + hex.EncodeToString(addr[:]),
			}
			if vanity == "" {
				break
			}
			pubkey := hex.EncodeToString(addr[:])
			if strings.HasPrefix(pubkey, vanity) {
				break
			}
		}
		if c.Bool("json") {
			wallets = append(wallets, wallet)
		} else {
			fmt.Println(wallet.Private, wallet.Public)
		}
	}
	if c.Bool("json") {
		b, _ := json.Marshal(wallets)
		fmt.Println(string(b))
	}
	return nil
}
