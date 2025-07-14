package advertiser

import (
	"testing"

	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

func TestNewWalletAdvertiser(t *testing.T) {
	// Test private key (DO NOT USE IN PRODUCTION)
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	tests := []struct {
		name              string
		chain             string
		privateKeyHex     string
		storageURL        string
		advertisableURI   string
		lookupResolverURL string
		expectError       bool
		errorContains     string
	}{
		{
			name:              "valid configuration",
			chain:             "mainnet",
			privateKeyHex:     privateKeyHex,
			storageURL:        "https://storage.example.com",
			advertisableURI:   "https://example.com/",
			lookupResolverURL: "https://lookup.example.com",
			expectError:       false,
		},
		{
			name:              "invalid advertisable URI",
			chain:             "mainnet",
			privateKeyHex:     privateKeyHex,
			storageURL:        "https://storage.example.com",
			advertisableURI:   "http://example.com/", // HTTP not allowed
			lookupResolverURL: "https://lookup.example.com",
			expectError:       true,
			errorContains:     "refusing to initialize with non-advertisable URI",
		},
		{
			name:              "invalid private key",
			chain:             "mainnet",
			privateKeyHex:     "invalid",
			storageURL:        "https://storage.example.com",
			advertisableURI:   "https://example.com/",
			lookupResolverURL: "https://lookup.example.com",
			expectError:       true,
			errorContains:     "invalid private key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wa, err := NewWalletAdvertiser(tt.chain, tt.privateKeyHex, tt.storageURL, tt.advertisableURI, nil, nil)
			
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, wa)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wa)
				assert.Equal(t, tt.chain, wa.chain)
				assert.Equal(t, tt.storageURL, wa.storageURL)
				assert.Equal(t, tt.advertisableURI, wa.advertisableURI)
				// lookupResolverURL is not stored as a field, it's used to create lookupResolverConfig
				assert.NotEmpty(t, wa.identityKey)
				assert.False(t, wa.initialized)
			}
		})
	}
}

func TestWalletAdvertiser_Init(t *testing.T) {
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	wa, err := NewWalletAdvertiser("mainnet", privateKeyHex, "https://storage.example.com", 
		"https://example.com/", nil, nil)
	require.NoError(t, err)
	
	// Test initialization
	err = wa.Init(t.Context())
	assert.NoError(t, err)
	assert.True(t, wa.initialized)
}

func TestWalletAdvertiser_CreateAdvertisements(t *testing.T) {
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	wa, err := NewWalletAdvertiser("mainnet", privateKeyHex, "https://storage.example.com", 
		"https://example.com/", nil, nil)
	require.NoError(t, err)
	
	// Test without initialization
	_, err = wa.CreateAdvertisements(t.Context(), []*types.AdvertisementData{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialize the Advertiser using Init()")
	
	// Initialize
	err = wa.Init(t.Context())
	require.NoError(t, err)
	
	// Test with invalid topic name
	adsData := []*types.AdvertisementData{
		{
			Protocol:           overlay.ProtocolSHIP,
			TopicOrServiceName: "invalid topic!", // Contains invalid character
		},
	}
	_, err = wa.CreateAdvertisements(t.Context(), adsData)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "refusing to create SHIP advertisement with invalid topic")
}

func TestWalletAdvertiser_ParseAdvertisement(t *testing.T) {
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	wa, err := NewWalletAdvertiser("mainnet", privateKeyHex, "https://storage.example.com", 
		"https://example.com/", nil, nil)
	require.NoError(t, err)
	
	// Test parsing valid SHIP advertisement
	// This would require creating a proper PushDrop script
	// For now, test error cases
	
	// Test with empty script
	emptyScript := script.NewFromBytes([]byte{})
	_, err = wa.ParseAdvertisement(emptyScript)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty script")
}

func TestWalletAdvertiser_FindAllAdvertisements(t *testing.T) {
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	wa, err := NewWalletAdvertiser("mainnet", privateKeyHex, "https://storage.example.com", 
		"https://example.com/", nil, nil)
	require.NoError(t, err)
	
	// Test without initialization
	_, err = wa.FindAllAdvertisements(t.Context(), overlay.ProtocolSHIP)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialize the Advertiser using Init()")
	
	// Initialize
	err = wa.Init(t.Context())
	require.NoError(t, err)
	
	// Test finding advertisements (returns empty for now)
	ads, err := wa.FindAllAdvertisements(t.Context(), overlay.ProtocolSHIP)
	assert.NoError(t, err)
	assert.Empty(t, ads)
}

func TestWalletAdvertiser_RevokeAdvertisements(t *testing.T) {
	privateKeyHex := "e0d7e1b8e8ab5b8f7e6fb9b0d7c9d8e8a7b8c9d0e1f2a3b4c5d6e7f8a9b0c1d2"
	
	wa, err := NewWalletAdvertiser("mainnet", privateKeyHex, "https://storage.example.com", 
		"https://example.com/", nil, nil)
	require.NoError(t, err)
	
	// Test with empty advertisements
	_, err = wa.RevokeAdvertisements(t.Context(), []*types.Advertisement{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must provide advertisements to revoke")
	
	// Test without initialization
	ads := []*types.Advertisement{
		{
			Protocol:       overlay.ProtocolSHIP,
			IdentityKey:    "test",
			Domain:         "example.com",
			TopicOrService: "test-topic",
		},
	}
	_, err = wa.RevokeAdvertisements(t.Context(), ads)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "initialize the Advertiser using Init()")
	
	// Initialize
	err = wa.Init(t.Context())
	require.NoError(t, err)
	
	// Test revoke (not implemented yet)
	_, err = wa.RevokeAdvertisements(t.Context(), ads)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoke advertisements not yet implemented")
}