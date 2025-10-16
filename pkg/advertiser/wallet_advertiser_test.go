// Package advertiser contains tests for the WalletAdvertiser functionality
package advertiser

import (
	"encoding/hex"
	"testing"

	"github.com/bsv-blockchain/go-sdk/transaction"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWalletAdvertiser(t *testing.T) {
	tests := []struct {
		name            string
		chain           string
		privateKey      string
		storageURL      string
		advertisableURI string
		lookupConfig    *types.LookupResolverConfig
		expectedError   string
		shouldSucceed   bool
	}{
		{
			name:            "Valid parameters",
			chain:           "main",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "https://storage.example.com",
			advertisableURI: "https://service.example.com/",
			lookupConfig:    nil,
			shouldSucceed:   true,
		},
		{
			name:            "Valid parameters with lookup config",
			chain:           "test",
			privateKey:      "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210",
			storageURL:      "http://localhost:8080",
			advertisableURI: "https://test.example.com/",
			lookupConfig: &types.LookupResolverConfig{
				HTTPSEndpoint: stringPtr("https://resolver.example.com"),
				MaxRetries:    intPtr(3),
				TimeoutMS:     intPtr(5000),
			},
			shouldSucceed: true,
		},
		{
			name:            "Empty chain",
			chain:           "",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "https://storage.example.com",
			advertisableURI: "https://service.example.com/",
			expectedError:   "chain parameter is required and cannot be empty",
		},
		{
			name:            "Empty private key",
			chain:           "main",
			privateKey:      "",
			storageURL:      "https://storage.example.com",
			advertisableURI: "https://service.example.com/",
			expectedError:   "privateKey parameter is required and cannot be empty",
		},
		{
			name:            "Invalid private key",
			chain:           "main",
			privateKey:      "invalid-hex",
			storageURL:      "https://storage.example.com",
			advertisableURI: "https://service.example.com/",
			expectedError:   "privateKey must be a valid hexadecimal string",
		},
		{
			name:            "Empty storage URL",
			chain:           "main",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "",
			advertisableURI: "https://service.example.com/",
			expectedError:   "storageURL parameter is required and cannot be empty",
		},
		{
			name:            "Invalid storage URL",
			chain:           "main",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "ftp://invalid.com",
			advertisableURI: "https://service.example.com/",
			expectedError:   "storageURL must be a valid HTTP or HTTPS URL",
		},
		{
			name:            "Empty advertisable URI",
			chain:           "main",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "https://storage.example.com",
			advertisableURI: "",
			expectedError:   "advertisableURI parameter is required and cannot be empty",
		},
		{
			name:            "Invalid advertisable URI",
			chain:           "main",
			privateKey:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			storageURL:      "https://storage.example.com",
			advertisableURI: "invalid-uri",
			expectedError:   "advertisableURI is not valid according to BRC-101 specification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			advertiser, err := NewWalletAdvertiser(tt.chain, tt.privateKey, tt.storageURL, tt.advertisableURI, tt.lookupConfig)

			if tt.shouldSucceed {
				require.NoError(t, err)
				assert.NotNil(t, advertiser)
				assert.Equal(t, tt.chain, advertiser.GetChain())
				assert.Equal(t, tt.storageURL, advertiser.GetStorageURL())
				assert.Equal(t, tt.advertisableURI, advertiser.GetAdvertisableURI())
				assert.False(t, advertiser.IsInitialized())
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, advertiser)
			}
		})
	}
}

func TestWalletAdvertiser_Init(t *testing.T) {
	advertiser, err := NewWalletAdvertiser(
		"main",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"https://storage.example.com",
		"https://service.example.com/",
		nil,
	)
	require.NoError(t, err)

	advertiser.SetSkipStorageValidation(true) // Skip storage validation for test

	err = advertiser.Init()
	require.NoError(t, err)
	assert.True(t, advertiser.IsInitialized())

	// Test double initialization
	err = advertiser.Init()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser is already initialized")
}

func TestWalletAdvertiser_CreateAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)
	advertiser.Finder = &MockFinder{} // Use mock finder to avoid needing wallet funding

	tests := []struct {
		name          string
		adsData       []*oa.AdvertisementData
		expectedError string
		shouldFail    bool
	}{
		{
			name: "Valid SHIP advertisement",
			adsData: []*oa.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSHIP,
					TopicOrServiceName: "tm_ship",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name: "Valid SLAP advertisement",
			adsData: []*oa.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSLAP,
					TopicOrServiceName: "tm_meter",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name:          "Empty advertisements array",
			adsData:       []*oa.AdvertisementData{},
			expectedError: "at least one advertisement data entry is required",
		},
		{
			name: "Empty topic name",
			adsData: []*oa.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSHIP,
					TopicOrServiceName: "",
				},
			},
			expectedError: "topicOrServiceName cannot be empty",
		},
		{
			name: "Invalid topic name",
			adsData: []*oa.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSHIP,
					TopicOrServiceName: "Invalid-Name",
				},
			},
			expectedError: "invalid topic or service name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := advertiser.CreateAdvertisements(tt.adsData)

			if tt.shouldFail || tt.expectedError != "" {
				require.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestWalletAdvertiser_FindAllAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)
	advertiser.Finder = &MockFinder{} // Use mock finder to avoid real network calls

	tests := []struct {
		name          string
		protocol      overlay.Protocol
		expectedError string
		shouldFail    bool
	}{
		{
			name:       "Valid SHIP protocol",
			protocol:   overlay.ProtocolSHIP,
			shouldFail: false, // Implementation is now complete
		},
		{
			name:       "Valid SLAP protocol",
			protocol:   overlay.ProtocolSLAP,
			shouldFail: false, // Implementation is now complete
		},
		{
			name:          "Invalid protocol",
			protocol:      "INVALID",
			expectedError: "unsupported protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := advertiser.FindAllAdvertisements(tt.protocol)

			if tt.shouldFail || tt.expectedError != "" {
				require.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestWalletAdvertiser_RevokeAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name           string
		advertisements []*oa.Advertisement
		expectedError  string
		shouldFail     bool
	}{
		{
			name: "Valid advertisement with BEEF",
			advertisements: []*oa.Advertisement{
				{
					Protocol:       overlay.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					Beef:           []byte("BEEF\x01\x00\x00\x00\x01\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00"), // Valid minimal BEEF data
					OutputIndex:    1,
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name:           "Empty advertisements array",
			advertisements: []*oa.Advertisement{},
			expectedError:  "at least one advertisement is required for revocation",
		},
		{
			name: "Advertisement missing BEEF",
			advertisements: []*oa.Advertisement{
				{
					Protocol:       overlay.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					OutputIndex:    1,
				},
			},
			expectedError: "advertisement at index 0 is missing BEEF data required for revocation",
		},
		{
			name: "Advertisement missing output index",
			advertisements: []*oa.Advertisement{
				{
					Protocol:       overlay.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					Beef:           []byte("test-beef"),
					OutputIndex:    0, // This will trigger the validation error
				},
			},
			expectedError: "advertisement at index 0 is missing output index required for revocation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := advertiser.RevokeAdvertisements(tt.advertisements)

			if tt.shouldFail || tt.expectedError != "" {
				require.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

type MockFinder struct{}

func (m *MockFinder) Advertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error) {
	return []*oa.Advertisement{
		{
			Protocol:       protocol,
			IdentityKey:    "02abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			Domain:         "example.com",
			TopicOrService: "test_service",
			Beef:           []byte("mock-beef-data"),
			OutputIndex:    1,
		},
	}, nil
}

func (m *MockFinder) CreateAdvertisements(adsData []*oa.AdvertisementData, _, _ string) (overlay.TaggedBEEF, error) {
	// Create mock topics based on the advertisements
	var topics []string
	for _, adData := range adsData {
		switch adData.Protocol {
		case overlay.ProtocolSHIP, overlay.ProtocolSLAP:
			topics = append(topics, "tm_"+adData.TopicOrServiceName)
		}
	}

	// Create a valid BEEF for testing that ParseAdvertisement can work with
	// Create a simple transaction with the advertisement script
	tx := &transaction.Transaction{
		Version:  1,
		LockTime: 0,
		Inputs:   []*transaction.TransactionInput{}, // Empty inputs for test
		Outputs: []*transaction.TransactionOutput{
			{
				Satoshis:      1,
				LockingScript: createMockPushDropScript(adsData[0]),
			},
		},
	}

	// Create BEEF from the transaction
	beef, err := transaction.NewBeefFromTransaction(tx)
	if err != nil {
		return overlay.TaggedBEEF{}, err
	}

	beefBytes, err := beef.Bytes()
	if err != nil {
		return overlay.TaggedBEEF{}, err
	}

	return overlay.TaggedBEEF{
		Beef:   beefBytes,
		Topics: topics,
	}, nil
}

// Helper function to create a mock PushDrop script
func createMockPushDropScript(adData *oa.AdvertisementData) *script.Script {
	// Create a valid public key (33 bytes) - this is a known valid public key
	pubKeyHex := "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	pubKeyBytes, _ := hex.DecodeString(pubKeyHex)

	// Start building the script
	s := &script.Script{}

	// Add public key
	_ = s.AppendPushData(pubKeyBytes)

	// Add OP_CHECKSIG
	_ = s.AppendOpcodes(script.OpCHECKSIG)

	// Prepare the 5 required fields for SHIP/SLAP advertisements
	fields := [][]byte{
		[]byte(string(adData.Protocol)), // Protocol identifier
		{0x02, 0xfe, 0x8d, 0x1e, 0xb1, 0xbc, 0xb3, 0x43, 0x2b, 0x1d, 0xb5, 0x83, 0x3f, 0xf5, 0xf2, 0x22, 0x6d, 0x9c, 0xb5, 0xe6, 0x5c, 0xee, 0x43, 0x05, 0x58, 0xc1, 0x8e, 0xd3, 0xa3, 0xc8, 0x6c, 0xe1, 0xaf}, // Identity key (33 bytes)
		[]byte("https://advertise-me.com"),         // Advertised URI
		[]byte(adData.TopicOrServiceName),          // Topic
		{0x30, 0x44, 0x02, 0x20, 0x01, 0x02, 0x03}, // Mock signature (DER format)
	}

	// Add fields using PushData
	for _, field := range fields {
		_ = s.AppendPushData(field)
	}

	// Add DROP operations to remove fields from stack
	notYetDropped := len(fields)
	for notYetDropped > 1 {
		_ = s.AppendOpcodes(script.Op2DROP)
		notYetDropped -= 2
	}
	if notYetDropped != 0 {
		_ = s.AppendOpcodes(script.OpDROP)
	}

	return s
}

func TestWalletAdvertiser_ParseAdvertisement(t *testing.T) {
	t.Run("Properly parses an advertisement script", func(t *testing.T) {
		// Create advertiser with a valid test private key
		testPrivateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		advertiser, err := NewWalletAdvertiser(
			"test",
			testPrivateKey,
			"https://fake-storage-url.com",
			"https://advertise-me.com/",
			nil,
		)
		require.NoError(t, err)

		advertiser.SetSkipStorageValidation(true)
		advertiser.Finder = &MockFinder{}

		err = advertiser.Init()
		require.NoError(t, err)

		// Create an advertisement first (matching TypeScript test)
		adsData := []*oa.AdvertisementData{
			{
				Protocol:           overlay.ProtocolSHIP,
				TopicOrServiceName: "tm_meter",
			},
		}

		result, err := advertiser.CreateAdvertisements(adsData)
		require.NoError(t, err)
		require.NotNil(t, result)

		beef, err := transaction.NewBeefFromBytes(result.Beef)
		require.NoError(t, err)

		// Parse the advertisement script
		var tx *transaction.Transaction
		for _, beefTx := range beef.Transactions {
			if len(beefTx.Transaction.Outputs) > 0 {
				tx = beefTx.Transaction
				break
			}
		}
		parsedAd, err := advertiser.ParseAdvertisement(tx.Outputs[0].LockingScript)

		require.NoError(t, err)
		assert.NotNil(t, parsedAd)
		assert.Equal(t, overlay.ProtocolSHIP, parsedAd.Protocol)
		assert.Equal(t, "tm_meter", parsedAd.TopicOrService)
		assert.Equal(t, "https://advertise-me.com", parsedAd.Domain)
		assert.Equal(t, "02fe8d1eb1bcb3432b1db5833ff5f2226d9cb5e65cee430558c18ed3a3c86ce1af", parsedAd.IdentityKey)
	})

	// TODO: Sad testing (matching TypeScript comment)
}

func TestWalletAdvertiser_MethodsRequireInitialization(t *testing.T) {
	advertiser, err := NewWalletAdvertiser(
		"main",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"https://storage.example.com",
		"https://service.example.com/",
		nil,
	)
	require.NoError(t, err)

	// Test that methods fail when not initialized
	_, err = advertiser.CreateAdvertisements([]*oa.AdvertisementData{{Protocol: overlay.ProtocolSHIP, TopicOrServiceName: "test"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.FindAllAdvertisements(overlay.ProtocolSHIP)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.RevokeAdvertisements([]*oa.Advertisement{{Protocol: overlay.ProtocolSHIP}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	testScript := script.NewFromBytes([]byte{0x01})
	_, err = advertiser.ParseAdvertisement(testScript)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")
}

// Helper functions

func setupInitializedAdvertiser(t *testing.T) *WalletAdvertiser {
	advertiser, err := NewWalletAdvertiser(
		"main",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		"https://storage.example.com",
		"https://service.example.com/",
		nil,
	)
	require.NoError(t, err)

	advertiser.SetSkipStorageValidation(true) // Skip storage validation for tests

	err = advertiser.Init()
	require.NoError(t, err)

	return advertiser
}

func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
