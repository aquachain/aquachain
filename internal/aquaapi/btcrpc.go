package aquaapi

type PublicBitcoinAPI struct {
	a *PublicBlockChainAPI
}

func NewPublicBitcoinAPI(a *PublicBlockChainAPI) *PublicBitcoinAPI {
	return &PublicBitcoinAPI{a: a}
}

// START BTC-JSON METHODS

func (p *PublicBitcoinAPI) Getblockcount() int64 {
	return p.a.BlockNumber().Int64()
}
