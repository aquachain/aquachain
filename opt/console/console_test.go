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

package console

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"
	mrand "math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"gitlab.com/aquachain/aquachain/aqua"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"
	"gitlab.com/aquachain/aquachain/core"
	"gitlab.com/aquachain/aquachain/internal/jsre"
	"gitlab.com/aquachain/aquachain/node"
	"gitlab.com/aquachain/aquachain/p2p"
	"gitlab.com/aquachain/aquachain/params"
)

const (
	testInstancePrefix = "TestWelcome"
	testAddress        = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
)

// hookedPrompter implements UserPrompter to simulate use input via channels.
type hookedPrompter struct {
	scheduler chan string
}

func (p *hookedPrompter) PromptInput(prompt string) (string, error) {
	// Send the prompt to the tester
	select {
	case p.scheduler <- prompt:
	case <-time.After(time.Second):
		return "", errors.New("prompt timeout")
	}
	// Retrieve the response and feed to the console
	select {
	case input := <-p.scheduler:
		return input, nil
	case <-time.After(time.Second):
		return "", errors.New("input timeout")
	}
}

func (p *hookedPrompter) PromptPassword(prompt string) (string, error) {
	return "", errors.New("not implemented")
}
func (p *hookedPrompter) PromptConfirm(prompt string) (bool, error) {
	return false, errors.New("not implemented")
}
func (p *hookedPrompter) SetHistory(history []string)              {}
func (p *hookedPrompter) AppendHistory(command string)             {}
func (p *hookedPrompter) ClearHistory()                            {}
func (p *hookedPrompter) SetWordCompleter(completer WordCompleter) {}

// tester is a console test environment for the console tests to operate on.
type tester struct {
	workspace string
	stack     *node.Node
	aquachain *aqua.Aquachain
	console   *Console
	input     *hookedPrompter
	output    *bytes.Buffer
}

// newTester creates a test environment based on which the console can operate.
// Please ensure you call Close() on the returned tester to avoid leaks.
func newTester(t *testing.T, confOverride func(*aqua.Config)) *tester {
	// Create a temporary storage for the node keys and initialize it
	workspace, err := os.MkdirTemp("", "console-tester-")
	if err != nil {
		t.Fatalf("failed to create temporary keystore: %v", err)
	}

	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second*10, fmt.Errorf("test timeout"))
	defer cancel()

	chainId := mrand.Uint64()%1000 + 100

	// inject a chaincfg
	chaincfg := &params.ChainConfig{}
	*chaincfg = *params.TestChainConfig
	chaincfg.ChainId = new(big.Int).SetUint64(chainId)
	params.AddChainConfig(t.Name(), chaincfg)

	// Create a networkless protocol stack and start an Aquachain service within
	stack, err := node.New(&node.Config{
		Context:           ctx,
		CloseMain:         func(err error) { panic(err.Error()) },
		DataDir:           workspace,
		UseLightweightKDF: true,
		Name:              t.Name(),
		P2P:               &p2p.Config{ChainId: chainId},
		RPCAllowIP:        []string{"127.0.0.1/32"},
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	genesis := core.DeveloperGenesisBlock(15, common.Address{})
	genesis.Config = chaincfg // inject chaincfg into genesis
	ethConf := &aqua.Config{
		Genesis:  genesis,
		Aquabase: common.HexToAddress(testAddress),
		Aquahash: &aquahash.Config{
			PowMode: aquahash.ModeTest,
		},
		ChainId: chainId,
	}
	if confOverride != nil {
		confOverride(ethConf)
	}

	p2pnodename := func() string {
		def := node.NewDefaultConfig()
		def.Name = t.Name()
		return def.NodeName()
	}
	if err = stack.Register(func(nodectx *node.ServiceContext) (node.Service, error) {
		return aqua.New(ctx, nodectx, ethConf, p2pnodename())
	}); err != nil {
		t.Fatalf("failed to register Aquachain protocol: %v", err)
	}
	// Start the node and assemble the JavaScript console around it
	if err = stack.Start(ctx); err != nil {
		t.Fatalf("failed to start test stack: %v", err)
	}
	client, err := stack.Attach(ctx, "newTester")
	if err != nil {
		t.Fatalf("failed to attach to node: %v", err)
	}
	prompter := &hookedPrompter{scheduler: make(chan string)}
	printer := new(bytes.Buffer)

	console, err := New(Config{
		DataDir:          stack.DataDir(),
		WorkingDirectory: "testdata",
		Client:           client,
		Prompter:         prompter,
		Printer:          printer,
		Preload:          []string{"preload.js"},
	})
	if err != nil {
		t.Fatalf("failed to create JavaScript console: %v", err)
	}
	// Create the final tester and return
	var aquachain *aqua.Aquachain
	stack.Service(&aquachain)

	return &tester{
		workspace: workspace,
		stack:     stack,
		aquachain: aquachain,
		console:   console,
		input:     prompter,
		output:    printer,
	}
}

// Close cleans up any temporary data folders and held resources.
func (env *tester) Close(t *testing.T) {
	if err := env.console.Stop(false); err != nil {
		t.Errorf("failed to stop embedded console: %v", err)
	}
	if err := env.stack.Stop(); err != nil {
		t.Errorf("failed to stop embedded node: %v", err)
	}
	os.RemoveAll(env.workspace)
}

// Tests that the node lists the correct welcome message, notably that it contains
// the instance name, coinbase account, block number, data directory and supported
// console modules.
func TestWelcome(t *testing.T) {
	tester := newTester(t, nil)
	defer tester.Close(t)

	tester.console.Welcome()

	output := tester.output.String()
	if want := "Welcome"; !strings.Contains(output, want) {
		t.Fatalf("console output missing welcome message: have\n%s\nwant also %s", output, want)
	}
	if want := fmt.Sprintf("instance: %s", testInstancePrefix); !strings.Contains(output, want) {
		t.Fatalf("console output missing instance: have\n%s\nwant also %s", output, want)
	}
	if want := fmt.Sprintf("coinbase: %s", testAddress); !strings.Contains(output, want) {
		t.Fatalf("console output missing coinbase: have\n%s\nwant also %s", output, want)
	}
	if want := "at block: 0"; !strings.Contains(output, want) {
		t.Fatalf("console output missing sync status: have\n%s\nwant also %s", output, want)
	}
	if want := fmt.Sprintf("datadir: %s", tester.workspace); !strings.Contains(output, want) {
		t.Fatalf("console output missing coinbase: have\n%s\nwant also %s", output, want)
	}
}

// Tests that JavaScript statement evaluation works as intended.
func TestEvaluate(t *testing.T) {
	tester := newTester(t, nil)
	defer tester.Close(t)

	tester.console.Evaluate("2 + 2")
	if output := tester.output.String(); !strings.Contains(output, "4") {
		t.Fatalf("statement evaluation failed: have %s, want %s", output, "4")
	}
}

// Tests that the console can be used in interactive mode.
func TestInteractive(t *testing.T) {
	// Create a tester and run an interactive console in the background
	tester := newTester(t, nil)
	defer tester.Close(t)

	go tester.console.Interactive(context.Background())

	// Wait for a promt and send a statement back
	select {
	case <-tester.input.scheduler:
	case <-time.After(time.Second):
		t.Fatalf("initial prompt timeout")
	}
	select {
	case tester.input.scheduler <- "2+2":
	case <-time.After(time.Second):
		t.Fatalf("input feedback timeout")
	}
	// Wait for the second promt and ensure first statement was evaluated
	select {
	case <-tester.input.scheduler:
	case <-time.After(time.Second):
		t.Fatalf("secondary prompt timeout")
	}
	if output := tester.output.String(); !strings.Contains(output, "4") {
		t.Fatalf("statement evaluation failed: have %s, want %s", output, "4")
	}
}

// Tests that preloaded JavaScript files have been executed before user is given
// input.
func TestPreload(t *testing.T) {
	tester := newTester(t, nil)
	defer tester.Close(t)

	tester.console.Evaluate("preloaded")
	if output := tester.output.String(); !strings.Contains(output, "some-preloaded-string") {
		t.Fatalf("preloaded variable missing: have %s, want %s", output, "some-preloaded-string")
	}
}
func testerr(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("failed to execute statement: %v", err)
	}
}

// Tests that JavaScript scripts can be executes from the configured asset path.
func TestExecute(t *testing.T) {
	var err error
	tester := newTester(t, nil)
	defer tester.Close(t)

	tester.console.ExecuteFile("exec.js")

	tester.console.Evaluate("execed")
	if output := tester.output.String(); !strings.Contains(output, "some-executed-string") {
		t.Fatalf("execed variable missing: have %s, want %s", output, "some-executed-string")
	}
	tester.output.Reset()
	tester.console.ExecuteFile("testwei.js")
	err = tester.console.Evaluate("txwei")
	testerr(t, err)
	if output := tester.output.String(); !strings.Contains(output, "1000000000000000000") {
		t.Fatalf("execed variable missing: have %s, want %s", output, "1000000000000000000")
	}

	tester.output.Reset()
	err = tester.console.Evaluate(`tx = {from: aqua.coinbase, to: '0xDA7064FB41A2a599275Dd74113787A7aA8ee3E4f', value: web3.toWei(1,'aqua')}`)
	testerr(t, err)
	err = tester.console.Evaluate("tx")
	log.Printf("tx: %v", tester.output.String())
}

// Tests that the JavaScript objects returned by statement executions are properly
// pretty printed instead of just displaing "[object]".
func TestPrettyPrint(t *testing.T) {
	tester := newTester(t, nil)
	defer tester.Close(t)

	tester.console.Evaluate("obj = {int: 1, string: 'two', list: [3, 3, 3], obj: {null: null, func: function(){}}}")

	// Define some specially formatted fields
	var (
		one   = jsre.NumberColor("1")
		two   = jsre.StringColor("\"two\"")
		three = jsre.NumberColor("3")
		null  = jsre.SpecialColor("null")
		fun   = jsre.FunctionColor("function()")
	)
	// Assemble the actual output we're after and verify
	want := `{
  int: ` + one + `,
  list: [` + three + `, ` + three + `, ` + three + `],
  obj: {
    null: ` + null + `,
    func: ` + fun + `
  },
  string: ` + two + `
}
`
	if output := tester.output.String(); output != want {
		t.Fatalf("pretty print mismatch: have %s, want %s", output, want)
	}
}

// Tests that the JavaScript exceptions are properly formatted and colored.
func TestPrettyError(t *testing.T) {
	tester := newTester(t, nil)
	defer tester.Close(t)
	tester.console.Evaluate("throw 'hello'")

	want := jsre.ErrorColor("hello") + "\n"
	if output := tester.output.String(); output != want {
		t.Fatalf("pretty error mismatch: have %q, want %q", output, want)
	}
}

// Tests that tests if the number of indents for JS input is calculated correct.
func TestIndenting(t *testing.T) {
	testCases := []struct {
		input               string
		expectedIndentCount int
	}{
		{`var a = 1;`, 0},
		{`"some string"`, 0},
		{`"some string with (parentesis`, 0},
		{`"some string with newline
		("`, 0},
		{`function v(a,b) {}`, 0},
		{`function f(a,b) { var str = "asd("; };`, 0},
		{`function f(a) {`, 1},
		{`function f(a, function(b) {`, 2},
		{`function f(a, function(b) {
		     var str = "a)}";
		  });`, 0},
		{`function f(a,b) {
		   var str = "a{b(" + a, ", " + b;
		   }`, 0},
		{`var str = "\"{"`, 0},
		{`var str = "'("`, 0},
		{`var str = "\\{"`, 0},
		{`var str = "\\\\{"`, 0},
		{`var str = 'a"{`, 0},
		{`var obj = {`, 1},
		{`var obj = { {a:1`, 2},
		{`var obj = { {a:1}`, 1},
		{`var obj = { {a:1}, b:2}`, 0},
		{`var obj = {}`, 0},
		{`var obj = {
			a: 1, b: 2
		}`, 0},
		{`var test = }`, -1},
		{`var str = "a\""; var obj = {`, 1},
	}

	for i, tt := range testCases {
		counted := countIndents(tt.input)
		if counted != tt.expectedIndentCount {
			t.Errorf("test %d: invalid indenting: have %d, want %d", i, counted, tt.expectedIndentCount)
		}
	}
}

func TestBigSmall(t *testing.T) {
	// test web3.fromWei and web3.toWei
	input := "1234500000000000000000" // wei
	output := "1234.5"                // coin
	bigconsole := newTester(t, nil)
	defer bigconsole.Close(t)
	// jsre := New("", os.Stdout)
	// defer jsre.Stop(false)

	jsre := bigconsole.console.jsre
	var err error
	_, err = jsre.Run("var big = new BigNumber('" + input + "');")
	if err != nil {
		t.Fatal("cannot run big:", err)
	}

	_, err = jsre.Run("var small = web3.fromWei(big, 'aqua');")
	if err != nil {
		t.Fatal("cannot run fromWei:", err)
	}

	val, err := jsre.Run("small.toString();")
	if err != nil {
		t.Fatal("cannot run small:", err)
	}

	if !val.IsString() {
		t.Errorf("expected string value, got %v", val)
	}

	got, _ := val.ToString()
	if output != got {
		t.Errorf("expected '%v', got '%v'", output, got)
	}

	// now back to wei

	_, err = jsre.Run("var big = web3.toWei(\"" + output + "\", 'aqua');")
	if err != nil {
		t.Fatal("cannot run toWei:", err)
	}

	val, err = jsre.Run("big.toString();")
	if err != nil {
		t.Fatal("cannot run big:", err)
	}

	if !val.IsString() {
		t.Errorf("expected string value, got %v", val)
	}

	got, _ = val.ToString()
	if input != got {
		t.Errorf("expected '%v', got '%v'", input, got)
	}
	println(got)

}

func TestBigSmall2(t *testing.T) {
	input := "1234500000000000000000" // wei
	output := "1234.5"                // coin
	bigconsole := newTester(t, nil)
	defer bigconsole.Close(t)
	// jsre := New("", os.Stdout)
	// defer jsre.Stop(false)

	jsre := bigconsole.console.jsre
	var err error
	_, err = jsre.Run("var big = new BigNumber('" + input + "');")
	if err != nil {
		t.Fatal("cannot run big:", err)
	}

	_, err = jsre.Run("var small = web3.fromWei(big, 'aqua');")
	if err != nil {
		t.Fatal("cannot run fromWei:", err)
	}

	var jstest = `
var inwei = '` + input + `';
var incoin = '` + output + `';
var small = web3.fromWei(inwei, 'aqua');
var big = web3.toWei(incoin, 'aqua');
if (small.toString() != incoin) throw 'small ' + small.toString() + ' ' + incoin;
if (big.toString() != inwei) throw 'big ' + big.toString() + ' ' + inwei;
`
	_, err = jsre.Run(jstest)
	if err != nil {
		t.Fatal("error:", err)
	}

}
