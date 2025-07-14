package utils

import (
	"context"
	"testing"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTokenSignatureCorrectlyLinked(t *testing.T) {
	t.Run("Validates a correctly-linked signature", func(t *testing.T) {
		// Create signer with seed 42 (matching TypeScript)
		// We need a 32-byte seed, so we'll create one with value 42
		seed := make([]byte, 32)
		seed[0] = 42
		signerKey, _ := ec.PrivateKeyFromBytes(seed)
		
		signerWallet, err := wallet.NewWallet(signerKey)
		require.NoError(t, err)

		// Get identity key
		identityKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{},
			IdentityKey:    true,
		}, "")
		require.NoError(t, err)
		
		// Create fields
		fields := [][]byte{
			[]byte("SHIP"),
			identityKeyResult.PublicKey.Compressed(),
			[]byte("https://domain.com"),
			[]byte("tm_meter"),
		}
		
		// Create data by concatenating fields
		var data []byte
		for _, field := range fields {
			data = append(data, field...)
		}
		
		// Create signature
		signResult, err := signerWallet.CreateSignature(ctx, wallet.CreateSignatureArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			Data: data,
		}, "")
		require.NoError(t, err)
		
		// Add signature to fields
		fields = append(fields, signResult.Signature.Serialize())
		
		// Get the locking public key
		lockingKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			ForSelf: true,
		}, "")
		require.NoError(t, err)
		
		// Validate
		valid := IsTokenSignatureCorrectlyLinked(lockingKeyResult.PublicKey, fields)
		assert.True(t, valid)
	})

	t.Run("Fails to validate a signature over data that is simply incorrect", func(t *testing.T) {
		// Create signer with seed 42 (matching TypeScript)
		// We need a 32-byte seed, so we'll create one with value 42
		seed := make([]byte, 32)
		seed[0] = 42
		signerKey, _ := ec.PrivateKeyFromBytes(seed)
		
		signerWallet, err := wallet.NewWallet(signerKey)
		require.NoError(t, err)

		// Get identity key
		identityKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{},
			IdentityKey:    true,
		}, "")
		require.NoError(t, err)
		
		// Create fields
		fields := [][]byte{
			[]byte("SHIP"),
			identityKeyResult.PublicKey.Compressed(),
			[]byte("https://domain.com"),
			[]byte("tm_meter"),
		}
		
		// Create data by concatenating fields
		var data []byte
		for _, field := range fields {
			data = append(data, field...)
		}
		
		// Create signature
		signResult, err := signerWallet.CreateSignature(ctx, wallet.CreateSignatureArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			Data: data,
		}, "")
		require.NoError(t, err)
		
		// Add signature to fields
		fields = append(fields, signResult.Signature.Serialize())
		
		// Get the locking public key
		lockingKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			ForSelf: true,
		}, "")
		require.NoError(t, err)
		
		// Tamper with fields - change SHIP to SLAP
		fields[0] = []byte("SLAP")
		
		// Validate - should fail because data was tampered
		valid := IsTokenSignatureCorrectlyLinked(lockingKeyResult.PublicKey, fields)
		assert.False(t, valid)
	})

	t.Run("Even if the signature is facially correct, fails if the claimed identity key is incorrect", func(t *testing.T) {
		// Create signer with seed 42
		seed := make([]byte, 32)
		seed[0] = 42
		signerKey, _ := ec.PrivateKeyFromBytes(seed)
		
		signerWallet, err := wallet.NewWallet(signerKey)
		require.NoError(t, err)

		// Create Taylor Swift with seed 69 (matching TypeScript comment)
		taylorSeed := make([]byte, 32)
		taylorSeed[0] = 69
		taylorSwiftKey, _ := ec.PrivateKeyFromBytes(taylorSeed)
		
		taylorSwiftWallet, err := wallet.NewWallet(taylorSwiftKey)
		require.NoError(t, err)

		// Get Taylor Swift's identity key
		taylorSwiftIdentityResult, err := taylorSwiftWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{},
			IdentityKey:    true,
		}, "")
		require.NoError(t, err)
		
		// Create fields claiming to be Taylor Swift
		fields := [][]byte{
			[]byte("SHIP"),
			taylorSwiftIdentityResult.PublicKey.Compressed(), // Claiming to be Taylor Swift
			[]byte("https://domain.com"),
			[]byte("tm_meter"),
		}
		
		// Create data by concatenating fields
		var data []byte
		for _, field := range fields {
			data = append(data, field...)
		}
		
		// Create signature with signer's key (not Taylor Swift's)
		signResult, err := signerWallet.CreateSignature(ctx, wallet.CreateSignatureArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			Data: data,
		}, "")
		require.NoError(t, err)
		
		// Add signature to fields
		fields = append(fields, signResult.Signature.Serialize())
		
		// Get the locking public key from signer (not Taylor Swift)
		lockingKeyResult, err := signerWallet.GetPublicKey(ctx, wallet.GetPublicKeyArgs{
			EncryptionArgs: wallet.EncryptionArgs{
				ProtocolID: wallet.Protocol{
					SecurityLevel: wallet.SecurityLevelEveryApp,
					Protocol:     "service host interconnect",
				},
				KeyID: "1",
				Counterparty: wallet.Counterparty{
					Type: wallet.CounterpartyTypeAnyone,
				},
			},
			ForSelf: true,
		}, "")
		require.NoError(t, err)
		
		// Validate - should fail because they're pretending to be someone they're not
		valid := IsTokenSignatureCorrectlyLinked(lockingKeyResult.PublicKey, fields)
		assert.False(t, valid)
	})
}

var ctx = context.Background()