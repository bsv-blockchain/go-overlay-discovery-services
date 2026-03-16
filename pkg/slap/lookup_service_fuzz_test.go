package slap

import (
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/shared"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// FuzzParseQueryObjectJSON tests the parseQueryObject method with random JSON inputs
// to ensure it handles malformed and edge-case JSON gracefully.
func FuzzParseQueryObjectJSON(f *testing.F) {
	shared.SeedParseQueryFuzz(f)
	// SLAP-specific seeds
	f.Add(`{"service": "ls_identity"}`)
	f.Add(`{"domain": "example.com", "service": "ls_payments", "limit": 10}`)
	f.Add(`{"service": 456}`)
	f.Add(`{"service": ""}`)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, jsonStr string) {
		shared.FuzzParseQueryBody(t, jsonStr, func(qi interface{}) error {
			_, err := service.parseQueryObject(qi)
			return err
		})
	})
}

// FuzzValidateQuerySLAP tests the validateQuery method with random query parameters.
func FuzzValidateQuerySLAP(f *testing.F) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }
	sortOrderPtr := func(s types.SortOrder) *types.SortOrder { return &s }

	f.Add(true, "example.com", "ls_payments", "key123", 10, 0, "asc")
	f.Add(false, "", "", "", 0, 0, "desc")
	f.Add(false, "test.com", "ls_identity", "", 100, 50, "asc")
	f.Add(false, "", "", "", -1, 0, "asc")
	f.Add(false, "", "", "", 0, -1, "asc")
	f.Add(false, "", "", "", 0, 0, "invalid")

	service := &LookupService{storage: nil}

	f.Fuzz(func(t *testing.T, findAll bool, domain, serviceName, identityKey string, limit, skip int, sortOrder string) {
		if len(domain)+len(serviceName)+len(identityKey)+len(sortOrder) > 10000 {
			t.Skip("input too large")
		}
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
		_ = service.validateQuery(query)
	})
}

// FuzzQueryObjectRoundTrip tests JSON marshaling/unmarshaling of query objects.
func FuzzQueryObjectRoundTrip(f *testing.F) {
	f.Add(`{"findAll": true}`)
	f.Add(`{"domain": "example.com", "service": "ls_payments"}`)
	f.Add(`{"limit": 10, "skip": 5, "sortOrder": "asc"}`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
		var slapQuery types.SLAPQuery
		shared.FuzzQueryObjectRoundTripHelper(t, jsonStr, &slapQuery)
	})
}

// FuzzDomainString tests domain string validation edge cases.
func FuzzDomainString(f *testing.F) {
	shared.SeedDomainFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, domain string) {
		shared.FuzzDomainValidationHelper(t, domain, func(d *string) error {
			return service.validateQuery(&types.SLAPQuery{Domain: d})
		})
	})
}

// FuzzServiceNameString tests service name string validation.
func FuzzServiceNameString(f *testing.F) {
	f.Add("ls_payments")
	f.Add("ls_identity")
	f.Add("tm_invalid")
	f.Add("")
	f.Add("invalid")
	f.Add("ls_")
	f.Add("ls_UPPER")
	f.Add("ls_with_numbers123")
	f.Add("ls_special-chars")
	longService := "ls_"
	for i := 0; i < 100; i++ {
		longService += "a"
	}
	f.Add(longService)

	f.Fuzz(func(t *testing.T, serviceName string) {
		if len(serviceName) > 10000 {
			t.Skip("input too large")
		}
		servicePtr := &serviceName
		service := &LookupService{storage: nil}
		_ = service.validateQuery(&types.SLAPQuery{Service: servicePtr})
	})
}

// FuzzIdentityKeyString tests identity key string validation.
func FuzzIdentityKeyString(f *testing.F) {
	shared.SeedIdentityKeyFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, identityKey string) {
		shared.FuzzIdentityKeyValidationHelper(t, identityKey, func(ik *string) error {
			return service.validateQuery(&types.SLAPQuery{IdentityKey: ik})
		})
	})
}

// FuzzPaginationParameters tests pagination parameter validation.
func FuzzPaginationParameters(f *testing.F) {
	shared.SeedPaginationFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, limit, skip int) {
		shared.FuzzPaginationValidationHelper(t, limit, skip, func(l, s *int) error {
			return service.validateQuery(&types.SLAPQuery{Limit: l, Skip: s})
		})
	})
}
