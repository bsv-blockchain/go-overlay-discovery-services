// Package ship implements the SHIP (Service Host Interconnect Protocol) lookup service functionality.
// the BSV overlay LookupService interface.
package ship

import (
	"context"
	"errors"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/shared"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Constants for SHIP service configuration
const (
	// Topic is the topic manager topic for SHIP advertisements
	Topic = "tm_ship"
	// Service is the lookup service identifier for SHIP
	Service = "ls_ship"
	// Identifier is the protocol identifier expected in PushDrop fields
	Identifier = "SHIP"
)

// Static error variables for err113 compliance
var (
	errQueryDomainInvalid      = errors.New("query.domain must be a string if provided")
	errQueryIdentityKeyInvalid = errors.New("query.identityKey must be a string if provided")
)

// LookupService implements the BSV overlay LookupService interface for SHIP protocol.
// It provides lookup capabilities for SHIP tokens within the overlay network,
// allowing discovery of nodes that host specific topics.
type LookupService struct {
	// BaseLookupService provides shared implementations for common lookup operations
	shared.BaseLookupService

	// storage is the SHIP storage implementation
	storage StorageInterface
}

// Compile-time verification that LookupService implements engine.LookupService
var _ engine.LookupService = (*LookupService)(nil)

// NewLookupService creates a new SHIP lookup service instance.
func NewLookupService(storage StorageInterface) *LookupService {
	doc := LookupDocumentation
	return &LookupService{
		BaseLookupService: shared.NewBaseLookupService(shared.BaseLookupConfig{
			Topic:               Topic,
			ServiceID:           Service,
			Identifier:          Identifier,
			MetaDataName:        "SHIP Lookup Service",
			MetaDataDescription: "Provides lookup capabilities for SHIP tokens.",
			Documentation:       &doc,
			StoreRecord:         storage.StoreSHIPRecord,
			DeleteRecord:        storage.DeleteSHIPRecord,
			FindAll:             storage.FindAll,
		}),
		storage: storage,
	}
}

// Lookup performs a lookup query and returns matching results.
// This method supports both legacy string queries ("findAll") and modern object-based queries.
// It validates query parameters and delegates to the appropriate storage methods.
//
// Supported query formats:
//   - String "findAll": Returns all SHIP records
//   - Object with SHIPQuery fields: Filters by domain, topics, identityKey with pagination
func (s *LookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	return s.BaseLookupService.Lookup(ctx, question, s)
}

// ParseAndExecuteQuery parses a raw query into a SHIPQuery, validates it,
// and executes the appropriate storage call (implements shared.QueryExecutor).
func (s *LookupService) ParseAndExecuteQuery(ctx context.Context, queryInterface interface{}) ([]types.UTXOReference, error) {
	queryObj, err := s.parseQueryObject(queryInterface)
	if err != nil {
		return nil, err
	}

	if queryObj.FindAll != nil && *queryObj.FindAll {
		return s.storage.FindAll(ctx, queryObj.Limit, queryObj.Skip, queryObj.SortOrder)
	}
	return s.storage.FindRecord(ctx, *queryObj)
}

// parseQueryObject parses and validates a query object
func (s *LookupService) parseQueryObject(query interface{}) (*types.SHIPQuery, error) {
	var shipQuery types.SHIPQuery
	if err := shared.ParseQueryJSON(query, &shipQuery); err != nil {
		return nil, err
	}

	// Validate query parameters
	if err := s.validateQuery(&shipQuery); err != nil {
		return nil, err
	}

	return &shipQuery, nil
}

// validateQuery validates the query parameters
func (s *LookupService) validateQuery(query *types.SHIPQuery) error {
	if err := shared.ValidateStringPtrField(query.Domain, errQueryDomainInvalid); err != nil {
		return err
	}
	if err := shared.ValidateStringPtrField(query.IdentityKey, errQueryIdentityKeyInvalid); err != nil {
		return err
	}

	// Validate pagination parameters
	return shared.ValidatePagination(query.Limit, query.Skip, query.SortOrder)
}
