package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidTopicOrServiceName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid BRC-22 Topic Manager names
		{"valid topic manager name with underscores", "tm_uhrp_files", true},
		{"valid topic manager name with multiple underscores", "tm_tempo_songs", true},
		{"valid short topic manager name", "tm_a", true},
		{"valid topic manager name with multiple parts", "tm_a_b_c", true},

		// Valid BRC-24 Lookup Service names
		{"valid lookup service name with underscores", "ls_uhrp_files", true},
		{"valid lookup service name with multiple parts", "ls_tempo_songs_search", true},
		{"valid short lookup service name", "ls_a", true},
		{"valid lookup service name with two parts", "ls_a_b", true},

		// Incorrect prefix cases
		{"invalid prefix tp", "tp_uhrp_files", false},
		{"invalid prefix um", "um_tempo_songs", false},

		// Uppercase letters not allowed
		{"uppercase in middle", "tm_Tempo_songs", false},
		{"uppercase at end", "ls_tempo_Songs", false},

		// Starting or ending with underscore (outside the prefix) should fail
		{"starts with underscore", "_tm_uhrp_files", false},
		{"ends with underscore", "tm_uhrp_files_", false},

		// Consecutive underscores are not allowed
		{"consecutive underscores after prefix", "tm__uhrp_files", false},
		{"consecutive underscores in middle", "ls_tempo__songs", false},

		// Only lower-case letters and underscores allowed (no digits or special characters)
		{"contains digit", "tm_uhrp_files2", false},
		{"contains dash", "ls_tempo_songs-search", false},
		{"contains percent", "tm_uhrp%files", false},

		// Length checks: Must not exceed 50 characters
		{"exceeds 50 characters", "tm_" + strings.Repeat("a", 48), false}, // 51 chars
		{"exactly 50 characters", "tm_" + strings.Repeat("a", 47), true},  // 50 chars

		// Empty string should be invalid
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTopicOrServiceName(tt.input)
			assert.Equal(t, tt.expected, result, "IsValidTopicOrServiceName(%q) = %v, want %v", tt.input, result, tt.expected)
		})
	}
}

func TestIsAdvertisableURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		// HTTPS-based tests
		{"valid HTTPS URI", "https://example.com", true},
		{"invalid plain HTTP URI", "http://example.com", false},
		{"invalid HTTPS URI with localhost", "https://localhost", false},
		{"invalid HTTPS URI with LOCALHOST uppercase", "https://LOCALHOST:8080", false},

		// Custom HTTPS-based schemes
		{"valid https+bsvauth URI", "https+bsvauth://example.com", true},
		{"valid https+bsvauth+smf URI", "https+bsvauth+smf://example.com", true},
		{"valid https+bsvauth+scrypt-offchain URI", "https+bsvauth+scrypt-offchain://example.com", true},
		{"valid https+rtt URI", "https+rtt://example.com", true},
		{"invalid custom HTTPS URI with localhost", "https+bsvauth+smf://localhost/lookup", false},
		{"invalid https+rtt URI with localhost", "https+rtt://localhost", false},

		// Path validation
		{"invalid HTTPS URI with path", "https://example.com/path", false},
		{"invalid custom HTTPS URI with path", "https+bsvauth://example.com/path", false},

		// WebSocket scheme
		{"valid wss URI", "wss://example.com", true},
		{"invalid wss URI with localhost", "wss://localhost", false},

		// JS8 Call–based URIs
		{"valid js8c+bsvauth+smf URI with proper query parameters", "js8c+bsvauth+smf:?lat=40&long=130&freq=40meters&radius=1000miles", true},
		{"invalid js8c+bsvauth+smf URI missing query", "js8c+bsvauth+smf:", false},
		{"invalid js8c+bsvauth+smf URI missing required parameter", "js8c+bsvauth+smf:?lat=40&long=130&freq=40meters", false}, // missing radius
		{"invalid js8c+bsvauth+smf URI with non-numeric latitude", "js8c+bsvauth+smf:?lat=abc&long=130&freq=40meters&radius=1000miles", false},
		{"invalid js8c+bsvauth+smf URI with zero frequency", "js8c+bsvauth+smf:?lat=40&long=130&freq=0&radius=1000miles", false},
		{"valid js8c+bsvauth+smf URI with numeric freq and radius", "js8c+bsvauth+smf:?lat=40&long=130&freq=7.0&radius=1000", true},
		{"invalid js8c+bsvauth+smf URI with out-of-range latitude", "js8c+bsvauth+smf:?lat=100&long=130&freq=7&radius=1000", false},

		// Unknown scheme should return false
		{"unknown scheme ftp", "ftp://example.com", false},
		{"unknown scheme mailto", "mailto:user@example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAdvertisableURI(tt.uri)
			assert.Equal(t, tt.expected, result, "IsAdvertisableURI(%q) = %v, want %v", tt.uri, result, tt.expected)
		})
	}
}