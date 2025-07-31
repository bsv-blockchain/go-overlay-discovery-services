package utils

import (
	"errors"
	"testing"
)

// TestWallet provides a test implementation of WalletInterface
type TestWallet struct {
	ShouldVerifyPass bool
	VerifyError      error
	ExpectedPubKey   string
	PubKeyError      error
}

func (w *TestWallet) VerifySignature(req SignatureVerificationRequest) SignatureVerificationResult {
	if w.VerifyError != nil {
		return SignatureVerificationResult{Valid: false, Error: w.VerifyError}
	}
	return SignatureVerificationResult{Valid: w.ShouldVerifyPass, Error: nil}
}

func (w *TestWallet) GetPublicKey(req PublicKeyRequest) PublicKeyResult {
	if w.PubKeyError != nil {
		return PublicKeyResult{PublicKey: "", Error: w.PubKeyError}
	}
	return PublicKeyResult{PublicKey: w.ExpectedPubKey, Error: nil}
}

func TestIsTokenSignatureCorrectlyLinked(t *testing.T) {
	// Test data setup
	validLockingPubKey := "03abc123def456"
	protocol := []byte("SHIP")
	identityKey := []byte{0x01, 0x02, 0x03, 0x04}
	extraData := []byte("extra")
	signature := []byte{0xff, 0xee, 0xdd}

	validFields := TokenFields{
		protocol,
		identityKey,
		extraData,
		signature,
	}

	tests := []struct {
		name           string
		lockingPubKey  string
		fields         TokenFields
		wallet         WalletInterface
		expectedResult bool
		expectedError  bool
		errorSubstring string
	}{
		{
			name:          "valid SHIP token",
			lockingPubKey: validLockingPubKey,
			fields:        validFields,
			wallet: &TestWallet{
				ShouldVerifyPass: true,
				ExpectedPubKey:   validLockingPubKey,
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:          "valid SLAP token",
			lockingPubKey: validLockingPubKey,
			fields: TokenFields{
				[]byte("SLAP"),
				identityKey,
				extraData,
				signature,
			},
			wallet: &TestWallet{
				ShouldVerifyPass: true,
				ExpectedPubKey:   validLockingPubKey,
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:          "insufficient fields",
			lockingPubKey: validLockingPubKey,
			fields: TokenFields{
				protocol,
				identityKey,
			},
			wallet:         &TestWallet{},
			expectedResult: false,
			expectedError:  true,
			errorSubstring: "insufficient fields",
		},
		{
			name:          "unknown protocol",
			lockingPubKey: validLockingPubKey,
			fields: TokenFields{
				[]byte("UNKNOWN"),
				identityKey,
				extraData,
				signature,
			},
			wallet:         &TestWallet{},
			expectedResult: false,
			expectedError:  true,
			errorSubstring: "unknown protocol",
		},
		{
			name:          "missing identity key field",
			lockingPubKey: validLockingPubKey,
			fields: TokenFields{
				protocol,
				signature,
			},
			wallet:         &TestWallet{},
			expectedResult: false,
			expectedError:  true,
			errorSubstring: "insufficient fields",
		},
		{
			name:          "signature verification error",
			lockingPubKey: validLockingPubKey,
			fields:        validFields,
			wallet: &TestWallet{
				VerifyError: errors.New("crypto error"),
			},
			expectedResult: false,
			expectedError:  true,
			errorSubstring: "signature verification failed",
		},
		{
			name:          "invalid signature",
			lockingPubKey: validLockingPubKey,
			fields:        validFields,
			wallet: &TestWallet{
				ShouldVerifyPass: false,
				ExpectedPubKey:   validLockingPubKey,
			},
			expectedResult: false,
			expectedError:  false,
		},
		{
			name:          "public key derivation error",
			lockingPubKey: validLockingPubKey,
			fields:        validFields,
			wallet: &TestWallet{
				ShouldVerifyPass: true,
				PubKeyError:      errors.New("derivation error"),
			},
			expectedResult: false,
			expectedError:  true,
			errorSubstring: "failed to get expected public key",
		},
		{
			name:          "public key mismatch",
			lockingPubKey: validLockingPubKey,
			fields:        validFields,
			wallet: &TestWallet{
				ShouldVerifyPass: true,
				ExpectedPubKey:   "different_key",
			},
			expectedResult: false,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := IsTokenSignatureCorrectlyLinked(tt.lockingPubKey, tt.fields, tt.wallet)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorSubstring != "" && !containsSubstring(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error to contain %q, but got: %v", tt.errorSubstring, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if result != tt.expectedResult {
				t.Errorf("Expected result %v, but got %v", tt.expectedResult, result)
			}
		})
	}
}

func TestFlattenFields(t *testing.T) {
	tests := []struct {
		name     string
		fields   TokenFields
		expected []byte
	}{
		{
			name:     "empty fields",
			fields:   TokenFields{},
			expected: []byte{},
		},
		{
			name: "single field",
			fields: TokenFields{
				[]byte("hello"),
			},
			expected: []byte("hello"),
		},
		{
			name: "multiple fields",
			fields: TokenFields{
				[]byte("hello"),
				[]byte("world"),
				[]byte{0x01, 0x02},
			},
			expected: []byte("helloworld\x01\x02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := flattenFields(tt.fields)
			if !bytesEqual(result, tt.expected) {
				t.Errorf("flattenFields() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestUTFBytesToString(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"empty", []byte{}, ""},
		{"ascii", []byte("hello"), "hello"},
		{"utf8", []byte("hello 世界"), "hello 世界"},
		{"binary", []byte{0x01, 0x02, 0x03}, "\x01\x02\x03"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UTFBytesToString(tt.data)
			if result != tt.expected {
				t.Errorf("UTFBytesToString(%v) = %q, expected %q", tt.data, result, tt.expected)
			}
		})
	}
}

func TestBytesToHex(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"empty", []byte{}, ""},
		{"single byte", []byte{0xff}, "ff"},
		{"multiple bytes", []byte{0x01, 0x23, 0xab, 0xcd}, "0123abcd"},
		{"zero bytes", []byte{0x00, 0x00}, "0000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BytesToHex(tt.data)
			if result != tt.expected {
				t.Errorf("BytesToHex(%v) = %q, expected %q", tt.data, result, tt.expected)
			}
		})
	}
}

func TestHexToBytes(t *testing.T) {
	tests := []struct {
		name        string
		hexStr      string
		expected    []byte
		expectError bool
	}{
		{"empty", "", []byte{}, false},
		{"single byte", "ff", []byte{0xff}, false},
		{"multiple bytes", "0123abcd", []byte{0x01, 0x23, 0xab, 0xcd}, false},
		{"uppercase", "ABCD", []byte{0xab, 0xcd}, false},
		{"mixed case", "aBcD", []byte{0xab, 0xcd}, false},
		{"invalid character", "xyz", nil, true},
		{"odd length", "abc", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := HexToBytes(tt.hexStr)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !bytesEqual(result, tt.expected) {
					t.Errorf("HexToBytes(%q) = %v, expected %v", tt.hexStr, result, tt.expected)
				}
			}
		})
	}
}

func TestMockWallet(t *testing.T) {
	wallet := &MockWallet{}

	// Test VerifySignature
	verifyResult := wallet.VerifySignature(SignatureVerificationRequest{})
	if verifyResult.Valid {
		t.Error("MockWallet should return invalid signature")
	}
	if verifyResult.Error == nil {
		t.Error("MockWallet should return an error for signature verification")
	}

	// Test GetPublicKey
	pubKeyResult := wallet.GetPublicKey(PublicKeyRequest{})
	if pubKeyResult.PublicKey != "" {
		t.Error("MockWallet should return empty public key")
	}
	if pubKeyResult.Error == nil {
		t.Error("MockWallet should return an error for public key derivation")
	}
}

func TestProtocolIDCreation(t *testing.T) {
	// Test SHIP protocol
	shipProtocol := ProtocolID{Type: 2, Description: "service host interconnect"}
	if shipProtocol.Type != 2 {
		t.Errorf("Expected Type 2, got %d", shipProtocol.Type)
	}
	if shipProtocol.Description != "service host interconnect" {
		t.Errorf("Expected 'service host interconnect', got %q", shipProtocol.Description)
	}

	// Test SLAP protocol
	slapProtocol := ProtocolID{Type: 2, Description: "service lookup availability"}
	if slapProtocol.Type != 2 {
		t.Errorf("Expected Type 2, got %d", slapProtocol.Type)
	}
	if slapProtocol.Description != "service lookup availability" {
		t.Errorf("Expected 'service lookup availability', got %q", slapProtocol.Description)
	}
}

// Helper functions for tests

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr) >= 0
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
