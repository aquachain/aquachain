package main

import (
	"compress/gzip"
	"encoding/hex"
	"fmt"
	"io"
	"os"

	"github.com/aquanetwork/aquachain/cmd/utils"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit  string
	app        = utils.NewApp(gitCommit, "aquahex encoder")
	OutputFlag = cli.StringFlag{
		Name:  "o",
		Usage: "output to file instead of stdout",
	}
	InputFlag = cli.StringFlag{
		Name:  "i",
		Usage: "input file(s) instead of stdin",
	}
	DecodeFlag = cli.BoolFlag{
		Name:  "d",
		Usage: "decode (default is to encode)",
	}
	GzipFlag = cli.BoolFlag{
		Name:  "gz",
		Usage: "use gzip compression",
	}
)

func init() {
	app.Usage = "Hex encoder/decoder"
	// app.Description = "Hex encoder/decoder"
	app.Flags = append(app.Flags, []cli.Flag{InputFlag, OutputFlag, DecodeFlag, GzipFlag}...)
	app.Action = streamer
	app.Name = "aquahex"
	app.HelpName = "aquahex help"
	app.ArgsUsage = ""
	app.UsageText = ""
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println("fatal:", err)
		os.Exit(111)
	}
}

func streamer(ctx *cli.Context) (err error) {
	var (
		input  io.Reader
		output io.Writer
	)
	input = os.Stdin
	if ctx.IsSet("i") {
		input, err = os.Open(ctx.String("i"))
		if err != nil {
			return err
		}
	}

	output = os.Stdout
	if ctx.IsSet("o") {
		filename := ctx.String("o")
		fmt.Println("opening", filename)
		output, err = os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
	}

	if ctx.Bool("d") {
		if ctx.Bool("gz") {
			if input, err = wrapGunzip(input); err != nil {
				return err
			}
		}
		return streamerDecode(ctx, input, output)
	}

	if ctx.Bool("gz") {
		if output, err = wrapGzip(output); err != nil {
			return err
		}
	}
	return streamerEncode(ctx, input, output)
}

func streamerEncode(ctx *cli.Context, input io.Reader, output io.Writer) (err error) {
	encoder := hex.NewEncoder(output)
	_, err = io.Copy(encoder, input)
	return err
}
func streamerDecode(ctx *cli.Context, input io.Reader, output io.Writer) (err error) {
	decoder := hex.NewDecoder(input)
	_, err = io.Copy(output, decoder)
	return err
}

func wrapGzip(out io.Writer) (output io.Writer, err error) {
	output, err = gzip.NewWriterLevel(out, gzip.BestCompression)
	return
}
func wrapGunzip(in io.Reader) (input io.Reader, err error) {
	input, err = gzip.NewReader(in)
	return
}
