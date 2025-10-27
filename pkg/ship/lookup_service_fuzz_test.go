package ship

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
	f.Add(`{"topics": ["tm_payments", "tm_chat"]}`)
	f.Add(`{"identityKey": "abc123"}`)
	f.Add(`{"limit": 10, "skip": 5}`)
	f.Add(`{"sortOrder": "asc"}`)
	f.Add(`{"sortOrder": "desc"}`)
	f.Add(`{"domain": "example.com", "topics": ["tm_payments"], "limit": 10}`)

	// Seed corpus with invalid/edge-case JSON
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`"findAll"`)
	f.Add(`{"domain": 123}`)
	f.Add(`{"topics": "not_an_array"}`)
	f.Add(`{"topics": [123, 456]}`)
	f.Add(`{"limit": -1}`)
	f.Add(`{"skip": -1}`)
	f.Add(`{"sortOrder": "invalid"}`)
	f.Add(`{"unknown_field": "value"}`)

	// Seed corpus with edge cases
	f.Add(`{"limit": 0}`)
	f.Add(`{"skip": 0}`)
	f.Add(`{"topics": []}`)
	f.Add(`{"domain": ""}`)
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

// FuzzValidateQuerySHIP tests the validateQuery method with random query parameters.
func FuzzValidateQuerySHIP(f *testing.F) {
	// Helper to create string pointer
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }
	sortOrderPtr := func(s types.SortOrder) *types.SortOrder { return &s }

	// Seed corpus with valid queries
	f.Add(true, "example.com", "tm_payments", "key123", 10, 0, "asc")
	f.Add(false, "", "", "", 0, 0, "desc")
	f.Add(false, "test.com", "tm_chat", "", 100, 50, "asc")

	// Seed corpus with invalid queries
	f.Add(false, "", "", "", -1, 0, "asc")    // negative limit
	f.Add(false, "", "", "", 0, -1, "asc")    // negative skip
	f.Add(false, "", "", "", 0, 0, "invalid") // invalid sort order

	// Create a service instance
	service := &LookupService{
		storage: nil,
	}

	f.Fuzz(func(t *testing.T, findAll bool, domain, topic, identityKey string, limit, skip int, sortOrder string) {
		// Build a query object
		query := &types.SHIPQuery{}

		if findAll {
			query.FindAll = boolPtr(findAll)
		}

		if domain != "" {
			query.Domain = strPtr(domain)
		}

		if topic != "" {
			query.Topics = []string{topic}
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
	f.Add(`{"domain": "example.com", "topics": ["tm_payments"]}`)
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

		// Try to unmarshal into SHIPQuery
		var shipQuery types.SHIPQuery
		err = json.Unmarshal(jsonBytes, &shipQuery)

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
		query := &types.SHIPQuery{
			Domain: domainPtr,
		}

		service := &LookupService{storage: nil}

		// Function should not panic on any input
		err := service.validateQuery(query)

		// We don't validate the error, just ensure no panic occurs
		_ = err
	})
}

// FuzzTopicsList tests topics list validation with various array structures.
func FuzzTopicsList(f *testing.F) {
	// Seed corpus with various topic lists
	f.Add("tm_payments", "tm_chat", "tm_identity")
	f.Add("", "", "")
	f.Add("tm_a", "ls_b", "invalid")
	f.Add("tm_"+string(make([]byte, 100)), "", "")

	f.Fuzz(func(t *testing.T, topic1, topic2, topic3 string) {
		// Build topics list
		topics := []string{}
		if topic1 != "" {
			topics = append(topics, topic1)
		}
		if topic2 != "" {
			topics = append(topics, topic2)
		}
		if topic3 != "" {
			topics = append(topics, topic3)
		}

		query := &types.SHIPQuery{
			Topics: topics,
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
		query := &types.SHIPQuery{
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
		query := &types.SHIPQuery{
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
