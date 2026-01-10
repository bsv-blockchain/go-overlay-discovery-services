// Package advertiser contains tests for the WalletAdvertiser functionality
package advertiser

import (
	"encoding/hex"
	"testing"

	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
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

// Tests for encoding functions

func TestWalletAdvertiser_encodeVarInt(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name     string
		value    uint64
		expected []byte
	}{
		{
			name:     "Zero",
			value:    0,
			expected: []byte{0x00},
		},
		{
			name:     "Single byte - small value",
			value:    100,
			expected: []byte{0x64},
		},
		{
			name:     "Single byte - max single byte (0xfc)",
			value:    0xfc,
			expected: []byte{0xfc},
		},
		{
			name:     "Two bytes - minimum (0xfd)",
			value:    0xfd,
			expected: []byte{0xfd, 0xfd, 0x00},
		},
		{
			name:     "Two bytes - 256",
			value:    256,
			expected: []byte{0xfd, 0x00, 0x01},
		},
		{
			name:     "Two bytes - max (0xffff)",
			value:    0xffff,
			expected: []byte{0xfd, 0xff, 0xff},
		},
		{
			name:     "Four bytes - minimum (0x10000)",
			value:    0x10000,
			expected: []byte{0xfe, 0x00, 0x00, 0x01, 0x00},
		},
		{
			name:     "Four bytes - 1 million",
			value:    1000000,
			expected: []byte{0xfe, 0x40, 0x42, 0x0f, 0x00},
		},
		{
			name:     "Four bytes - max (0xffffffff)",
			value:    0xffffffff,
			expected: []byte{0xfe, 0xff, 0xff, 0xff, 0xff},
		},
		{
			name:     "Eight bytes - minimum (0x100000000)",
			value:    0x100000000,
			expected: []byte{0xff, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00},
		},
		{
			name:     "Eight bytes - large value",
			value:    0x123456789abcdef0,
			expected: []byte{0xff, 0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := advertiser.encodeVarInt(tt.value)
			assert.Equal(t, tt.expected, result, "encodeVarInt(%d) = %v, expected %v", tt.value, result, tt.expected)
		})
	}
}

func TestWalletAdvertiser_encodeUint32(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name     string
		value    uint32
		expected []byte
	}{
		{
			name:     "Zero",
			value:    0,
			expected: []byte{0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "One",
			value:    1,
			expected: []byte{0x01, 0x00, 0x00, 0x00},
		},
		{
			name:     "256 (little endian)",
			value:    256,
			expected: []byte{0x00, 0x01, 0x00, 0x00},
		},
		{
			name:     "0x12345678 (little endian)",
			value:    0x12345678,
			expected: []byte{0x78, 0x56, 0x34, 0x12},
		},
		{
			name:     "Max uint32",
			value:    0xffffffff,
			expected: []byte{0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := advertiser.encodeUint32(tt.value)
			assert.Equal(t, tt.expected, result, "encodeUint32(%d) = %v, expected %v", tt.value, result, tt.expected)
		})
	}
}

func TestWalletAdvertiser_encodeUint64(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name     string
		value    uint64
		expected []byte
	}{
		{
			name:     "Zero",
			value:    0,
			expected: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "One",
			value:    1,
			expected: []byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "1 satoshi (typical value)",
			value:    100000000,
			expected: []byte{0x00, 0xe1, 0xf5, 0x05, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name:     "0x123456789abcdef0 (little endian)",
			value:    0x123456789abcdef0,
			expected: []byte{0xf0, 0xde, 0xbc, 0x9a, 0x78, 0x56, 0x34, 0x12},
		},
		{
			name:     "Max uint64",
			value:    0xffffffffffffffff,
			expected: []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := advertiser.encodeUint64(tt.value)
			assert.Equal(t, tt.expected, result, "encodeUint64(%d) = %v, expected %v", tt.value, result, tt.expected)
		})
	}
}

func TestWalletAdvertiser_encodeTransaction(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name        string
		tx          *Transaction
		checkPrefix []byte // Check that the result starts with these bytes
		minLen      int    // Minimum expected length
	}{
		{
			name: "Empty transaction",
			tx: &Transaction{
				Version:  1,
				Inputs:   []TransactionInput{},
				Outputs:  []TransactionOutput{},
				LockTime: 0,
			},
			// Version (4 bytes) + input count varint (1) + output count varint (1) + locktime (4)
			checkPrefix: []byte{0x01, 0x00, 0x00, 0x00}, // Version 1 little endian
			minLen:      10,
		},
		{
			name: "Transaction with one input",
			tx: &Transaction{
				Version: 2,
				Inputs: []TransactionInput{
					{
						PreviousOutput: OutPoint{
							Hash:  [32]byte{0x01, 0x02, 0x03, 0x04},
							Index: 0,
						},
						ScriptSig: []byte{0xab, 0xcd},
						Sequence:  0xffffffff,
					},
				},
				Outputs:  []TransactionOutput{},
				LockTime: 0,
			},
			checkPrefix: []byte{0x02, 0x00, 0x00, 0x00}, // Version 2 little endian
			minLen:      50,                             // Version + input count + input data + output count + locktime
		},
		{
			name: "Transaction with one output",
			tx: &Transaction{
				Version: 1,
				Inputs:  []TransactionInput{},
				Outputs: []TransactionOutput{
					{
						Value:         1000,
						LockingScript: []byte{0x76, 0xa9, 0x14}, // OP_DUP OP_HASH160 <20 bytes>
					},
				},
				LockTime: 500000,
			},
			checkPrefix: []byte{0x01, 0x00, 0x00, 0x00}, // Version 1 little endian
			minLen:      20,
		},
		{
			name: "Full transaction with inputs and outputs",
			tx: &Transaction{
				Version: 1,
				Inputs: []TransactionInput{
					{
						PreviousOutput: OutPoint{
							Hash:  [32]byte{0xff, 0xee, 0xdd, 0xcc},
							Index: 1,
						},
						ScriptSig: []byte{0x48, 0x30, 0x45}, // Partial signature bytes
						Sequence:  0xfffffffe,
					},
				},
				Outputs: []TransactionOutput{
					{
						Value:         50000,
						LockingScript: []byte{0x76, 0xa9, 0x14, 0x00, 0x01, 0x02},
					},
					{
						Value:         10000,
						LockingScript: []byte{0xa9, 0x14},
					},
				},
				LockTime: 0,
			},
			checkPrefix: []byte{0x01, 0x00, 0x00, 0x00},
			minLen:      70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := advertiser.encodeTransaction(tt.tx)

			// Verify minimum length
			assert.GreaterOrEqual(t, len(result), tt.minLen, "encoded transaction should be at least %d bytes", tt.minLen)

			// Verify prefix (version)
			assert.Equal(t, tt.checkPrefix, result[:len(tt.checkPrefix)], "transaction should start with correct version bytes")

			// Verify locktime is at the end (4 bytes)
			lockTimeBytes := advertiser.encodeUint32(tt.tx.LockTime)
			assert.Equal(t, lockTimeBytes, result[len(result)-4:], "transaction should end with correct locktime bytes")
		})
	}
}

func TestWalletAdvertiser_encodeTransactionsAsBEEF(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tests := []struct {
		name         string
		transactions []*Transaction
		checkMagic   bool
		minLen       int
	}{
		{
			name:         "Empty transactions",
			transactions: []*Transaction{},
			checkMagic:   true,
			minLen:       9, // BEEF (4) + version (4) + tx count varint (1)
		},
		{
			name: "Single empty transaction",
			transactions: []*Transaction{
				{
					Version:  1,
					Inputs:   []TransactionInput{},
					Outputs:  []TransactionOutput{},
					LockTime: 0,
				},
			},
			checkMagic: true,
			minLen:     19, // BEEF header + empty tx
		},
		{
			name: "Multiple transactions",
			transactions: []*Transaction{
				{
					Version:  1,
					Inputs:   []TransactionInput{},
					Outputs:  []TransactionOutput{},
					LockTime: 0,
				},
				{
					Version: 2,
					Inputs: []TransactionInput{
						{
							PreviousOutput: OutPoint{Hash: [32]byte{0x01}, Index: 0},
							ScriptSig:      []byte{0xaa},
							Sequence:       0xffffffff,
						},
					},
					Outputs:  []TransactionOutput{},
					LockTime: 100,
				},
			},
			checkMagic: true,
			minLen:     60,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := advertiser.encodeTransactionsAsBEEF(tt.transactions)

			// Verify minimum length
			assert.GreaterOrEqual(t, len(result), tt.minLen, "BEEF data should be at least %d bytes", tt.minLen)

			if tt.checkMagic {
				// Verify BEEF magic bytes
				assert.Equal(t, []byte("BEEF"), result[:4], "BEEF data should start with magic bytes")

				// Verify version (0x01000000 little endian)
				assert.Equal(t, []byte{0x01, 0x00, 0x00, 0x00}, result[4:8], "BEEF should have version 1")

				// Verify transaction count varint
				txCountVarInt := advertiser.encodeVarInt(uint64(len(tt.transactions)))
				assert.Equal(t, txCountVarInt, result[8:8+len(txCountVarInt)], "BEEF should have correct transaction count")
			}
		})
	}
}

func TestWalletAdvertiser_encodeTransaction_Deterministic(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	tx := &Transaction{
		Version: 1,
		Inputs: []TransactionInput{
			{
				PreviousOutput: OutPoint{
					Hash:  [32]byte{0xaa, 0xbb, 0xcc, 0xdd},
					Index: 5,
				},
				ScriptSig: []byte{0x01, 0x02, 0x03, 0x04, 0x05},
				Sequence:  0xffffffff,
			},
		},
		Outputs: []TransactionOutput{
			{
				Value:         12345,
				LockingScript: []byte{0x76, 0xa9},
			},
		},
		LockTime: 0,
	}

	// Encode multiple times and verify deterministic output
	result1 := advertiser.encodeTransaction(tx)
	result2 := advertiser.encodeTransaction(tx)
	result3 := advertiser.encodeTransaction(tx)

	assert.Equal(t, result1, result2, "encodeTransaction should be deterministic")
	assert.Equal(t, result2, result3, "encodeTransaction should be deterministic")
}

func TestWalletAdvertiser_encodeTransactionsAsBEEF_Deterministic(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	transactions := []*Transaction{
		{
			Version:  1,
			Inputs:   []TransactionInput{},
			Outputs:  []TransactionOutput{{Value: 1000, LockingScript: []byte{0x51}}},
			LockTime: 0,
		},
	}

	// Encode multiple times and verify deterministic output
	result1 := advertiser.encodeTransactionsAsBEEF(transactions)
	result2 := advertiser.encodeTransactionsAsBEEF(transactions)
	result3 := advertiser.encodeTransactionsAsBEEF(transactions)

	assert.Equal(t, result1, result2, "encodeTransactionsAsBEEF should be deterministic")
	assert.Equal(t, result2, result3, "encodeTransactionsAsBEEF should be deterministic")
}

func TestWalletAdvertiser_encodeTransaction_StructureValidation(t *testing.T) {
	advertiser := setupInitializedAdvertiser(t)

	// Create a transaction with known values to validate byte-level structure
	tx := &Transaction{
		Version: 1,
		Inputs: []TransactionInput{
			{
				PreviousOutput: OutPoint{
					Hash:  [32]byte{}, // 32 zero bytes
					Index: 0,
				},
				ScriptSig: []byte{},
				Sequence:  0xffffffff,
			},
		},
		Outputs: []TransactionOutput{
			{
				Value:         0,
				LockingScript: []byte{},
			},
		},
		LockTime: 0,
	}

	result := advertiser.encodeTransaction(tx)

	// Verify structure:
	// - Version: 4 bytes (0x01000000)
	// - Input count: 1 byte (0x01)
	// - Input: 32 (hash) + 4 (index) + 1 (script len) + 0 (script) + 4 (sequence) = 41 bytes
	// - Output count: 1 byte (0x01)
	// - Output: 8 (value) + 1 (script len) + 0 (script) = 9 bytes
	// - Locktime: 4 bytes (0x00000000)
	// Total: 4 + 1 + 41 + 1 + 9 + 4 = 60 bytes
	expectedLen := 60
	assert.Len(t, result, expectedLen, "transaction should be exactly %d bytes", expectedLen)

	// Verify version (bytes 0-3)
	assert.Equal(t, []byte{0x01, 0x00, 0x00, 0x00}, result[0:4], "version should be 1")

	// Verify input count (byte 4)
	assert.Equal(t, byte(0x01), result[4], "input count should be 1")

	// Verify output count (byte 46 = 4 + 1 + 41)
	assert.Equal(t, byte(0x01), result[46], "output count should be 1")

	// Verify locktime (last 4 bytes)
	assert.Equal(t, []byte{0x00, 0x00, 0x00, 0x00}, result[56:60], "locktime should be 0")
}
