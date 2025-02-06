package crypto_test

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"log"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"gitlab.com/aquachain/aquachain/crypto"
	"gitlab.com/aquachain/aquachain/p2p/discover"
)

func testGenerateKey(t *testing.T, limit int) error {
	t.Logf("hi")
	for i := 0; i < limit; i++ {
		pk, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		pub := pk.PubKey()
		ecdsa_pub := pub.ToECDSA()

		err = checkKeyPair(pk, pub, ecdsa_pub)
		if err != nil {
			log.Printf("Public key: %02x (compressed)", pub.SerializeCompressed()) // 33 bytes
			log.Printf("Public key: %02x", crypto.FromECDSAPub(ecdsa_pub))         // 65 bytes
			log.Printf("Public key: %02x", pub.SerializeUncompressed())            // 65 bytes
			log.Printf("Public key: %02x", crypto.FromECDSAPub(pub.ToECDSA()))     // 65 bytes

			var discoID discover.NodeID
			copy(discoID[:], crypto.FromECDSAPub(ecdsa_pub)[1:])
			log.Printf("Node ID: %s", discoID.String())
			log.Printf("Wallet:  %s", crypto.PubkeyToAddress(*pub).Hex())
			{
				if pk == nil {
					t.Fatal("wtf")
				}
				log.Printf("PrivateKey1:   \"%02x\",", crypto.Ecdsa2Btcec(pk.ToECDSA()))
				log.Printf("PrivateKey2:   \"%02x\",", pk.Serialize())
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func checkKeyPair(pk *btcec.PrivateKey, pub *btcec.PublicKey, ecdsa_pub *ecdsa.PublicKey) error {
	if !pub.IsEqual(pk.PubKey()) {
		return fmt.Errorf("public key mismatch 1")
	}
	if !bytes.Equal(crypto.FromECDSAPub(ecdsa_pub), pub.SerializeUncompressed()) {
		return fmt.Errorf("public key mismatch 2")
	}

	pke := pk.ToECDSA()
	if pke.X.Cmp(ecdsa_pub.X) != 0 || pke.Y.Cmp(ecdsa_pub.Y) != 0 {
		return fmt.Errorf("public key mismatch 3")
	}

	// from btcec.PrivateKey to ecdsa.PrivateKey and back
	ecdsa_pk := pk.ToECDSA()
	pk2 := crypto.Ecdsa2Btcec(ecdsa_pk)
	if !pk.Key.Equals(&pk2.Key) {
		return fmt.Errorf("private key mismatch")
	}
	if !pub.IsEqual(pk2.PubKey()) {
		return fmt.Errorf("public key mismatch 4")
	}

	return nil
}

var ErrPubKey = fmt.Errorf("public key mismatch")

func TestPrintKeys(t *testing.T) {
	if err := testGenerateKey(t, 10000); err != nil {
		t.Fatal(err)
	}
}
