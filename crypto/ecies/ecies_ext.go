package ecies

import "github.com/btcsuite/btcd/btcec/v2"

func PrivateKeyFromBtcec(prv *btcec.PrivateKey) *PrivateKey {
	ecds := prv.ToECDSA()
	pub := ImportECDSAPublic(&ecds.PublicKey)
	return &PrivateKey{*pub, ecds.D}
}
