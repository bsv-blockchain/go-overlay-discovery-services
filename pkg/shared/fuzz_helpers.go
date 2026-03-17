package shared

import (
	"encoding/json"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// skipInputTooLarge is the skip message used when fuzz input exceeds the size threshold.
const skipInputTooLarge = "input too large"

// FuzzQueryObjectRoundTripHelper implements the common FuzzQueryObjectRoundTrip logic.
// It unmarshals JSON into an interface, marshals it back, then unmarshals into the target type.
// The target parameter should be a pointer to the query type (e.g. *SHIPQuery or *SLAPQuery).
func FuzzQueryObjectRoundTripHelper(t *testing.T, jsonStr string, target interface{}) {
	t.Helper()
	if len(jsonStr) > 10000 {
		t.Skip(skipInputTooLarge)
	}
	// Try to unmarshal into interface
	var queryInterface interface{}
	err := json.Unmarshal([]byte(jsonStr), &queryInterface)
	if err != nil {
		return
	}

	// Try to marshal back to JSON
	jsonBytes, err := json.Marshal(queryInterface)
	if err != nil {
		t.Errorf("Failed to marshal query interface: %v", err)
		return
	}

	// Try to unmarshal into the target query type
	err = json.Unmarshal(jsonBytes, target)

	// Function should not panic, errors are acceptable
	_ = err
}

// FuzzDomainValidationHelper implements the common domain validation fuzz logic.
// The validateFn receives a domain pointer and should call the appropriate validateQuery method.
func FuzzDomainValidationHelper(t *testing.T, domain string, validateFn func(*string) error) {
	t.Helper()
	if len(domain) > 10000 {
		t.Skip(skipInputTooLarge)
	}
	// Function should not panic on any input
	err := validateFn(&domain)
	_ = err
}

// FuzzIdentityKeyValidationHelper implements the common identity key validation fuzz logic.
// The validateFn receives an identityKey pointer and should call the appropriate validateQuery method.
func FuzzIdentityKeyValidationHelper(t *testing.T, identityKey string, validateFn func(*string) error) {
	t.Helper()
	if len(identityKey) > 10000 {
		t.Skip(skipInputTooLarge)
	}
	// Function should not panic on any input
	err := validateFn(&identityKey)
	_ = err
}

// FuzzPaginationValidationHelper implements the common pagination validation fuzz logic.
// The validateFn receives limit and skip pointers and should call the appropriate validateQuery method.
func FuzzPaginationValidationHelper(t *testing.T, limit, skip int, validateFn func(*int, *int) error) {
	t.Helper()
	// Function should not panic on any input
	err := validateFn(&limit, &skip)
	_ = err
}

// StrPtr returns a pointer to the given string. Shared helper for fuzz tests.
func StrPtr(s string) *string { return &s }

// IntPtr returns a pointer to the given int. Shared helper for fuzz tests.
func IntPtr(i int) *int { return &i }

// BoolPtr returns a pointer to the given bool. Shared helper for fuzz tests.
func BoolPtr(b bool) *bool { return &b }

// SortOrderPtr returns a pointer to the given SortOrder. Shared helper for fuzz tests.
func SortOrderPtr(s types.SortOrder) *types.SortOrder { return &s }
