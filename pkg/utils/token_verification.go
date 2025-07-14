package utils

import (
	"context"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

// IsTokenSignatureCorrectlyLinked checks that the BRC-48 locking key and the signature 
// are valid and linked to the claimed identity key.
func IsTokenSignatureCorrectlyLinked(lockingPublicKey *ec.PublicKey, fields [][]byte) bool {
	if len(fields) < 3 {
		return false
	}

	// Make a copy of fields to avoid modifying the original
	fieldsCopy := make([][]byte, len(fields))
	copy(fieldsCopy, fields)

	// The signature is the last field, which needs to be removed for verification
	signatureBytes := fieldsCopy[len(fieldsCopy)-1]
	fieldsCopy = fieldsCopy[:len(fieldsCopy)-1]

	// Parse the signature
	sig, err := ec.ParseSignature(signatureBytes)
	if err != nil {
		return false
	}

	// The protocol is in the first field
	protocolStr := string(fieldsCopy[0])
	var protocol wallet.Protocol
	switch protocolStr {
	case "SHIP":
		protocol = wallet.Protocol{
			SecurityLevel: wallet.SecurityLevelEveryApp,
			Protocol:     "service host interconnect",
		}
	case "SLAP":
		protocol = wallet.Protocol{
			SecurityLevel: wallet.SecurityLevelEveryApp,
			Protocol:     "service lookup availability",
		}
	default:
		return false
	}

	// The identity key is in the second field (needs to be parsed as public key)
	identityKeyBytes := fieldsCopy[1]
	identityPubKey, err := ec.ParsePubKey(identityKeyBytes)
	if err != nil {
		return false
	}

	// Concatenate all fields (except signature) into data
	var data []byte
	for _, field := range fieldsCopy {
		data = append(data, field...)
	}

	// Create an "anyone" wallet
	anyonePrivKey, _ := wallet.AnyoneKey()
	anyoneWallet, err := wallet.NewWallet(anyonePrivKey)
	if err != nil {
		return false
	}

	// Verify the signature
	ctx := context.Background()
	verifyArgs := wallet.VerifySignatureArgs{
		EncryptionArgs: wallet.EncryptionArgs{
			ProtocolID: protocol,
			KeyID:      "1",
			Counterparty: wallet.Counterparty{
				Type:         wallet.CounterpartyTypeOther,
				Counterparty: identityPubKey,
			},
		},
		Data:      data,
		Signature: sig,
	}

	verifyResult, err := anyoneWallet.VerifySignature(ctx, verifyArgs, "")
	if err != nil || !verifyResult.Valid {
		return false
	}

	// Get the expected locking public key
	pubKeyArgs := wallet.GetPublicKeyArgs{
		EncryptionArgs: wallet.EncryptionArgs{
			ProtocolID: protocol,
			KeyID:      "1",
			Counterparty: wallet.Counterparty{
				Type:         wallet.CounterpartyTypeOther,
				Counterparty: identityPubKey,
			},
		},
		ForSelf: false,
	}

	pubKeyResult, err := anyoneWallet.GetPublicKey(ctx, pubKeyArgs, "")
	if err != nil {
		return false
	}

	// Compare the locking public key with the expected one
	// Both public keys should match
	return pubKeyResult.PublicKey.IsEqual(lockingPublicKey)
}