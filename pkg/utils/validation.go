// Package utils provides utility functions for overlay discovery services.
// This package contains validation functions and helpers for working with
// overlay advertisements, topic/service names, and URI validation.
package utils

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// Compiled regex patterns for validation
var (
	// topicServiceNameRegex validates topic or service names based on BRC-87 guidelines.
	// Pattern: must start with tm_ or ls_, contain only lowercase letters and underscores
	topicServiceNameRegex = regexp.MustCompile(`^(?:tm_|ls_)[a-z]+(?:_[a-z]+)*$`)

	// numberRegex extracts numeric values from strings (for JS8 Call validation)
	numberRegex = regexp.MustCompile(`(\d+(?:\.\d+)?)`)
)

// IsAdvertisableURI checks if the provided URI is advertisable, with a recognized URI prefix.
// Applies scheme-specific validation rules as defined by the BRC-101 overlay advertisement spec.
//
// Supported schemes:
//   - HTTPS-based schemes (https://, https+bsvauth://, https+bsvauth+smf://,
//     https+bsvauth+scrypt-offchain://, https+rtt://) - Uses URL parser and disallows localhost
//   - WSS URIs (wss://) - For real-time lookup streaming, disallows localhost
//   - JS8 Call-based URIs (js8c+bsvauth+smf:) - Requires query parameters: lat, long, freq, radius
//
// Parameters:
//   - uri: The URI string to validate
//
// Returns:
//   - bool: true if the URI is valid and advertisable, false otherwise
func IsAdvertisableURI(uri string) bool {
	if uri == "" || strings.TrimSpace(uri) == "" {
		return false
	}

	// HTTPS-based schemes - disallow localhost
	if strings.HasPrefix(uri, "https://") {
		return validateCustomHTTPSURI(uri, "https://")
	} else if strings.HasPrefix(uri, "https+bsvauth://") {
		// Plain auth over HTTPS, but no payment can be collected
		return validateCustomHTTPSURI(uri, "https+bsvauth://")
	} else if strings.HasPrefix(uri, "https+bsvauth+smf://") {
		// Auth and payment over HTTPS
		return validateCustomHTTPSURI(uri, "https+bsvauth+smf://")
	} else if strings.HasPrefix(uri, "https+bsvauth+scrypt-offchain://") {
		// A protocol allowing you to also supply sCrypt off-chain values to the topical admissibility checking context
		return validateCustomHTTPSURI(uri, "https+bsvauth+scrypt-offchain://")
	} else if strings.HasPrefix(uri, "https+rtt://") {
		// A protocol allowing overlays that deal with real-time transactions (non-finals)
		return validateCustomHTTPSURI(uri, "https+rtt://")
	} else if strings.HasPrefix(uri, "wss://") {
		// WSS for real-time event-listening lookups
		return validateWSSURI(uri)
	} else if strings.HasPrefix(uri, "js8c+bsvauth+smf:") {
		// JS8 Call-based advertisement
		return validateJS8CallURI(uri)
	}

	// If none of the known prefixes match, the URI is not advertisable
	return false
}

// validateCustomHTTPSURI validates a URL by substituting its scheme if needed.
// This helper function handles custom HTTPS-based schemes by replacing them with "https://"
// for URL parsing, then validates the hostname and path.
func validateCustomHTTPSURI(uri, prefix string) bool {
	// Replace the custom scheme with "https://" for parsing
	modifiedURI := strings.Replace(uri, prefix, "https://", 1)

	parsedURL, err := url.Parse(modifiedURI)
	if err != nil {
		return false
	}

	// Disallow localhost
	if strings.ToLower(parsedURL.Hostname()) == "localhost" {
		return false
	}

	// Path must be root path only
	if parsedURL.Path != "/" {
		return false
	}

	return true
}

// validateWSSURI validates WebSocket Secure URIs for real-time lookup streaming.
func validateWSSURI(uri string) bool {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return false
	}

	if parsedURL.Scheme != "wss" {
		return false
	}

	// Disallow localhost
	if strings.ToLower(parsedURL.Hostname()) == "localhost" {
		return false
	}

	return true
}

// validateJS8CallURI validates JS8 Call-based advertisement URIs.
// Requires query string with parameters: lat, long, freq, and radius.
// Validates latitude (-90 to 90), longitude (-180 to 180), and positive frequency/radius values.
func validateJS8CallURI(uri string) bool {
	// Expect a query string with parameters
	queryIndex := strings.Index(uri, "?")
	if queryIndex == -1 {
		return false
	}

	queryStr := uri[queryIndex+1:] // Skip the '?' character
	values, err := url.ParseQuery(queryStr)
	if err != nil {
		return false
	}

	// Required parameters: lat, long, freq, and radius
	latStr := values.Get("lat")
	longStr := values.Get("long")
	freqStr := values.Get("freq")
	radiusStr := values.Get("radius")

	if latStr == "" || longStr == "" || freqStr == "" || radiusStr == "" {
		return false
	}

	// Validate latitude and longitude ranges
	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil || lat < -90 || lat > 90 {
		return false
	}

	lon, err := strconv.ParseFloat(longStr, 64)
	if err != nil || lon < -180 || lon > 180 {
		return false
	}

	// Validate frequency: extract the first number from the freq string
	// Check for negative sign first
	if strings.HasPrefix(strings.TrimSpace(freqStr), "-") {
		return false
	}
	freqMatches := numberRegex.FindStringSubmatch(freqStr)
	if len(freqMatches) < 2 {
		return false
	}
	freqVal, err := strconv.ParseFloat(freqMatches[1], 64)
	if err != nil || freqVal <= 0 {
		return false
	}

	// Validate radius: extract the first number from the radius string
	// Check for negative sign first
	if strings.HasPrefix(strings.TrimSpace(radiusStr), "-") {
		return false
	}
	radiusMatches := numberRegex.FindStringSubmatch(radiusStr)
	if len(radiusMatches) < 2 {
		return false
	}
	radiusVal, err := strconv.ParseFloat(radiusMatches[1], 64)
	if err != nil || radiusVal <= 0 {
		return false
	}

	// JS8 is more of a "demo" / "example". We include it to demonstrate that
	// overlays can be advertised in many, many ways.
	// If we were actually going to do this for real we would probably want to
	// restrict the radius to a maximum value, establish and check for allowed units.
	// Doing overlays over HF radio with js8c would be very interesting none the less.
	// For now, we assume any positive numbers are acceptable.
	return true
}

// IsValidTopicOrServiceName checks if the provided service name is valid based on BRC-87 guidelines.
//
// Rules:
//   - Must be between 1-50 characters total
//   - Must start with "tm_" (topic) or "ls_" (lookup service) prefix
//   - After prefix, must contain only lowercase letters and underscores
//   - Underscores can only separate groups of lowercase letters (no consecutive underscores)
//
// Parameters:
//   - name: The topic or service name to validate
//
// Returns:
//   - bool: true if the name is valid, false otherwise
//
// Examples:
//   - Valid: "tm_payments", "ls_identity_verification", "tm_chat_messages"
//   - Invalid: "payments", "TM_payments", "tm_", "tm__double", "tm_payments_"
func IsValidTopicOrServiceName(name string) bool {
	// Check length constraint (1-50 characters)
	if len(name) < 1 || len(name) > 50 {
		return false
	}

	// Check pattern
	return topicServiceNameRegex.MatchString(name)
}
