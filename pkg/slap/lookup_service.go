// Package slap implements the SLAP (Service Lookup Availability Protocol) lookup service functionality.
// The BSV overlay LookupService interface.
package slap

import (
	"context"
	"errors"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/shared"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Constants for SLAP service configuration
const (
	// Topic is the topic manager topic for SLAP advertisements
	Topic = "tm_slap"
	// Service is the lookup service identifier for SLAP
	Service = "ls_slap"
	// Identifier is the protocol identifier expected in PushDrop fields
	Identifier = "SLAP"
)

// Static error variables for err113 compliance
var (
	errQueryDomainInvalid      = errors.New("query.domain must be a string if provided")
	errQueryTopicsInvalid      = errors.New("query.topics must be an array of strings if provided")
	errQueryIdentityKeyInvalid = errors.New("query.identityKey must be a string if provided")
)

// LookupService implements the BSV overlay LookupService interface for SLAP protocol.
// It provides lookup capabilities for SLAP tokens within the overlay network,
// allowing discovery of nodes that offer specific services.
type LookupService struct {
	// BaseLookupService provides shared implementations for common lookup operations
	shared.BaseLookupService

	// storage is the SLAP storage implementation
	storage StorageInterface
}

// Compile-time verification that LookupService implements engine.LookupService
var _ engine.LookupService = (*LookupService)(nil)

// NewLookupService creates a new SLAP lookup service instance.
func NewLookupService(storage StorageInterface) *LookupService {
	doc := LookupDocumentation
	return &LookupService{
		BaseLookupService: shared.NewBaseLookupService(shared.BaseLookupConfig{
			Topic:               Topic,
			ServiceID:           Service,
			Identifier:          Identifier,
			MetaDataName:        "SLAP Lookup Service",
			MetaDataDescription: "Provides lookup capabilities for SLAP tokens.",
			Documentation:       &doc,
			StoreRecord:         storage.StoreSLAPRecord,
			DeleteRecord:        storage.DeleteSLAPRecord,
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
//   - String "findAll": Returns all SLAP records
//   - Object with SLAPQuery fields: Filters by domain, service, identityKey with pagination
func (s *LookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	return s.BaseLookupService.Lookup(ctx, question, s)
}

// ParseAndExecuteQuery parses a raw query into a SLAPQuery, validates it,
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
func (s *LookupService) parseQueryObject(query interface{}) (*types.SLAPQuery, error) {
	var slapQuery types.SLAPQuery
	if err := shared.ParseQueryJSON(query, &slapQuery); err != nil {
		return nil, err
	}

	// Validate query parameters
	if err := s.validateQuery(&slapQuery); err != nil {
		return nil, err
	}

	return &slapQuery, nil
}

// validateQuery validates the query parameters
func (s *LookupService) validateQuery(query *types.SLAPQuery) error {
	if err := shared.ValidateStringPtrField(query.Domain, errQueryDomainInvalid); err != nil {
		return err
	}
	if err := shared.ValidateStringPtrField(query.Service, errQueryTopicsInvalid); err != nil {
		return err
	}
	if err := shared.ValidateStringPtrField(query.IdentityKey, errQueryIdentityKeyInvalid); err != nil {
		return err
	}

	// Validate pagination parameters
	return shared.ValidatePagination(query.Limit, query.Skip, query.SortOrder)
}
