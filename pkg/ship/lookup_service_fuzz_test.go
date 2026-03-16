package ship

import (
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/shared"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// FuzzParseQueryObjectJSON tests the parseQueryObject method with random JSON inputs
// to ensure it handles malformed and edge-case JSON gracefully.
func FuzzParseQueryObjectJSON(f *testing.F) {
	shared.SeedParseQueryFuzz(f)
	// SHIP-specific seeds
	f.Add(`{"topics": ["tm_payments", "tm_chat"]}`)
	f.Add(`{"domain": "example.com", "topics": ["tm_payments"], "limit": 10}`)
	f.Add(`{"topics": "not_an_array"}`)
	f.Add(`{"topics": [123, 456]}`)
	f.Add(`{"topics": []}`)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, jsonStr string) {
		shared.FuzzParseQueryBody(t, jsonStr, func(qi interface{}) error {
			_, err := service.parseQueryObject(qi)
			return err
		})
	})
}

// FuzzValidateQuerySHIP tests the validateQuery method with random query parameters.
func FuzzValidateQuerySHIP(f *testing.F) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	boolPtr := func(b bool) *bool { return &b }
	sortOrderPtr := func(s types.SortOrder) *types.SortOrder { return &s }

	f.Add(true, "example.com", "tm_payments", "key123", 10, 0, "asc")
	f.Add(false, "", "", "", 0, 0, "desc")
	f.Add(false, "test.com", "tm_chat", "", 100, 50, "asc")
	f.Add(false, "", "", "", -1, 0, "asc")
	f.Add(false, "", "", "", 0, -1, "asc")
	f.Add(false, "", "", "", 0, 0, "invalid")

	service := &LookupService{storage: nil}

	f.Fuzz(func(t *testing.T, findAll bool, domain, topic, identityKey string, limit, skip int, sortOrder string) {
		if len(domain)+len(topic)+len(identityKey)+len(sortOrder) > 10000 {
			t.Skip("input too large")
		}
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
		_ = service.validateQuery(query)
	})
}

// FuzzQueryObjectRoundTrip tests JSON marshaling/unmarshaling of query objects.
func FuzzQueryObjectRoundTrip(f *testing.F) {
	f.Add(`{"findAll": true}`)
	f.Add(`{"domain": "example.com", "topics": ["tm_payments"]}`)
	f.Add(`{"limit": 10, "skip": 5, "sortOrder": "asc"}`)

	f.Fuzz(func(t *testing.T, jsonStr string) {
		var shipQuery types.SHIPQuery
		shared.FuzzQueryObjectRoundTripHelper(t, jsonStr, &shipQuery)
	})
}

// FuzzDomainString tests domain string validation edge cases.
func FuzzDomainString(f *testing.F) {
	shared.SeedDomainFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, domain string) {
		shared.FuzzDomainValidationHelper(t, domain, func(d *string) error {
			return service.validateQuery(&types.SHIPQuery{Domain: d})
		})
	})
}

// FuzzTopicsList tests topics list validation with various array structures.
func FuzzTopicsList(f *testing.F) {
	f.Add("tm_payments", "tm_chat", "tm_identity")
	f.Add("", "", "")
	f.Add("tm_a", "ls_b", "invalid")
	f.Add("tm_"+string(make([]byte, 100)), "", "")

	f.Fuzz(func(t *testing.T, topic1, topic2, topic3 string) {
		if len(topic1)+len(topic2)+len(topic3) > 10000 {
			t.Skip("input too large")
		}
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

		service := &LookupService{storage: nil}
		_ = service.validateQuery(&types.SHIPQuery{Topics: topics})
	})
}

// FuzzIdentityKeyString tests identity key string validation.
func FuzzIdentityKeyString(f *testing.F) {
	shared.SeedIdentityKeyFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, identityKey string) {
		shared.FuzzIdentityKeyValidationHelper(t, identityKey, func(ik *string) error {
			return service.validateQuery(&types.SHIPQuery{IdentityKey: ik})
		})
	})
}

// FuzzPaginationParameters tests pagination parameter validation.
func FuzzPaginationParameters(f *testing.F) {
	shared.SeedPaginationFuzz(f)

	service := &LookupService{storage: nil}
	f.Fuzz(func(t *testing.T, limit, skip int) {
		shared.FuzzPaginationValidationHelper(t, limit, skip, func(l, s *int) error {
			return service.validateQuery(&types.SHIPQuery{Limit: l, Skip: s})
		})
	})
}
