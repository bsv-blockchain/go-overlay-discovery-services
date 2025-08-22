// Package advertiser contains tests for the WalletAdvertiser functionality
package advertiser

import (
	"github.com/bsv-blockchain/go-sdk/transaction"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	overlayAdvertiser "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
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
				assert.NoError(t, err)
				assert.NotNil(t, advertiser)
				assert.Equal(t, tt.chain, advertiser.GetChain())
				assert.Equal(t, tt.storageURL, advertiser.GetStorageURL())
				assert.Equal(t, tt.advertisableURI, advertiser.GetAdvertisableURI())
				assert.False(t, advertiser.IsInitialized())
			} else {
				assert.Error(t, err)
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
	assert.NoError(t, err)
	assert.True(t, advertiser.IsInitialized())

	// Test double initialization
	err = advertiser.Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser is already initialized")
}

func TestWalletAdvertiser_CreateAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name          string
		adsData       []*overlayAdvertiser.AdvertisementData
		expectedError string
		shouldFail    bool
	}{
		{
			name: "Valid SHIP advertisement",
			adsData: []*overlayAdvertiser.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSHIP,
					TopicOrServiceName: "payments",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name: "Valid SLAP advertisement",
			adsData: []*overlayAdvertiser.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSLAP,
					TopicOrServiceName: "identity_verification",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name:          "Empty advertisements array",
			adsData:       []*overlayAdvertiser.AdvertisementData{},
			expectedError: "at least one advertisement data entry is required",
		},
		{
			name: "Empty topic name",
			adsData: []*overlayAdvertiser.AdvertisementData{
				{
					Protocol:           overlay.ProtocolSHIP,
					TopicOrServiceName: "",
				},
			},
			expectedError: "topicOrServiceName cannot be empty",
		},
		{
			name: "Invalid topic name",
			adsData: []*overlayAdvertiser.AdvertisementData{
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
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestWalletAdvertiser_FindAllAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

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
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestWalletAdvertiser_RevokeAdvertisements(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name           string
		advertisements []*overlayAdvertiser.Advertisement
		expectedError  string
		shouldFail     bool
	}{
		{
			name: "Valid advertisement with BEEF",
			advertisements: []*overlayAdvertiser.Advertisement{
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
			advertisements: []*overlayAdvertiser.Advertisement{},
			expectedError:  "at least one advertisement is required for revocation",
		},
		{
			name: "Advertisement missing BEEF",
			advertisements: []*overlayAdvertiser.Advertisement{
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
			advertisements: []*overlayAdvertiser.Advertisement{
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
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
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
		advertiser.SetTestMode(true)

		err = advertiser.Init()
		require.NoError(t, err)

		// Create an advertisement first (matching TypeScript test)
		adsData := []*overlayAdvertiser.AdvertisementData{
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

		assert.NoError(t, err)
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
	_, err = advertiser.CreateAdvertisements([]*overlayAdvertiser.AdvertisementData{{Protocol: overlay.ProtocolSHIP, TopicOrServiceName: "test"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.FindAllAdvertisements(overlay.ProtocolSHIP)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.RevokeAdvertisements([]*overlayAdvertiser.Advertisement{{Protocol: overlay.ProtocolSHIP}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	testScript := script.NewFromBytes([]byte{0x01})
	_, err = advertiser.ParseAdvertisement(testScript)
	assert.Error(t, err)
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
	advertiser.SetTestMode(true)              // Enable test mode for mock data

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
