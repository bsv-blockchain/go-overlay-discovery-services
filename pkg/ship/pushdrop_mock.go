package ship

import (
	"encoding/hex"
	"fmt"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// MockPushDropDecoder provides a simple mock implementation of PushDropDecoder for testing
type MockPushDropDecoder struct{}

// NewMockPushDropDecoder creates a new mock PushDrop decoder
func NewMockPushDropDecoder() *MockPushDropDecoder {
	return &MockPushDropDecoder{}
}

// Decode provides a mock implementation that returns predefined fields for testing
// In a real implementation, this would parse the locking script according to PushDrop format
func (m *MockPushDropDecoder) Decode(lockingScript string) (*types.PushDropResult, error) {
	// For testing purposes, we'll return a fixed result
	// In a real implementation, this would decode the actual locking script
	if lockingScript == "" {
		return nil, fmt.Errorf("empty locking script")
	}

	// Validate hex format
	if _, err := hex.DecodeString(lockingScript); err != nil {
		return nil, fmt.Errorf("invalid hex format: %w", err)
	}

	// Return mock SHIP data
	return &types.PushDropResult{
		Fields: [][]byte{
			[]byte("SHIP"),                 // Protocol identifier
			[]byte{0x01, 0x02, 0x03, 0x04}, // Mock identity key
			[]byte("https://example.com"),  // Mock domain
			[]byte("tm_bridge"),            // Mock topic
		},
	}, nil
}

// MockUtils provides a simple mock implementation of Utils for testing
type MockUtils struct{}

// NewMockUtils creates a new mock utils instance
func NewMockUtils() *MockUtils {
	return &MockUtils{}
}

// ToUTF8 converts byte data to UTF-8 string
func (m *MockUtils) ToUTF8(data []byte) string {
	return string(data)
}

// ToHex converts byte data to hexadecimal string
func (m *MockUtils) ToHex(data []byte) string {
	return hex.EncodeToString(data)
}
