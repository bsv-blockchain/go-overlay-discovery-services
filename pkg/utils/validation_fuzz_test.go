// Package utils provides utility functions for validating URIs and service names.
package utils

import (
	"testing"
)

// FuzzIsAdvertisableURI tests the IsAdvertisableURI function with random inputs
// to ensure it never panics and handles all edge cases gracefully.
func FuzzIsAdvertisableURI(f *testing.F) {
	// Seed corpus with valid examples from different schemes
	f.Add("https://example.com/")
	f.Add("https+bsvauth://example.com/")
	f.Add("https+bsvauth+smf://example.com/")
	f.Add("https+bsvauth+scrypt-offchain://example.com/")
	f.Add("https+rtt://example.com/")
	f.Add("wss://example.com")
	f.Add("js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100")

	// Seed corpus with invalid examples
	f.Add("")
	f.Add("   ")
	f.Add("http://example.com")
	f.Add("https://localhost/")
	f.Add("https://example.com/path")
	f.Add("ftp://example.com")
	f.Add("js8c+bsvauth+smf:")
	f.Add("js8c+bsvauth+smf:?lat=91&long=0&freq=1&radius=1")

	// Seed corpus with edge cases
	f.Add("https://")
	f.Add("://example.com")
	f.Add("https://example.com:99999/")
	f.Add("wss://")
	f.Add("js8c+bsvauth+smf:?")
	f.Add("https://[::1]/")
	f.Add("https://192.168.1.1/")

	f.Fuzz(func(t *testing.T, uri string) {
		// Function should not panic on any input
		result := IsAdvertisableURI(uri)

		// If result is true, verify it's actually a valid URI format
		// (we can't validate full correctness here, but we can ensure consistency)
		if result {
			// Valid URIs should not be empty or whitespace-only
			if uri == "" || len(uri) < 4 {
				t.Errorf("IsAdvertisableURI returned true for invalid short URI: %q", uri)
			}
		}
	})
}

// FuzzIsValidTopicOrServiceName tests the IsValidTopicOrServiceName function
// with random inputs to ensure it handles all edge cases without panicking.
func FuzzIsValidTopicOrServiceName(f *testing.F) {
	// Seed corpus with valid topic names
	f.Add("tm_payments")
	f.Add("tm_chat_messages")
	f.Add("tm_identity_verification_service")
	f.Add("tm_a")
	f.Add("ls_payments")
	f.Add("ls_identity_verification")

	// Seed corpus with invalid examples
	f.Add("")
	f.Add("tm_")
	f.Add("ls_")
	f.Add("payments")
	f.Add("TM_payments")
	f.Add("tm_Payments")
	f.Add("tm_payments123")
	f.Add("tm_payments-special")
	f.Add("tm_payments__double")
	f.Add("tm_payments_")
	f.Add("tm__payments")

	// Seed corpus with edge cases
	f.Add("t")
	f.Add("tm")
	f.Add("ls")
	f.Add("sv_payments")
	// exactly 50 chars
	f.Add("tm_" + string(make([]byte, 48)))
	// too long
	f.Add("tm_" + string(make([]byte, 100)))

	f.Fuzz(func(t *testing.T, name string) {
		// Function should not panic on any input
		result := IsValidTopicOrServiceName(name)

		// If result is true, verify basic constraints
		if result {
			// Valid names must be within length constraints
			if len(name) < 1 || len(name) > 50 {
				t.Errorf("IsValidTopicOrServiceName returned true for invalid length: %d (name: %q)", len(name), name)
			}

			// Valid names must start with tm_ or ls_
			if len(name) >= 3 {
				prefix := name[:3]
				if prefix != "tm_" && prefix != "ls_" {
					t.Errorf("IsValidTopicOrServiceName returned true for invalid prefix: %q (name: %q)", prefix, name)
				}
			}
		}
	})
}

// FuzzValidateCustomHTTPSURI tests the validateCustomHTTPSURI function
// with random inputs to ensure robustness.
func FuzzValidateCustomHTTPSURI(f *testing.F) {
	// Seed corpus with valid examples
	f.Add("custom://example.com/", "custom://")
	f.Add("https://example.com/", "https://")
	f.Add("https+bsvauth://example.com/", "https+bsvauth://")

	// Seed corpus with invalid examples
	f.Add("custom://localhost/", "custom://")
	f.Add("custom://example.com/path", "custom://")
	f.Add("custom://[invalid", "custom://")

	// Seed corpus with edge cases
	f.Add("custom://", "custom://")
	f.Add("://example.com/", "://")
	f.Add("custom://192.168.1.1/", "custom://")
	f.Add("custom://[::1]/", "custom://")

	f.Fuzz(func(t *testing.T, uri, prefix string) {
		// Function should not panic on any input
		_ = validateCustomHTTPSURI(uri, prefix)
		// We don't validate the result as this is an internal function
		// The main goal is to ensure it doesn't panic
	})
}

// FuzzValidateWSSURI tests the validateWSSURI function with random inputs.
func FuzzValidateWSSURI(f *testing.F) {
	// Seed corpus with valid examples
	f.Add("wss://example.com")
	f.Add("wss://example.com:443")
	f.Add("wss://example.com/path")

	// Seed corpus with invalid examples
	f.Add("wss://localhost")
	f.Add("ws://example.com")
	f.Add("wss://[invalid")
	f.Add("")

	// Seed corpus with edge cases
	f.Add("wss://")
	f.Add("wss://192.168.1.1")
	f.Add("wss://[::1]")
	f.Add("wss://example.com:99999")

	f.Fuzz(func(t *testing.T, uri string) {
		// Function should not panic on any input
		_ = validateWSSURI(uri)
		// We don't validate the result as this is an internal function
		// The main goal is to ensure it doesn't panic
	})
}

// FuzzValidateJS8CallURI tests the validateJS8CallURI function with random inputs.
func FuzzValidateJS8CallURI(f *testing.F) {
	// Seed corpus with valid examples
	f.Add("js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100")
	f.Add("js8c+bsvauth+smf:?lat=0&long=0&freq=1&radius=1")
	f.Add("js8c+bsvauth+smf:?lat=90&long=180&freq=7.078MHz&radius=100km")
	f.Add("js8c+bsvauth+smf:?lat=-90&long=-180&freq=7.078&radius=100.5")

	// Seed corpus with invalid examples
	f.Add("js8c+bsvauth+smf:")
	f.Add("js8c+bsvauth+smf:?lat=91&long=0&freq=1&radius=1")
	f.Add("js8c+bsvauth+smf:?lat=0&long=181&freq=1&radius=1")
	f.Add("js8c+bsvauth+smf:?lat=0&long=0&freq=0&radius=1")
	f.Add("js8c+bsvauth+smf:?lat=0&long=0&freq=1&radius=0")
	f.Add("js8c+bsvauth+smf:?lat=0&long=0&freq=-1&radius=1")

	// Seed corpus with edge cases
	f.Add("js8c+bsvauth+smf:?")
	f.Add("js8c+bsvauth+smf:?lat=&long=&freq=&radius=")
	f.Add("js8c+bsvauth+smf:?lat=invalid&long=invalid&freq=invalid&radius=invalid")
	f.Add("js8c+bsvauth+smf:?lat=0&long=0&freq=abc&radius=xyz")

	f.Fuzz(func(t *testing.T, uri string) {
		// Function should not panic on any input
		_ = validateJS8CallURI(uri)
		// We don't validate the result as this is an internal function
		// The main goal is to ensure it doesn't panic
	})
}
