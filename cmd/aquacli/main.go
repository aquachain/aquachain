package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/aquachain/aquachain/cmd/utils"
	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/core/types"
	"gitlab.com/aquachain/aquachain/internal/debug"
	"gitlab.com/aquachain/aquachain/opt/aquaclient"
	"gitlab.com/aquachain/aquachain/params"
	"gitlab.com/aquachain/aquachain/rlp"
	"gitlab.com/aquachain/aquachain/rpc"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	big1   = big.NewInt(1)
	Config = params.Testnet2ChainConfig
)

var gitCommit = ""

var (
	app = utils.NewApp(gitCommit, "usage")
)

func init() {
	app.Name = "aquacli"
	app.Action = runit
	app.Flags = append(debug.Flags, []cli.Flag{
		cli.StringFlag{
			Value: filepath.Join(utils.DataDirFlag.Value.String(), "testnet2/aquachain.ipc"),
			Name:  "rpc",
			Usage: "path or url to rpc",
		},
	}...)
}

//valid block #1 using -testnet2
var header1 = &types.Header{
	Difficulty: big.NewInt(4096),
	Extra:      []byte{0xd4, 0x83, 0x01, 0x07, 0x04, 0x89, 0x61, 0x71, 0x75, 0x61, 0x63, 0x68, 0x61, 0x69, 0x6e, 0x85, 0x6c, 0x69, 0x6e, 0x75, 0x78},
	GasLimit:   4704588,
	GasUsed:    0,
	// Hash: "0x73851a4d607acd8341cf415beeed9c8b8c803e1e835cb45080f6af7a2127e807",
	Coinbase:    common.HexToAddress("0xcf8e5ba37426404bef34c3ca4fa2d2ed9be41e58"),
	MixDigest:   common.Hash{},
	Nonce:       types.BlockNonce{0x70, 0xc2, 0xdd, 0x45, 0xa3, 0x10, 0x17, 0x35},
	Number:      big.NewInt(1),
	ParentHash:  common.HexToHash("0xde434983d3ada19cd43c44d8ad5511bad01ed12b3cc9a99b1717449a245120df"),
	ReceiptHash: common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	UncleHash:   common.HexToHash("0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347"),
	Root:        common.HexToHash("0x194b1927f77b77161b58fed1184990d8f7b345fabf8ef8706ee865a844f73bc3"),
	Time:        big.NewInt(1536181711),
	TxHash:      common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421"),
	Version:     2,
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Println("fatal:", err)
	}
}

func runit(ctx *cli.Context) error {
	rpcclient, err := getclient(ctx)
	if err != nil {
		return err
	}
	aqua := aquaclient.NewClient(rpcclient)
	parent, err := aqua.BlockByNumber(context.Background(), nil)
	if err != nil {
		return err
	}
	_, _ = aqua.GetWork(context.Background())
	parent.SetVersion(Config.GetBlockVersion(parent.Number()))
	/* tstart := time.Now()
	tstamp := tstart.Unix()
	num := parent.Number()
	numnew := num.Add(num, common.Big1)
	 hdr := &types.Header{
		ParentHash: parent.Hash(),
		Number:     numnew,
		GasLimit:   core.CalcGasLimit(parent),
		Extra:      []byte("aqua"),
		Time:       big.NewInt(tstamp),
		Version:    Config.GetBlockVersion(numnew),
	} */
	fmt.Println(parent, header1)
	block1 := types.NewBlock(header1, nil, nil, nil)
	encoded, err := rlp.EncodeToBytes(&block1)
	if err != nil {
		return err
	}
	if !aqua.SubmitBlock(context.Background(), encoded) {
		fmt.Println("failed")
		return fmt.Errorf("failed")
	} else {
		fmt.Println("success", block1)
	}
	return nil

}

func getclient(ctx *cli.Context) (*rpc.Client, error) {
	if strings.HasPrefix(ctx.String("rpc"), "http") {
		return rpc.DialHTTP(ctx.String("rpc"))
	} else {
		return rpc.DialIPC(context.Background(), ctx.String("rpc"))
	}
}
