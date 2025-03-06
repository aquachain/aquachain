package main

import (
	"context"
	"errors"
	"fmt"

	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/urfave/cli/v3"
	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common/log"
	"gitlab.com/aquachain/aquachain/internal/debug"
)

var errMainQuit = errors.New("run success")

var appconfig = struct {
	chain string
}{
	chain: "testnet3",
}

func main() {
	app := &cli.Command{
		Name:  "newtester",
		Usage: "newtester is a command line tool for testing",
		Flags: append([]cli.Flag{
			utils.ChainFlag,
			utils.DoitNowFlag,
			utils.ConfigFileFlag,
			utils.DataDirFlag}, debug.Flags...),
		Action: func(ctx context.Context, cmd *cli.Command) error { // cli.ActionFunc
			for ctx.Err() == nil {
				log.Infof("reticulating splines (name=%s chain=%s)", cmd.Name, appconfig.chain)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(2 * time.Second):
				}
			}
			return nil
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			appconfig.chain = cmd.String(utils.ChainFlag.Name)
			return ctx, nil
		},
		Commands: []*cli.Command{},
	}

	ctx, cancelcause := context.WithCancelCause(context.Background())
	defer cancelcause(errMainQuit)

	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
		sig := <-ch
		cancelcause(fmt.Errorf("received signal %s", sig))
	}()

	app.Run(ctx, os.Args)

	err := context.Cause(ctx)
	if err != nil && err != errMainQuit {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
