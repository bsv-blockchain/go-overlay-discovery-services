package utils

import (
	"bytes"
	"context"
	"testing"

	"github.com/bsv-blockchain/go-sdk/overlay"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/require"
)

// signedTokenFixture creates a signed token with the given protocol and returns
// the fields (with signature appended) and the locking public key hex.
func signedTokenFixture(t *testing.T, ctx context.Context, signerWallet *wallet.Wallet, protocol string, protocolID overlay.ProtocolID, identityDER []byte) (TokenFields, string) {
	t.Helper()
	fields := make(TokenFields, 0, 5)
	fields = append(fields, []byte(protocol), identityDER, []byte("https://domain.com"), []byte("tm_meter"))
	data := flattenFields(fields)

	sigResult, err := signerWallet.CreateSignature(ctx, wallet.CreateSignatureArgs{
		EncryptionArgs: wallet.EncryptionArgs{
			ProtocolID: wallet.Protocol{
				SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      string(protocolID),
			},
			KeyID:        "1",
			Counterparty: wallet.Counterparty{Type: wallet.CounterpartyTypeAnyone},
		},
		Data: data,
	}, "")
	require.NoError(t, err)

	sigDER, err := sigResult.Signature.ToDER()
	require.NoError(t, err)
	fields = append(fields, sigDER)

	forSelfTrue := true
	pubKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
		EncryptionArgs: wallet.EncryptionArgs{
			ProtocolID: wallet.Protocol{
				SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty,
				Protocol:      string(protocolID),
			},
			KeyID:        "1",
			Counterparty: wallet.Counterparty{Type: wallet.CounterpartyTypeAnyone},
		},
		ForSelf: &forSelfTrue,
	}, "")
	require.NoError(t, err)

	return fields, pubKeyResult.PublicKey.ToDERHex()
}

// newTestWallet creates a new wallet and returns it along with its identity key DER bytes.
func newTestWallet(t *testing.T, ctx context.Context) (*wallet.Wallet, []byte) {
	t.Helper()
	key, err := ec.NewPrivateKey()
	require.NoError(t, err)
	w, err := wallet.NewWallet(key)
	require.NoError(t, err)
	idResult, err := w.GetPublicKey(ctx, wallet.GetPublicKeyArgs{IdentityKey: true}, "")
	require.NoError(t, err)
	return w, idResult.PublicKey.ToDER()
}

func TestIsTokenSignatureCorrectlyLinked(t *testing.T) {
	ctx := context.Background()

	t.Run("validates a correctly-linked SHIP signature", func(t *testing.T) {
		signerWallet, identityDER := newTestWallet(t, ctx)
		fields, pubKeyHex := signedTokenFixture(t, ctx, signerWallet, "SHIP", overlay.ProtocolIDSHIP, identityDER)

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, pubKeyHex, fields)
		require.NoError(t, err)
		require.True(t, valid)
	})

	t.Run("validates a correctly-linked SLAP signature", func(t *testing.T) {
		signerWallet, identityDER := newTestWallet(t, ctx)
		fields, pubKeyHex := signedTokenFixture(t, ctx, signerWallet, "SLAP", overlay.ProtocolIDSLAP, identityDER)

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, pubKeyHex, fields)
		require.NoError(t, err)
		require.True(t, valid)
	})

	t.Run("fails to validate signature over tampered data", func(t *testing.T) {
		signerWallet, identityDER := newTestWallet(t, ctx)
		fields, pubKeyHex := signedTokenFixture(t, ctx, signerWallet, "SHIP", overlay.ProtocolIDSHIP, identityDER)

		// Tamper with the protocol field after signing
		fields[0] = []byte("SLAP")

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, pubKeyHex, fields)
		require.NoError(t, err)
		require.False(t, valid)
	})

	t.Run("fails if claimed identity key is incorrect", func(t *testing.T) {
		signerWallet, _ := newTestWallet(t, ctx)
		_, imposterIdentityDER := newTestWallet(t, ctx)

		// Sign with signer's key but claim imposter's identity
		fields, pubKeyHex := signedTokenFixture(t, ctx, signerWallet, "SHIP", overlay.ProtocolIDSHIP, imposterIdentityDER)

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, pubKeyHex, fields)
		require.NoError(t, err)
		require.False(t, valid)
	})

	t.Run("fails with insufficient fields", func(t *testing.T) {
		fields := TokenFields{
			[]byte("SHIP"),
			[]byte("insufficient"),
		}

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, "any", fields)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient fields")
		require.False(t, valid)
	})

	t.Run("fails with unknown protocol", func(t *testing.T) {
		fields := TokenFields{
			[]byte("UNKNOWN"),
			[]byte{
				0x02, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
			},
			[]byte("data"),
			[]byte{
				0x30, 0x44, 0x02, 0x20, // Basic DER signature structure
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x02, 0x20,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
				0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
			},
		}

		valid, err := IsTokenSignatureCorrectlyLinked(ctx, "any", fields)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown protocol")
		require.False(t, valid)
	})
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
			if !bytes.Equal(result, tt.expected) {
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
		//nolint:gosmopolitan // Test case requires specific UTF-8 characters including Chinese
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
				if !bytes.Equal(result, tt.expected) {
					t.Errorf("HexToBytes(%q) = %v, expected %v", tt.hexStr, result, tt.expected)
				}
			}
		})
	}
}
