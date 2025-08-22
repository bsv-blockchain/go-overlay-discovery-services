package utils

import (
	"encoding/hex"
	"errors"
	"fmt"
)

// ProtocolID represents the protocol identifier used in BRC-48 token validation
type ProtocolID struct {
	Type        int    // Protocol type (e.g., 2 for service protocols)
	Description string // Protocol description
}

// TokenFields represents the fields of a PushDrop token for SHIP or SLAP advertisement
type TokenFields [][]byte

// SignatureVerificationRequest represents the data needed to verify a signature
type SignatureVerificationRequest struct {
	Data         []byte
	Signature    []byte
	Counterparty string // Identity key in hex format
	ProtocolID   ProtocolID
	KeyID        string
}

// SignatureVerificationResult represents the result of signature verification
type SignatureVerificationResult struct {
	Valid bool
	Error error
}

// PublicKeyRequest represents a request to get a public key
type PublicKeyRequest struct {
	Counterparty string // Identity key in hex format
	ProtocolID   ProtocolID
	KeyID        string
}

// PublicKeyResult represents the result of getting a public key
type PublicKeyResult struct {
	PublicKey string
	Error     error
}

// WalletInterface defines the interface for wallet operations needed for token validation.
// This interface should be implemented by the actual BSV SDK wallet when available.
type WalletInterface interface {
	// VerifySignature verifies a signature against the provided data and counterparty identity
	VerifySignature(req SignatureVerificationRequest) SignatureVerificationResult

	// GetPublicKey derives the expected public key for the given counterparty and protocol
	GetPublicKey(req PublicKeyRequest) PublicKeyResult
}

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
func IsTokenSignatureCorrectlyLinked(lockingPublicKey string, fields TokenFields, wallet WalletInterface) (bool, error) {
	if len(fields) < 3 {
		return false, errors.New("insufficient fields in token (need at least protocol, identity key, and signature)")
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

	var protocolID ProtocolID
	if protocolString == "SHIP" {
		protocolID = ProtocolID{Type: 2, Description: "service host interconnect"}
	} else if protocolString == "SLAP" {
		protocolID = ProtocolID{Type: 2, Description: "service lookup availability"}
	} else {
		return false, fmt.Errorf("unknown protocol: %s", protocolString)
	}

	// The identity key is in the second field
	if len(dataFields) < 2 {
		return false, errors.New("missing identity key field")
	}
	identityKeyBytes := dataFields[1]
	identityKey := hex.EncodeToString(identityKeyBytes)

	// First, we ensure that the signature over the data is valid for the claimed identity key
	data := flattenFields(dataFields)

	verifyReq := SignatureVerificationRequest{
		Data:         data,
		Signature:    signature,
		Counterparty: identityKey,
		ProtocolID:   protocolID,
		KeyID:        "1",
	}

	verifyResult := wallet.VerifySignature(verifyReq)
	if verifyResult.Error != nil {
		return false, fmt.Errorf("signature verification failed: %w", verifyResult.Error)
	}
	if !verifyResult.Valid {
		return false, nil // Invalid signature, but not a technical error
	}

	// Then, we ensure that the locking public key matches the correct derived child
	pubKeyReq := PublicKeyRequest{
		Counterparty: identityKey,
		ProtocolID:   protocolID,
		KeyID:        "1",
	}

	pubKeyResult := wallet.GetPublicKey(pubKeyReq)
	if pubKeyResult.Error != nil {
		return false, fmt.Errorf("failed to get expected public key: %w", pubKeyResult.Error)
	}

	return pubKeyResult.PublicKey == lockingPublicKey, nil
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

// MockWallet provides a mock implementation of WalletInterface for testing purposes.
// In production, this should be replaced with the actual BSV SDK wallet implementation.
type MockWallet struct{}

// VerifySignature provides a mock implementation that always returns invalid.
// Replace this with actual BSV SDK implementation.
func (w *MockWallet) VerifySignature(req SignatureVerificationRequest) SignatureVerificationResult {
	return SignatureVerificationResult{
		Valid: false,
		Error: errors.New("mock wallet: signature verification not implemented"),
	}
}

// GetPublicKey provides a mock implementation that returns an error.
// Replace this with actual BSV SDK implementation.
func (w *MockWallet) GetPublicKey(req PublicKeyRequest) PublicKeyResult {
	return PublicKeyResult{
		PublicKey: "",
		Error:     errors.New("mock wallet: public key derivation not implemented"),
	}
}
