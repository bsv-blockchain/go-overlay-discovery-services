package shared

import (
	"encoding/json"
	"testing"
)

// SeedDomainFuzz adds the standard domain seed corpus to a fuzz target.
func SeedDomainFuzz(f *testing.F) {
	f.Helper()
	f.Add("example.com")
	f.Add("sub.example.com")
	f.Add("example.com:8080")
	f.Add("198.51.100.1")
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
}

// SeedIdentityKeyFuzz adds the standard identity key seed corpus to a fuzz target.
func SeedIdentityKeyFuzz(f *testing.F) {
	f.Helper()
	f.Add("0123456789abcdef")
	f.Add("deadbeef")
	f.Add("")
	f.Add("not_hex")
	f.Add("0x1234")
	f.Add(string(make([]byte, 1000)))
}

// SeedPaginationFuzz adds the standard pagination seed corpus to a fuzz target.
func SeedPaginationFuzz(f *testing.F) {
	f.Helper()
	f.Add(0, 0)
	f.Add(10, 5)
	f.Add(100, 0)
	f.Add(1, 1000000)
	f.Add(-1, 0)
	f.Add(0, -1)
	f.Add(-100, -100)
	f.Add(2147483647, 2147483647) // Max int32
}

// SeedParseQueryFuzz adds the standard JSON parse query seed corpus to a fuzz target.
func SeedParseQueryFuzz(f *testing.F) {
	f.Helper()
	// Valid query JSON examples
	f.Add(`{"findAll": true}`)
	f.Add(`{"domain": "example.com"}`)
	f.Add(`{"identityKey": "abc123"}`)
	f.Add(`{"limit": 10, "skip": 5}`)
	f.Add(`{"sortOrder": "asc"}`)
	f.Add(`{"sortOrder": "desc"}`)

	// Invalid/edge-case JSON
	f.Add(`{}`)
	f.Add(`null`)
	f.Add(`"findAll"`)
	f.Add(`{"domain": 123}`)
	f.Add(`{"limit": -1}`)
	f.Add(`{"skip": -1}`)
	f.Add(`{"sortOrder": "invalid"}`)
	f.Add(`{"unknown_field": "value"}`)

	// Edge cases
	f.Add(`{"limit": 0}`)
	f.Add(`{"skip": 0}`)
	f.Add(`{"domain": ""}`)
	f.Add(`{"findAll": false}`)
	f.Add(`[1, 2, 3]`)
	f.Add(`true`)
	f.Add(`123`)
}

// FuzzParseQueryBody runs the common fuzz body for parseQueryObject tests.
// The parseFn should call the service's parseQueryObject method.
func FuzzParseQueryBody(t *testing.T, jsonStr string, parseFn func(interface{}) error) {
	t.Helper()
	if len(jsonStr) > 10000 {
		t.Skip(skipInputTooLarge)
	}
	var queryInterface interface{}
	if err := json.Unmarshal([]byte(jsonStr), &queryInterface); err != nil {
		return
	}
	// Function should not panic on any input
	_ = parseFn(queryInterface)
}
