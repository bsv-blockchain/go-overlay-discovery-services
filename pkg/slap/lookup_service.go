// Package slap implements the SLAP (Service Lookup Availability Protocol) lookup service functionality.
// The BSV overlay LookupService interface.
package slap

import (
	"context"
	"errors"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/transaction"

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
	// DiscoveryNoOps provides no-op implementations for methods not relevant to discovery services
	shared.DiscoveryNoOps

	// storage is the SLAP storage implementation
	storage StorageInterface
}

// Compile-time verification that LookupService implements engine.LookupService
var _ engine.LookupService = (*LookupService)(nil)

// NewLookupService creates a new SLAP lookup service instance.
func NewLookupService(storage StorageInterface) *LookupService {
	return &LookupService{
		storage: storage,
	}
}

// OutputAdmittedByTopic handles an output being admitted by topic.
// This method processes SLAP advertisements encoded in locking scripts using PushDrop format.
// It validates the protocol identifier and stores the SLAP record if valid.
//
// Expected PushDrop fields:
//   - fields[0]: Protocol identifier (must be "SLAP")
//   - fields[1]: Identity key in hex format
//   - fields[2]: Domain string
//   - fields[3]: Service name supported
func (s *LookupService) OutputAdmittedByTopic(ctx context.Context, payload *engine.OutputAdmittedByTopic) error {
	fields, err := shared.ParsePushDropOutput(payload, Topic, Identifier)
	if err != nil {
		return err
	}
	if fields == nil {
		return nil // Silently ignore non-matching topics/protocols
	}

	return s.storage.StoreSLAPRecord(ctx, fields.Txid, fields.OutputIndex, fields.IdentityKey, fields.Domain, fields.FourthField)
}

// OutputSpent handles an output being spent.
// This method removes the corresponding SLAP record when the UTXO is spent.
func (s *LookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	return shared.HandleOutputSpent(ctx, payload, Topic, s.storage.DeleteSLAPRecord)
}

// OutputEvicted handles an output being evicted.
// This method removes the corresponding SLAP record when the UTXO is evicted from the mempool.
func (s *LookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	return shared.HandleOutputEvicted(ctx, outpoint, s.storage.DeleteSLAPRecord)
}

// Lookup performs a lookup query and returns matching results.
// This method supports both legacy string queries ("findAll") and modern object-based queries.
// It validates query parameters and delegates to the appropriate storage methods.
//
// Supported query formats:
//   - String "findAll": Returns all SLAP records
//   - Object with SLAPQuery fields: Filters by domain, service, identityKey with pagination
func (s *LookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	return shared.ExecuteLookup(ctx, question, s)
}

// ServiceName returns the SLAP service identifier for the shared lookup executor.
func (s *LookupService) ServiceName() string {
	return Service
}

// FindAll returns all SLAP records with optional pagination (implements shared.QueryExecutor).
func (s *LookupService) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	return s.storage.FindAll(ctx, limit, skip, sortOrder)
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

// GetDocumentation returns the service documentation.
// This method provides comprehensive documentation about the SLAP lookup service,
// including usage examples and best practices.
func (s *LookupService) GetDocumentation() string {
	return LookupDocumentation
}

// GetMetaData returns the service metadata.
// This method provides basic information about the SLAP lookup service
// including name and description.
func (s *LookupService) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SLAP Lookup Service",
		Description: "Provides lookup capabilities for SLAP tokens.",
	}
}
