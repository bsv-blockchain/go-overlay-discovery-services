// Package utils provides utility functions for validating URIs and service names.
package utils

import (
	"strings"
	"testing"
)

func TestIsAdvertisableURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		// Empty/invalid inputs
		{"empty string", "", false},
		{"whitespace only", "   ", false},

		// HTTPS URIs
		{"valid https", "https://example.com/", true},
		{"https with localhost", "https://localhost/", false},
		{"https with path", "https://example.com/path", false},

		// HTTPS+bsvauth URIs
		{"valid https+bsvauth", "https+bsvauth://example.com/", true},
		{"https+bsvauth with localhost", "https+bsvauth://localhost/", false},

		// HTTPS+bsvauth+smf URIs
		{"valid https+bsvauth+smf", "https+bsvauth+smf://example.com/", true},
		{"https+bsvauth+smf with localhost", "https+bsvauth+smf://localhost/", false},

		// HTTPS+bsvauth+scrypt-offchain URIs
		{"valid https+bsvauth+scrypt-offchain", "https+bsvauth+scrypt-offchain://example.com/", true},
		{"https+bsvauth+scrypt-offchain with localhost", "https+bsvauth+scrypt-offchain://localhost/", false},

		// HTTPS+rtt URIs
		{"valid https+rtt", "https+rtt://example.com/", true},
		{"https+rtt with localhost", "https+rtt://localhost/", false},

		// WSS URIs
		{"valid wss", "wss://example.com", true},
		{"wss with localhost", "wss://localhost", false},
		{"invalid wss scheme", "ws://example.com", false},

		// JS8 Call URIs
		{"valid js8c", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100", true},
		{"js8c missing query", "js8c+bsvauth+smf:", false},
		{"js8c missing lat", "js8c+bsvauth+smf:?long=-74.0060&freq=7.078&radius=100", false},
		{"js8c missing long", "js8c+bsvauth+smf:?lat=40.7128&freq=7.078&radius=100", false},
		{"js8c missing freq", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&radius=100", false},
		{"js8c missing radius", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078", false},
		{"js8c invalid lat high", "js8c+bsvauth+smf:?lat=91&long=-74.0060&freq=7.078&radius=100", false},
		{"js8c invalid lat low", "js8c+bsvauth+smf:?lat=-91&long=-74.0060&freq=7.078&radius=100", false},
		{"js8c invalid long high", "js8c+bsvauth+smf:?lat=40.7128&long=181&freq=7.078&radius=100", false},
		{"js8c invalid long low", "js8c+bsvauth+smf:?lat=40.7128&long=-181&freq=7.078&radius=100", false},
		{"js8c invalid freq", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=abc&radius=100", false},
		{"js8c zero freq", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=0&radius=100", false},
		{"js8c negative freq", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=-7.078&radius=100", false},
		{"js8c invalid radius", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=xyz", false},
		{"js8c zero radius", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=0", false},
		{"js8c freq with units", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078MHz&radius=100km", true},
		{"js8c decimal values", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100.5", true},

		// Unsupported schemes
		{"http scheme", "http://example.com", false},
		{"ftp scheme", "ftp://example.com", false},
		{"unknown scheme", "unknown://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAdvertisableURI(tt.uri)
			if result != tt.expected {
				t.Errorf("IsAdvertisableURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestIsValidTopicOrServiceName(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		expected bool
	}{
		// Valid topic names
		{"valid topic simple", "tm_payments", true},
		{"valid topic with underscores", "tm_chat_messages", true},
		{"valid topic complex", "tm_identity_verification_service", true},

		// Valid lookup service names
		{"valid lookup service simple", "ls_payments", true},
		{"valid lookup service with underscores", "ls_identity_verification", true},

		// Invalid - wrong prefix
		{"no prefix", "payments", false},
		{"wrong prefix", "sv_payments", false},
		{"uppercase prefix", "TM_payments", false},
		{"Ls prefix", "Ls_payments", false},

		// Invalid - bad format after prefix
		{"empty after prefix", "tm_", false},
		{"uppercase letters", "tm_Payments", false},
		{"numbers", "tm_payments123", false},
		{"special characters", "tm_payments-special", false},
		{"consecutive underscores", "tm_payments__double", false},
		{"trailing underscore", "tm_payments_", false},
		{"leading underscore after prefix", "tm__payments", false},

		// Invalid - length constraints
		{"too long", "tm_" + strings.Repeat("a", 48), false},        // 50+ chars total
		{"exactly 50 chars", "tm_" + strings.Repeat("a", 47), true}, // exactly 50 chars
		{"single char", "t", false},

		// Edge cases
		{"empty string", "", false},
		{"just tm", "tm", false},
		{"just ls", "ls", false},
		{"minimal valid tm", "tm_a", true},
		{"minimal valid ls", "ls_a", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTopicOrServiceName(tt.service)
			if result != tt.expected {
				t.Errorf("IsValidTopicOrServiceName(%q) = %v, expected %v", tt.service, result, tt.expected)
			}
		})
	}
}

func TestValidateCustomHTTPSURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		prefix   string
		expected bool
	}{
		{"valid custom scheme", "custom://example.com/", "custom://", true},
		{"localhost blocked", "custom://localhost/", "custom://", false},
		{"path not allowed", "custom://example.com/path", "custom://", false},
		{"malformed URL", "custom://[invalid", "custom://", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCustomHTTPSURI(tt.uri, tt.prefix)
			if result != tt.expected {
				t.Errorf("validateCustomHTTPSURI(%q, %q) = %v, expected %v", tt.uri, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestValidateWSSURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{"valid wss", "wss://example.com", true},
		{"localhost blocked", "wss://localhost", false},
		{"wrong scheme", "ws://example.com", false},
		{"malformed URL", "wss://[invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateWSSURI(tt.uri)
			if result != tt.expected {
				t.Errorf("validateWSSURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}

func TestValidateJS8CallURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected bool
	}{
		{"valid complete", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100", true},
		{"no query string", "js8c+bsvauth+smf:", false},
		{"missing lat", "js8c+bsvauth+smf:?long=-74.0060&freq=7.078&radius=100", false},
		{"invalid lat", "js8c+bsvauth+smf:?lat=invalid&long=-74.0060&freq=7.078&radius=100", false},
		{"lat out of range high", "js8c+bsvauth+smf:?lat=91&long=-74.0060&freq=7.078&radius=100", false},
		{"lat out of range low", "js8c+bsvauth+smf:?lat=-91&long=-74.0060&freq=7.078&radius=100", false},
		{"long out of range high", "js8c+bsvauth+smf:?lat=40.7128&long=181&freq=7.078&radius=100", false},
		{"long out of range low", "js8c+bsvauth+smf:?lat=40.7128&long=-181&freq=7.078&radius=100", false},
		{"zero frequency", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=0&radius=100", false},
		{"negative frequency", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=-1&radius=100", false},
		{"zero radius", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=0", false},
		{"freq with units", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078MHz&radius=100", true},
		{"malformed query", "js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100&invalid=%", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateJS8CallURI(tt.uri)
			if result != tt.expected {
				t.Errorf("validateJS8CallURI(%q) = %v, expected %v", tt.uri, result, tt.expected)
			}
		})
	}
}
