// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

package keystore

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/cespare/cp"
	"github.com/davecgh/go-spew/spew"
	"gitlab.com/aquachain/aquachain/aqua/accounts"
	"gitlab.com/aquachain/aquachain/common"
)

func getTestDir() (string, string) {
	var basedir, _ = filepath.Abs(filepath.Join("..", "..", "..", "testdata", "testkeystore")) // test changes with TestTestDirs
	var cachetestDir, _ = filepath.Abs(filepath.Join(basedir, "keystore"))                     // test changes with TestTestDirs

	return basedir, cachetestDir
}

var (
	basedir, cachetestDir = getTestDir()
	cachetestAccounts     = []accounts.Account{
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(cachetestDir, "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8")},
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(cachetestDir, "aaa")},
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(cachetestDir, "zzz")},
		},
	}
)

func TestTestDirs(t *testing.T) {
	for _, filename := range []string{
		basedir,
		cachetestDir,
		filepath.Join(cachetestDir, "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
		filepath.Join(cachetestDir, "aaa"),
		filepath.Join(cachetestDir, "zzz"),
	} {
		if _, err := os.Stat(filename); err != nil {
			t.Errorf("test path %s does not exist", filename)
			println(err.Error())
			os.Exit(1) // this is a fatal error
		}
	}
}

func TestCacheInitialReload(t *testing.T) {
	cache, _ := newAccountCache(cachetestDir)
	accounts := cache.accounts()
	if !reflect.DeepEqual(accounts, cachetestAccounts) {
		t.Fatalf("got initial accounts: %swant %s", spew.Sdump(accounts), spew.Sdump(cachetestAccounts))
	}
}

func TestCacheAddDeleteOrder(t *testing.T) {
	cache, _ := newAccountCache(filepath.Join(basedir, "no-such-dir"))
	cache.watcher.running = true // prevent unexpected reloads

	accs := []accounts.Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "-309830980"},
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "ggg"},
		},
		{
			Address: common.HexToAddress("8bda78331c916a08481428e4b07c96d3e916d165"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "zzzzzz-the-very-last-one.keyXXX"},
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "SOMETHING.key"},
		},
		{
			Address: common.HexToAddress("7ef5a6135f1fd6a02593eedc869c6d41d934aef8"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8"},
		},
		{
			Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "aaa"},
		},
		{
			Address: common.HexToAddress("289d485d9771714cce91d3393d764e1311907acc"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: "zzz"},
		},
	}
	for _, a := range accs {
		cache.add(a)
	}
	// Add some of them twice to check that they don't get reinserted.
	cache.add(accs[0])
	cache.add(accs[2])

	// Check that the account list is sorted by filename.
	wantAccounts := make([]accounts.Account, len(accs))
	copy(wantAccounts, accs)
	sort.Sort(accountsByURL(wantAccounts))
	list := cache.accounts()
	if !reflect.DeepEqual(list, wantAccounts) {
		t.Fatalf("got accounts: %s\nwant %s", spew.Sdump(accs), spew.Sdump(wantAccounts))
	}
	for _, a := range accs {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e")) {
		t.Errorf("expected hasAccount(%x) to return false", common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"))
	}

	// Delete a few keys from the cache.
	for i := 0; i < len(accs); i += 2 {
		cache.delete(wantAccounts[i])
	}
	cache.delete(accounts.Account{Address: common.HexToAddress("fd9bd350f08ee3c0c19b85a8e16114a11a60aa4e"), URL: accounts.URL{Scheme: KeyStoreScheme, Path: "something"}})

	// Check content again after deletion.
	wantAccountsAfterDelete := []accounts.Account{
		wantAccounts[1],
		wantAccounts[3],
		wantAccounts[5],
	}
	list = cache.accounts()
	if !reflect.DeepEqual(list, wantAccountsAfterDelete) {
		t.Fatalf("got accounts after delete: %s\nwant %s", spew.Sdump(list), spew.Sdump(wantAccountsAfterDelete))
	}
	for _, a := range wantAccountsAfterDelete {
		if !cache.hasAddress(a.Address) {
			t.Errorf("expected hasAccount(%x) to return true", a.Address)
		}
	}
	if cache.hasAddress(wantAccounts[0].Address) {
		t.Errorf("expected hasAccount(%x) to return false", wantAccounts[0].Address)
	}
}

func TestCacheFind(t *testing.T) {
	dir := filepath.Join(basedir, "dir")
	cache, _ := newAccountCache(dir)
	cache.watcher.running = true // prevent unexpected reloads

	accs := []accounts.Account{
		{
			Address: common.HexToAddress("095e7baea6a6c7c4c2dfeb977efac326af552d87"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(dir, "a.key")},
		},
		{
			Address: common.HexToAddress("2cac1adea150210703ba75ed097ddfe24e14f213"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(dir, "b.key")},
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(dir, "c.key")},
		},
		{
			Address: common.HexToAddress("d49ff4eeb0b2686ed89c0fc0f2b6ea533ddbbd5e"),
			URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(dir, "c2.key")},
		},
	}
	for _, a := range accs {
		cache.add(a)
	}

	nomatchAccount := accounts.Account{
		Address: common.HexToAddress("f466859ead1932d743d622cb74fc058882e8648a"),
		URL:     accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Join(dir, "something")},
	}
	tests := []struct {
		Query      accounts.Account
		WantResult accounts.Account
		WantError  error
	}{
		// by address
		{Query: accounts.Account{Address: accs[0].Address}, WantResult: accs[0]},
		// by file
		{Query: accounts.Account{URL: accs[0].URL}, WantResult: accs[0]},
		// by basename
		{Query: accounts.Account{URL: accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Base(accs[0].URL.Path)}}, WantResult: accs[0]},
		// by file and address
		{Query: accs[0], WantResult: accs[0]},
		// ambiguous address, tie resolved by file
		{Query: accs[2], WantResult: accs[2]},
		// ambiguous address error
		{
			Query: accounts.Account{Address: accs[2].Address},
			WantError: &AmbiguousAddrError{
				Addr:    accs[2].Address,
				Matches: []accounts.Account{accs[2], accs[3]},
			},
		},
		// no match error
		{Query: nomatchAccount, WantError: ErrNoMatch},
		{Query: accounts.Account{URL: nomatchAccount.URL}, WantError: ErrNoMatch},
		{Query: accounts.Account{URL: accounts.URL{Scheme: KeyStoreScheme, Path: filepath.Base(nomatchAccount.URL.Path)}}, WantError: ErrNoMatch},
		{Query: accounts.Account{Address: nomatchAccount.Address}, WantError: ErrNoMatch},
	}
	for i, test := range tests {
		a, err := cache.find(test.Query)
		if !reflect.DeepEqual(err, test.WantError) {
			t.Errorf("test %d: error mismatch for query %v\ngot %q\nwant %q", i, test.Query, err, test.WantError)
			continue
		}
		if a != test.WantResult {
			t.Errorf("test %d: result mismatch for query %v\ngot %v\nwant %v", i, test.Query, a, test.WantResult)
			continue
		}
	}
}

func waitForAccounts(wantAccounts []accounts.Account, ks *KeyStore) error {
	var list []accounts.Account
	for d := 200 * time.Millisecond; d < 8*time.Second; d *= 2 {
		list = ks.Accounts()
		if reflect.DeepEqual(list, wantAccounts) {
			// ks should have also received change notifications
			select {
			case <-ks.changes:
			default:
				return fmt.Errorf("wasn't notified of new accounts")
			}
			return nil
		}
		time.Sleep(d)
	}
	return fmt.Errorf("\ngot  %v\nwant %v", list, wantAccounts)
}

// TestUpdatedKeyfileContents tests that updating the contents of a keystore file
// is noticed by the watcher, and the account cache is updated accordingly
func TestUpdatedKeyfileContents(t *testing.T) {
	t.Skip()
	t.Parallel()

	// Create a temporary kesytore to test with
	rand.Seed(time.Now().UnixNano())
	dir := filepath.Join(os.TempDir(), fmt.Sprintf("aqua-keystore-watch-test-%d-%d", os.Getpid(), rand.Int()))
	ks := NewKeyStore(dir, LightScryptN, LightScryptP)

	list := ks.Accounts()
	if len(list) > 0 {
		t.Error("initial account list not empty:", list)
	}
	time.Sleep(100 * time.Millisecond)

	// Create the directory and copy a key file into it.
	os.MkdirAll(dir, 0700)
	defer os.RemoveAll(dir)
	file := filepath.Join(dir, "aaa")

	// Place one of our testfiles in there
	if err := cp.CopyFile(file, cachetestAccounts[0].URL.Path); err != nil {
		t.Fatal(err)
	}

	// ks should see the account.
	wantAccounts := []accounts.Account{cachetestAccounts[0]}
	wantAccounts[0].URL = accounts.URL{Scheme: KeyStoreScheme, Path: file}
	if err := waitForAccounts(wantAccounts, ks); err != nil {
		t.Error(err)
		return
	}

	// needed so that modTime of `file` is different to its current value after forceCopyFile
	time.Sleep(1000 * time.Millisecond)

	// Now replace file contents
	if err := forceCopyFile(file, cachetestAccounts[1].URL.Path); err != nil {
		t.Fatal(err)
		return
	}
	wantAccounts = []accounts.Account{cachetestAccounts[1]}
	wantAccounts[0].URL = accounts.URL{Scheme: KeyStoreScheme, Path: file}
	if err := waitForAccounts(wantAccounts, ks); err != nil {
		t.Errorf("First replacement failed")
		t.Error(err)
		return
	}

	// needed so that modTime of `file` is different to its current value after forceCopyFile
	time.Sleep(1000 * time.Millisecond)

	// Now replace file contents again
	if err := forceCopyFile(file, cachetestAccounts[2].URL.Path); err != nil {
		t.Fatal(err)
		return
	}
	wantAccounts = []accounts.Account{cachetestAccounts[2]}
	wantAccounts[0].URL = accounts.URL{Scheme: KeyStoreScheme, Path: file}
	if err := waitForAccounts(wantAccounts, ks); err != nil {
		t.Errorf("Second replacement failed")
		t.Error(err)
		return
	}

	// needed so that modTime of `file` is different to its current value after ioutil.WriteFile
	time.Sleep(1000 * time.Millisecond)

	// Now replace file contents with crap
	if err := ioutil.WriteFile(file, []byte("foo"), 0644); err != nil {
		t.Fatal(err)
		return
	}
	if err := waitForAccounts([]accounts.Account{}, ks); err != nil {
		t.Errorf("Emptying account file failed")
		t.Error(err)
		return
	}
}

// forceCopyFile is like cp.CopyFile, but doesn't complain if the destination exists.
func forceCopyFile(dst, src string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(dst, data, 0644)
}
