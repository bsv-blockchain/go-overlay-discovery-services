// Package advertiser contains tests for the WalletAdvertiser functionality
package advertiser

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPushDropDecoder is a mock implementation of the PushDropDecoder interface
type MockPushDropDecoder struct {
	mock.Mock
}

func (m *MockPushDropDecoder) Decode(lockingScript string) (*types.PushDropResult, error) {
	args := m.Called(lockingScript)
	return args.Get(0).(*types.PushDropResult), args.Error(1)
}

// MockUtils is a mock implementation of the Utils interface
type MockUtils struct {
	mock.Mock
}

func (m *MockUtils) ToUTF8(data []byte) string {
	args := m.Called(data)
	return args.String(0)
}

func (m *MockUtils) ToHex(data []byte) string {
	args := m.Called(data)
	return args.String(0)
}

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

	// Test initialization without dependencies
	err = advertiser.Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PushDropDecoder must be set before initialization")

	// Set PushDrop decoder but not utils
	mockDecoder := &MockPushDropDecoder{}
	advertiser.SetPushDropDecoder(mockDecoder)

	err = advertiser.Init()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Utils must be set before initialization")

	// Set both dependencies
	mockUtils := &MockUtils{}
	// Set up expectations for the cryptographic context setup
	mockUtils.On("ToUTF8", []byte("test")).Return("test")
	mockUtils.On("ToHex", []byte("test")).Return("74657374")

	// Set up expectation for PushDrop decoder test (it's okay if it returns an error)
	mockDecoder.On("Decode", mock.AnythingOfType("string")).Return(&types.PushDropResult{Fields: [][]byte{}}, fmt.Errorf("test error"))

	advertiser.SetUtils(mockUtils)
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
		adsData       []types.AdvertisementData
		expectedError string
		shouldFail    bool
	}{
		{
			name: "Valid SHIP advertisement",
			adsData: []types.AdvertisementData{
				{
					Protocol:           types.ProtocolSHIP,
					TopicOrServiceName: "payments",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name: "Valid SLAP advertisement",
			adsData: []types.AdvertisementData{
				{
					Protocol:           types.ProtocolSLAP,
					TopicOrServiceName: "identity_verification",
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name:          "Empty advertisements array",
			adsData:       []types.AdvertisementData{},
			expectedError: "at least one advertisement data entry is required",
		},
		{
			name: "Invalid protocol",
			adsData: []types.AdvertisementData{
				{
					Protocol:           "INVALID",
					TopicOrServiceName: "payments",
				},
			},
			expectedError: "unsupported protocol",
		},
		{
			name: "Empty topic name",
			adsData: []types.AdvertisementData{
				{
					Protocol:           types.ProtocolSHIP,
					TopicOrServiceName: "",
				},
			},
			expectedError: "topicOrServiceName cannot be empty",
		},
		{
			name: "Invalid topic name",
			adsData: []types.AdvertisementData{
				{
					Protocol:           types.ProtocolSHIP,
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
		protocol      string
		expectedError string
		shouldFail    bool
	}{
		{
			name:       "Valid SHIP protocol",
			protocol:   "SHIP",
			shouldFail: false, // Implementation is now complete
		},
		{
			name:       "Valid SLAP protocol",
			protocol:   "SLAP",
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
		advertisements []types.Advertisement
		expectedError  string
		shouldFail     bool
	}{
		{
			name: "Valid advertisement with BEEF",
			advertisements: []types.Advertisement{
				{
					Protocol:       types.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					Beef:           []byte("BEEF\x01\x00\x00\x00\x01\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00"), // Valid minimal BEEF data
					OutputIndex:    intPtr(0),
				},
			},
			shouldFail: false, // Implementation is now complete
		},
		{
			name:           "Empty advertisements array",
			advertisements: []types.Advertisement{},
			expectedError:  "at least one advertisement is required for revocation",
		},
		{
			name: "Advertisement missing BEEF",
			advertisements: []types.Advertisement{
				{
					Protocol:       types.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					OutputIndex:    intPtr(0),
				},
			},
			expectedError: "advertisement at index 0 is missing BEEF data required for revocation",
		},
		{
			name: "Advertisement missing output index",
			advertisements: []types.Advertisement{
				{
					Protocol:       types.ProtocolSHIP,
					IdentityKey:    "test-key",
					Domain:         "example.com",
					TopicOrService: "payments",
					Beef:           []byte("test-beef"),
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
	advertiser := setupInitializedAdvertiser(t)

	t.Run("Valid SHIP advertisement", func(t *testing.T) {
		// Create a fresh advertiser for this test with clean mocks
		testAdvertiser, err := NewWalletAdvertiser(
			"main",
			"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"https://storage.example.com",
			"https://service.example.com/",
			nil,
		)
		require.NoError(t, err)

		testMockDecoder := &MockPushDropDecoder{}
		testMockUtils := &MockUtils{}

		outputScript := []byte{0x01, 0x02, 0x03}
		scriptHex := hex.EncodeToString(outputScript)

		// Set up mock expectations for initialization
		testMockUtils.On("ToUTF8", []byte("test")).Return("test")
		testMockUtils.On("ToHex", []byte("test")).Return("74657374")
		testMockDecoder.On("Decode", mock.AnythingOfType("string")).Return(&types.PushDropResult{Fields: [][]byte{}}, fmt.Errorf("test error")).Once()

		// Set up mock expectations for the actual test
		testMockDecoder.On("Decode", scriptHex).Return(&types.PushDropResult{
			Fields: [][]byte{
				[]byte("SHIP"),
				[]byte{0xab, 0xcd, 0xef},
				[]byte("example.com"),
				[]byte("payments"),
			},
		}, nil)

		testMockUtils.On("ToUTF8", []byte("SHIP")).Return("SHIP")
		testMockUtils.On("ToHex", []byte{0xab, 0xcd, 0xef}).Return("abcdef")
		testMockUtils.On("ToUTF8", []byte("example.com")).Return("example.com")
		testMockUtils.On("ToUTF8", []byte("payments")).Return("payments")

		testAdvertiser.SetPushDropDecoder(testMockDecoder)
		testAdvertiser.SetUtils(testMockUtils)
		testAdvertiser.SetSkipStorageValidation(true)

		err = testAdvertiser.Init()
		require.NoError(t, err)

		result, err := testAdvertiser.ParseAdvertisement(outputScript)

		assert.NoError(t, err)
		assert.Equal(t, types.ProtocolSHIP, result.Protocol)
		assert.Equal(t, "abcdef", result.IdentityKey)
		assert.Equal(t, "example.com", result.Domain)
		assert.Equal(t, "payments", result.TopicOrService)

		testMockDecoder.AssertExpectations(t)
		testMockUtils.AssertExpectations(t)
	})

	t.Run("Empty output script", func(t *testing.T) {
		result, err := advertiser.ParseAdvertisement([]byte{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "output script cannot be empty")
		assert.Equal(t, types.Advertisement{}, result)
	})
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
	_, err = advertiser.CreateAdvertisements([]types.AdvertisementData{{Protocol: types.ProtocolSHIP, TopicOrServiceName: "test"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.FindAllAdvertisements("SHIP")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.RevokeAdvertisements([]types.Advertisement{{Protocol: types.ProtocolSHIP}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WalletAdvertiser must be initialized")

	_, err = advertiser.ParseAdvertisement([]byte{0x01})
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

	mockDecoder := &MockPushDropDecoder{}
	mockUtils := &MockUtils{}

	// Set up expectations for the cryptographic context setup
	mockUtils.On("ToUTF8", []byte("test")).Return("test")
	mockUtils.On("ToHex", []byte("test")).Return("74657374")

	// Set up expectation for PushDrop decoder test (it's okay if it returns an error)
	mockDecoder.On("Decode", mock.AnythingOfType("string")).Return(&types.PushDropResult{Fields: [][]byte{}}, fmt.Errorf("test error"))

	advertiser.SetPushDropDecoder(mockDecoder)
	advertiser.SetUtils(mockUtils)
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
