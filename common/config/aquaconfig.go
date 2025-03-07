// config package for all other packages to enjoy
//
// preferably it should only import hexutil.
// (this package and the branch using it is a work in progress)
package config

import (
	"time"

	"gitlab.com/aquachain/aquachain/common"
	"gitlab.com/aquachain/aquachain/common/alerts"
	"gitlab.com/aquachain/aquachain/common/hexutil"
	"gitlab.com/aquachain/aquachain/consensus/aquahash"

	// these imports should be reversed, and instead import this package
	"gitlab.com/aquachain/aquachain/aqua/downloader" // TODO remove
	"gitlab.com/aquachain/aquachain/aqua/gasprice"

	// TODO remove
	// TODO remove
	"gitlab.com/aquachain/aquachain/core" // TODO remove
	"gitlab.com/aquachain/aquachain/node" // TODO remove
	"gitlab.com/aquachain/aquachain/p2p"  // TODO remove
)

type Nodeconfig = node.Config // TODO remove
type P2pconfig = p2p.Config   // TODO remove

type AquahashConfig = aquahash.Config

type GaspriceConfig = gasprice.Config

type TxPoolConfig = core.TxPoolConfig

type EthstatsConfig struct {
	URL string `toml:",omitempty"`
}

type AquachainConfigFull struct {
	Info      any            `toml:",omitempty"` // this is so config file can have a comment at the top and still parse
	Aqua      *Aquaconfig    // aquachain config
	Node      *Nodeconfig    // p2p node config
	Aquastats EthstatsConfig `toml:",omitempty"`
	p2P       *P2pconfig     `toml:",omitempty"` // same pointer as Node.P2P
}

// Copy doesnt copy everything
func (a *AquachainConfigFull) Copy() *AquachainConfigFull {
	if a == nil {
		return nil
	}
	var c = *a

	if a.Aqua != nil {
		c.Aqua = new(Aquaconfig)
		*c.Aqua = *a.Aqua
	}

	if a.Node != nil {
		c.Node = new(Nodeconfig)
		*c.Node = *a.Node
		if a.Node.P2P != nil {
			c.Node.P2P = new(P2pconfig)
			*c.Node.P2P = *a.Node.P2P
			c.p2P = c.Node.P2P // same new pointer as Node.P2P
		}
	}

	if a.p2P != nil {
		c.p2P = new(P2pconfig)
		*c.p2P = *a.p2P
	}

	if c.Aqua != nil {
		c.Aqua.Genesis = a.Aqua.Genesis // same pointer
		if a.Aqua.ExtraData != nil {
			c.Aqua.ExtraData = make(hexutil.Bytes, len(a.Aqua.ExtraData))
			copy(c.Aqua.ExtraData, a.Aqua.ExtraData)
		}
		c.Aqua.p2pnodename = a.Aqua.p2pnodename
	}

	if c.Node != nil {
		c.Node.P2P = a.Node.P2P // same pointer
		c.Node.P2P.ChainId = a.Node.P2P.ChainId
	}
	return &c
}

type ctxval string

var (
	CtxDoitNow ctxval = "doitnow"
)

//go:generate gencodec -type Aquaconfig -field-override AquaConfigMarshaling -formats toml -out gen_config.go

type Aquaconfig struct {
	// The genesis block, which is inserted if the database is empty.
	// If nil, the Aquachain main net block is used.
	Genesis *core.Genesis `toml:",omitempty"`

	// Protocol options
	ChainId   uint64 // Network ID to use for selecting peers to connect to
	SyncMode  downloader.SyncMode
	NoPruning bool `toml:"NoPruning"`

	// Database options
	SkipBcVersionCheck bool `toml:"-"`
	DatabaseHandles    int  `toml:"-"`
	DatabaseCache      int
	TrieCache          int
	TrieTimeout        time.Duration

	// Mining-related options
	Aquabase     common.Address `toml:",omitempty"`
	MinerThreads int            `toml:",omitempty"`
	ExtraData    hexutil.Bytes  `toml:",omitempty"`
	GasPrice     uint64         // TODO use uint64 since it wont go above 1e18 anyways

	// Aquahash options
	Aquahash *AquahashConfig

	// Transaction pool options
	TxPool TxPoolConfig

	// Gas Price Oracle options
	GPO GaspriceConfig

	// Enables tracking of SHA3 preimages in the VM
	EnablePreimageRecording bool

	// Miscellaneous options

	JavascriptDirectory string `toml:"-"` // for console/attach only

	// Alert options
	Alerts      alerts.AlertConfig `toml:",omitempty"`
	p2pnodename string             `toml:"-"`
}

func (a *Aquaconfig) GetNodeName() string {
	return a.p2pnodename
}
func (a *Aquaconfig) SetNodeName(name string) {
	a.p2pnodename = name
}

// AquaConfigMarshaling must be changed if the Config struct changes.
type AquaConfigMarshaling struct {
	ExtraData hexutil.Bytes
}
