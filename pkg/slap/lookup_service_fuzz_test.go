package slap

import (
	"encoding/json"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// FuzzParseQueryObjectJSON tests the parseQueryObject method with random JSON inputs
// to ensure it handles malformed and edge-case JSON gracefully.
func FuzzParseQueryObjectJSON(f *testing.F) {
	// Seed corpus with valid query JSON examples
	f.Add(`{"findAll": true}`)
	f.Add(`{"domain": "example.com"}`)
	f.Add(`{"service": "ls_identity"}`)
	f.Add(`{"identityKey": "abc123"}`)
	f.Add(`{"limit": 10, "skip": 5}`)
	f.Add(`{"sortOrder": "asc"}`)
	f.Add(`{"sortOrder": "desc"}`)
	f.Add(`{"domain": "example.com", "service": "ls_payments", "limit": 10}`)

	// Seed corpus with invalid/edge-case JSON
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`"findAll"`)
	f.Add(`{"domain": 123}`)
	f.Add(`{"service": 456}`)
	f.Add(`{"limit": -1}`)
	f.Add(`{"skip": -1}`)
	f.Add(`{"sortOrder": "invalid"}`)
	f.Add(`{"unknown_field": "value"}`)

	// Seed corpus with edge cases
	f.Add(`{"limit": 0}`)
	f.Add(`{"skip": 0}`)
	f.Add(`{"domain": ""}`)
	f.Add(`{"service": ""}`)
	f.Add(`{"findAll": false}`)
	f.Add(`[1, 2, 3]`)
	f.Add(`true`)
	f.Add(`123`)

	// Create a service instance with a mock storage
	service := &LookupService{
		storage: nil, // We don't need actual storage for this test
	}

	f.Fuzz(func(t *testing.T, jsonStr string) {
		// First, try to unmarshal to ensure it's valid JSON
		var queryInterface interface{}
		err := json.Unmarshal([]byte(jsonStr), &queryInterface)
		if err != nil {
			// Invalid JSON should be rejected, but shouldn't panic
			return
		}

		// Function should not panic on any input
		_, err = service.parseQueryObject(queryInterface)

		// We don't validate the result or error, just ensure no panic occurs
		// Errors are expected for invalid query structures
		_ = err
	})
}

// FuzzValidateQuerySLAP tests the validateQuery method with random query parameters.
func FuzzValidateQuerySLAP(f *testing.F) {
	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }
	sortOrderPtr := func(s types.SortOrder) *types.SortOrder { return &s }

	// Seed corpus with valid queries
	f.Add(true, "example.com", "ls_payments", "key123", 10, 0, "asc")
	f.Add(false, "", "", "", 0, 0, "desc")
	f.Add(false, "test.com", "ls_identity", "", 100, 50, "asc")

	// Seed corpus with invalid queries
	f.Add(false, "", "", "", -1, 0, "asc")    // negative limit
	f.Add(false, "", "", "", 0, -1, "asc")    // negative skip
	f.Add(false, "", "", "", 0, 0, "invalid") // invalid sort order

	// Create a service instance
	service := &LookupService{
		storage: nil,
	}

	f.Fuzz(func(t *testing.T, findAll bool, domain, serviceName, identityKey string, limit, skip int, sortOrder string) {
		// Build a query object
		query := &types.SLAPQuery{}

		if findAll {
			query.FindAll = boolPtr(findAll)
		}

		if domain != "" {
			query.Domain = strPtr(domain)
		}

		if serviceName != "" {
			query.Service = strPtr(serviceName)
		}

		if identityKey != "" {
			query.IdentityKey = strPtr(identityKey)
		}

		if limit != 0 {
			query.Limit = intPtr(limit)
		}

		if skip != 0 {
			query.Skip = intPtr(skip)
		}

		if sortOrder != "" {
			so := types.SortOrder(sortOrder)
			query.SortOrder = sortOrderPtr(so)
		}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We don't validate the error, just ensure no panic occurs
		_ = err
	})
}

// FuzzQueryObjectRoundTrip tests JSON marshaling/unmarshaling of query objects.
func FuzzQueryObjectRoundTrip(f *testing.F) {
	// Seed corpus with various JSON structures
	f.Add(`{"findAll": true}`)
	f.Add(`{"domain": "example.com", "service": "ls_payments"}`)
	f.Add(`{"limit": 10, "skip": 5, "sortOrder": "asc"}`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
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

		// Try to unmarshal into SLAPQuery
		var slapQuery types.SLAPQuery
		err = json.Unmarshal(jsonBytes, &slapQuery)

		// Function should not panic, errors are acceptable
		_ = err
	})
}

// FuzzDomainString tests domain string validation edge cases.
func FuzzDomainString(f *testing.F) {
	// Seed corpus with various domain formats
	f.Add("example.com")
	f.Add("sub.example.com")
	f.Add("example.com:8080")
	f.Add("192.168.1.1")
	f.Add("localhost")
	f.Add("[::1]")
	f.Add("")
	f.Add(".")
	f.Add(".example.com")
	f.Add("example.com.")
	f.Add("ex ample.com")
	f.Add("example..com")
	f.Add("example$.com")
	// Test very long domain name
	longDomain := ""
	for i := 0; i < 255; i++ {
		longDomain += "a"
	}
	f.Add(longDomain)

	f.Fuzz(func(t *testing.T, domain string) {
		// Create a query with the fuzzed domain
		domainPtr := &domain
		query := &types.SLAPQuery{
			Domain: domainPtr,
		}

		service := &LookupService{storage: nil}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We don't validate the error, just ensure no panic occurs
		_ = err
	})
}

// FuzzServiceNameString tests service name string validation.
func FuzzServiceNameString(f *testing.F) {
	// Seed corpus with various service name formats
	f.Add("ls_payments")
	f.Add("ls_identity")
	f.Add("tm_invalid") // topic manager prefix instead of lookup service
	f.Add("")
	f.Add("invalid")
	f.Add("ls_")
	f.Add("ls_UPPER")
	f.Add("ls_with_numbers123")
	f.Add("ls_special-chars")
	// Test very long service name
	longService := "ls_"
	for i := 0; i < 100; i++ {
		longService += "a"
	}
	f.Add(longService)

	f.Fuzz(func(t *testing.T, serviceName string) {
		// Create a query with the fuzzed service name
		servicePtr := &serviceName
		query := &types.SLAPQuery{
			Service: servicePtr,
		}

		service := &LookupService{storage: nil}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We don't validate the error, just ensure no panic occurs
		_ = err
	})
}

// FuzzIdentityKeyString tests identity key string validation.
func FuzzIdentityKeyString(f *testing.F) {
	// Seed corpus with various identity key formats
	f.Add("0123456789abcdef")
	f.Add("deadbeef")
	f.Add("")
	f.Add("not_hex")
	f.Add("0x1234")
	f.Add(string(make([]byte, 1000)))

	f.Fuzz(func(t *testing.T, identityKey string) {
		// Create a query with the fuzzed identity key
		identityKeyPtr := &identityKey
		query := &types.SLAPQuery{
			IdentityKey: identityKeyPtr,
		}

		service := &LookupService{storage: nil}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We don't validate the error, just ensure no panic occurs
		_ = err
	})
}

// FuzzPaginationParameters tests pagination parameter validation.
func FuzzPaginationParameters(f *testing.F) {
	// Seed corpus with various pagination values
	f.Add(0, 0)
	f.Add(10, 5)
	f.Add(100, 0)
	f.Add(1, 1000000)
	f.Add(-1, 0)
	f.Add(0, -1)
	f.Add(-100, -100)
	f.Add(2147483647, 2147483647) // Max int32

	f.Fuzz(func(t *testing.T, limit, skip int) {
		// Create a query with the fuzzed pagination parameters
		limitPtr := &limit
		skipPtr := &skip
		query := &types.SLAPQuery{
			Limit: limitPtr,
			Skip:  skipPtr,
		}

		service := &LookupService{storage: nil}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We expect errors for negative values, but no panics
		_ = err
	})
}
