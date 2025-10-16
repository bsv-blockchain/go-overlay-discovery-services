package utils

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/overlay"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
)

// Static error variables for err113 compliance
var (
	errInsufficientTokenFields = errors.New("insufficient fields in token (need at least protocol, identity key, and signature)")
	errUnknownProtocol         = errors.New("unknown protocol")
	errMissingIdentityKeyField = errors.New("missing identity key field")
)

// TokenFields represents the fields of a PushDrop token for SHIP or SLAP advertisement
type TokenFields [][]byte

// IsTokenSignatureCorrectlyLinked checks that the BRC-48 locking key and the signature
// are valid and linked to the claimed identity key.
//
// This function validates:
// 1. The signature over the token data is valid for the claimed identity key
// 2. The locking public key matches the correct derived child key
//
// Parameters:
//   - lockingPublicKey: The public key used in the output's locking script (hex string)
//   - fields: The fields of the PushDrop token for the SHIP or SLAP advertisement
//   - wallet: Implementation of WalletInterface for cryptographic operations
//
// Returns:
//   - bool: true if the token's signature is properly linked to the claimed identity key
//   - error: error if validation fails due to technical issues (nil for invalid signatures)
func IsTokenSignatureCorrectlyLinked(ctx context.Context, lockingPublicKey string, fields TokenFields) (bool, error) {
	if len(fields) < 3 {
		return false, errInsufficientTokenFields
	}

	// Make a copy to avoid mutating the original
	fieldsCopy := make(TokenFields, len(fields))
	copy(fieldsCopy, fields)

	// The signature is the last field, which needs to be removed for verification
	signature := fieldsCopy[len(fieldsCopy)-1]
	dataFields := fieldsCopy[:len(fieldsCopy)-1]

	// The protocol is in the first field
	protocolBytes := dataFields[0]
	protocolString := string(protocolBytes)

	protocolID := string(overlay.Protocol(protocolString).ID())
	if protocolID == "" {
		return false, fmt.Errorf("%w: %s", errUnknownProtocol, protocolString)
	}

	// The identity key is in the second field
	if len(dataFields) < 2 {
		return false, errMissingIdentityKeyField
	}
	identityKeyBytes := dataFields[1]
	identityKey, err := ec.PublicKeyFromBytes(identityKeyBytes)
	if err != nil {
		return false, fmt.Errorf("invalid identity key: %w", err)
	}

	// First, we ensure that the signature over the data is valid for the claimed identity key
	data := flattenFields(dataFields)

	// Convert signature bytes to ec.Signature
	sig, err := ec.FromDER(signature)
	if err != nil {
		return false, fmt.Errorf("failed to parse signature: %w", err)
	}

	encryptionArgs := wallet.EncryptionArgs{
		Counterparty: wallet.Counterparty{
			Type:         wallet.CounterpartyTypeOther,
			Counterparty: identityKey,
		},
		ProtocolID: wallet.Protocol{
			SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
			Protocol:      protocolID,
		},
		KeyID: "1",
	}

	verifyReq := wallet.VerifySignatureArgs{
		EncryptionArgs: encryptionArgs,
		Data:           data,
		Signature:      sig,
	}
	anyoneWallet, _ := wallet.NewWallet(nil)
	verifyResult, err := anyoneWallet.VerifySignature(ctx, verifyReq, "")
	if err != nil {
		return false, fmt.Errorf("signature verification failed: %w", err)
	}
	if !verifyResult.Valid {
		return false, nil // Invalid signature, but not a technical error
	}

	// Then, we ensure that the locking public key matches the correct derived child
	pubKeyReq := wallet.GetPublicKeyArgs{EncryptionArgs: encryptionArgs}
	pubKeyResult, err := anyoneWallet.GetPublicKey(ctx, pubKeyReq, "")
	if err != nil {
		return false, fmt.Errorf("failed to get expected public key: %w", err)
	}

	return pubKeyResult.PublicKey.ToDERHex() == lockingPublicKey, nil
}

// flattenFields concatenates all field bytes into a single byte slice for signature verification
func flattenFields(fields TokenFields) []byte {
	var result []byte
	for _, field := range fields {
		result = append(result, field...)
	}
	return result
}

// UTFBytesToString converts UTF-8 bytes to string
func UTFBytesToString(data []byte) string {
	return string(data)
}

// BytesToHex converts bytes to hex string
func BytesToHex(data []byte) string {
	return hex.EncodeToString(data)
}

// HexToBytes converts hex string to bytes
func HexToBytes(hexStr string) ([]byte, error) {
	return hex.DecodeString(hexStr)
}
