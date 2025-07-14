package utils

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// IsValidTopicOrServiceName checks if the given name is valid based on BRC-87 guidelines
func IsValidTopicOrServiceName(name string) bool {
	// Pattern matches: ^(?=.{1,50}$)(?:tm_|ls_)[a-z]+(?:_[a-z]+)*$
	// This ensures:
	// - Total length between 1-50 characters
	// - Must start with either "tm_" or "ls_"
	// - Followed by lowercase letters, optionally with underscores between words
	if len(name) < 1 || len(name) > 50 {
		return false
	}
	match, _ := regexp.MatchString("^(?:tm_|ls_)[a-z]+(?:_[a-z]+)*$", name)
	return match
}

// IsAdvertisableURI checks if the provided URI is advertisable, with a recognized URI prefix.
// Applies scheme-specific validation rules as defined by the BRC-101 overlay advertisement spec.
func IsAdvertisableURI(uri string) bool {
	if uri == "" || strings.TrimSpace(uri) == "" {
		return false
	}

	// Helper function: validate a URL by substituting its scheme if needed.
	validateCustomHTTPSURI := func(uri string, prefix string) bool {
		// Replace the custom prefix with https:// for URL parsing
		modifiedURI := strings.Replace(uri, prefix, "https://", 1)
		
		parsedURL, err := url.Parse(modifiedURI)
		if err != nil {
			return false
		}
		
		// Must have hostname
		if parsedURL.Hostname() == "" {
			return false
		}
		
		// Disallow localhost
		if strings.ToLower(parsedURL.Hostname()) == "localhost" {
			return false
		}
		
		// Path must be "/" (root only)
		if parsedURL.Path != "/" && parsedURL.Path != "" {
			return false
		}
		
		return true
	}

	// HTTPS-based schemes – disallow localhost.
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
		// WSS for real-time event-listening lookups.
		parsedURL, err := url.Parse(uri)
		if err != nil {
			return false
		}
		
		if parsedURL.Scheme != "wss" {
			return false
		}
		
		if strings.ToLower(parsedURL.Hostname()) == "localhost" {
			return false
		}
		
		return true
	} else if strings.HasPrefix(uri, "js8c+bsvauth+smf:") {
		// JS8 Call–based advertisement.
		// Expect a query string with parameters.
		queryIndex := strings.Index(uri, "?")
		if queryIndex == -1 {
			return false
		}
		
		queryStr := uri[queryIndex+1:]
		params, err := url.ParseQuery(queryStr)
		if err != nil {
			return false
		}
		
		// Required parameters: lat, long, freq, and radius.
		latStr := params.Get("lat")
		longStr := params.Get("long")
		freqStr := params.Get("freq")
		radiusStr := params.Get("radius")
		
		if latStr == "" || longStr == "" || freqStr == "" || radiusStr == "" {
			return false
		}
		
		// Validate latitude and longitude ranges.
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil || lat < -90 || lat > 90 {
			return false
		}
		
		lon, err := strconv.ParseFloat(longStr, 64)
		if err != nil || lon < -180 || lon > 180 {
			return false
		}
		
		// Validate frequency: extract the first number from the freq string.
		freqRegex := regexp.MustCompile(`(-?\d+(?:\.\d+)?)`)
		freqMatch := freqRegex.FindStringSubmatch(freqStr)
		if len(freqMatch) < 2 {
			return false
		}
		
		freqVal, err := strconv.ParseFloat(freqMatch[1], 64)
		if err != nil || freqVal <= 0 {
			return false
		}
		
		// Validate radius: extract the first number from the radius string.
		radiusMatch := freqRegex.FindStringSubmatch(radiusStr)
		if len(radiusMatch) < 2 {
			return false
		}
		
		radiusVal, err := strconv.ParseFloat(radiusMatch[1], 64)
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
	
	// If none of the known prefixes match, the URI is not advertisable.
	return false
}