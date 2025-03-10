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
	"fmt"

	cli "github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/aqua/accounts/keystore"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/opt/console"
	"gitlab.com/aquachain/aquachain/subcommands/aquaflags"
	"gitlab.com/aquachain/aquachain/subcommands/mainctxs"
)

var (
	accountCommand = &cli.Command{
		Name:     "account",
		Usage:    "Manage accounts",
		Category: "ACCOUNT COMMANDS",
		Description: `

Manage accounts, list all existing accounts, import a private key into a new
account, create a new account or update an existing account.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keystore.
It is safe to transfer the entire directory or the individual keys therein
between aquachain nodes by simply copying.

Make sure you backup your keys regularly.`,
		Commands: []*cli.Command{
			{
				Name:   "list",
				Usage:  "Print summary of existing accounts",
				Action: MigrateFlags(accountList),
				Flags: []cli.Flag{
					aquaflags.DataDirFlag,
					aquaflags.KeyStoreDirFlag,
				},
				Description: `
Print a short summary of all accounts`,
			},
			{
				Name:   "new",
				Usage:  "Create a new account",
				Action: MigrateFlags(accountCreate),
				Flags: []cli.Flag{
					aquaflags.DataDirFlag,
					aquaflags.KeyStoreDirFlag,
					aquaflags.PasswordFileFlag,
				},
				Description: `
    aquachain account new

Creates a new account and prints the address.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the --password flag:

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
`,
			}, {
				Name:   "generatePhrase",
				Usage:  "Create a new mnemonic account",
				Action: MigrateFlags(accountGenerateMnemonic),
				Flags: []cli.Flag{
					aquaflags.DataDirFlag,
					aquaflags.KeyStoreDirFlag,
					aquaflags.PasswordFileFlag,
				},
				Description: `
    This only prints! Does not store key.
`,
			},
			{
				Name:      "update",
				Usage:     "Update an existing account",
				Action:    MigrateFlags(accountUpdate),
				ArgsUsage: "<address>",
				Flags: []cli.Flag{
					aquaflags.DataDirFlag,
					aquaflags.KeyStoreDirFlag,
				},
				Description: `
    aquachain account update <address>

Update an existing account.

The account is saved in the newest version in encrypted format, you are prompted
for a passphrase to unlock the account and another to save the updated file.

This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.

For non-interactive use the passphrase can be specified with the --password flag:

    aquachain account update [options] <address>

Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.
`,
			},
			{
				Name:   "import",
				Usage:  "Import a private key into a new account",
				Action: MigrateFlags(accountImport),
				Flags: []cli.Flag{
					aquaflags.DataDirFlag,
					aquaflags.KeyStoreDirFlag,
					aquaflags.PasswordFileFlag,
				},
				ArgsUsage: "<keyFile>",
				Description: `
    aquachain account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to contain an unencrypted private key in hexadecimal format.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the -password flag:

    aquachain account import [options] <keyfile>

Note:
As you can directly copy your encrypted accounts to another aquachain instance,
this import mechanism is not needed when you transfer an account between
nodes.
`,
			},
		},
	}
)

func accountList(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool(aquaflags.NoKeysFlag.Name) {
		Fatalf("Listing accounts is disabled (-nokeys)")
	}
	stack, _ := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, mainctxs.MainCancelCause())
	var index int
	for _, wallet := range stack.AccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			fmt.Printf("Account #%d: 0x%x %s\n", index, account.Address, &account.URL)
			index++
		}
	}
	return nil
}

// tries unlocking the specified account a few times.
func unlockAccount(cmd *cli.Command, ks *keystore.KeyStore, address string, i int, passwords []string) (accounts.Account, string) {
	if cmd.Bool(aquaflags.NoKeysFlag.Name) {
		Fatalf("Unlocking accounts is disabled")
	}
	account, err := MakeAddress(ks, address)
	if err != nil {
		Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = ks.Unlock(account, password)
		if err == nil {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return account, password
		}
		if err, ok := err.(*keystore.AmbiguousAddrError); ok {
			log.Info("Unlocked account", "address", account.Address.Hex())
			return ambiguousAddrRecovery(ks, err, password), password
		}
		if err != keystore.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	Fatalf("Failed to unlock account %s (%v)", address, err)

	return accounts.Account{}, ""
}

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(ks *keystore.KeyStore, err *keystore.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.URL)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for i := range err.Matches {
		if errr := ks.Unlock(err.Matches[i], auth); errr == nil {
			match = &err.Matches[i]
			break
		}
	}
	if match == nil {
		Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.URL)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.URL)
		}
	}
	return *match
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(_ context.Context, cmd *cli.Command) error {
	cfg := AquachainConfig{Node: DefaultNodeConfig(gitCommit, clientIdentifier)}
	// Load config file.
	if file := cmd.String(aquaflags.ConfigFileFlag.Name); file != "" {
		if err := LoadConfigFromFile(file, &cfg); err != nil {
			Fatalf("%v", err)
		}
	}
	SetNodeConfig(cmd, cfg.Node)
	scryptN, scryptP, keydir, err := cfg.Node.AccountConfig()

	if err != nil {
		Fatalf("Failed to read configuration: %v", err)
	}

	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password. Backup your keystore directory.", true, 0, MakePasswordList(cmd))

	address, err := keystore.StoreKey(keydir, password, scryptN, scryptP)

	if err != nil {
		Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Address: {0x%x}\n", address)
	return nil
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() == 0 {
		Fatalf("No accounts specified to update")
	}
	stack, _ := MakeConfigNode(ctx, cmd, gitCommit, clientIdentifier, mainctxs.MainCancelCause())
	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	for _, addr := range cmd.Args().Slice() {
		account, oldPassword := unlockAccount(cmd, ks, addr, 0, nil)
		newPassword := getPassPhrase("Please give a new password. Do not forget this password.", true, 0, nil)
		if err := ks.Update(account, oldPassword, newPassword); err != nil {
			Fatalf("Could not update the account: %v", err)
		}
	}
	return nil
}

func accountImport(ctx context.Context, cmd *cli.Command) error {
	keyfile := cmd.Args().First()
	if len(keyfile) == 0 {
		Fatalf("keyfile must be given as argument")
	}
	key, err := crypto.LoadECDSA(keyfile)
	if err != nil {
		Fatalf("Failed to load the private key: %v", err)
	}
	stack, _ := MakeConfigNode(ctx, cmd, cmd.String("gitCommit"), cmd.String("clientIdentifier"), mainctxs.MainCancelCause())
	passphrase := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, MakePasswordList(cmd))

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acct, err := ks.ImportECDSA(key, passphrase)
	if err != nil {
		Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}
