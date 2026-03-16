// Package ship implements the SHIP (Service Host Interconnect Protocol) lookup service functionality.
// the BSV overlay LookupService interface.
package ship

import (
	"context"
	"errors"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/transaction"

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
	// storage is the SHIP storage implementation
	storage StorageInterface
}

// Compile-time verification that LookupService implements engine.LookupService
var _ engine.LookupService = (*LookupService)(nil)

// NewLookupService creates a new SHIP lookup service instance.
func NewLookupService(storage StorageInterface) *LookupService {
	return &LookupService{
		storage: storage,
	}
}

// OutputAdmittedByTopic handles an output being admitted by topic.
// This method processes SHIP advertisements encoded in locking scripts using PushDrop format.
// It validates the protocol identifier and stores the SHIP record if valid.
//
// Expected PushDrop fields:
//   - fields[0]: Protocol identifier (must be "SHIP")
//   - fields[1]: Identity key in hex format
//   - fields[2]: Domain string
//   - fields[3]: Topic/service supported
func (s *LookupService) OutputAdmittedByTopic(ctx context.Context, payload *engine.OutputAdmittedByTopic) error {
	fields, err := shared.ParsePushDropOutput(payload, Topic, Identifier)
	if err != nil {
		return err
	}
	if fields == nil {
		return nil // Silently ignore non-matching topics/protocols
	}

	return s.storage.StoreSHIPRecord(ctx, fields.Txid, fields.OutputIndex, fields.IdentityKey, fields.Domain, fields.FourthField)
}

// OutputSpent handles an output being spent.
// This method removes the corresponding SHIP record when the UTXO is spent.
func (s *LookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	return shared.HandleOutputSpent(ctx, payload, Topic, s.storage.DeleteSHIPRecord)
}

// OutputEvicted handles an output being evicted.
// This method removes the corresponding SHIP record when the UTXO is evicted from the mempool.
func (s *LookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	return shared.HandleOutputEvicted(ctx, outpoint, s.storage.DeleteSHIPRecord)
}

// OutputNoLongerRetainedInHistory handles outputs no longer retained in history.
// Called when a Topic Manager decides that historical retention of the specified UTXO is no longer required.
// For SHIP discovery services, this is typically a no-op as they don't maintain historical retention.
func (s *LookupService) OutputNoLongerRetainedInHistory(_ context.Context, _ *transaction.Outpoint, _ string) error {
	// Discovery services don't have the concept of historical retention, so we ignore it
	return nil
}

// OutputBlockHeightUpdated handles block height updates for transactions.
// Called when the block height of a transaction is updated (e.g., when a transaction is included in a block).
// For SHIP discovery services, this is typically a no-op as they don't track block heights.
func (s *LookupService) OutputBlockHeightUpdated(_ context.Context, _ *chainhash.Hash, _ uint32, _ uint64) error {
	// Discovery services don't handle block height updates, so we ignore it
	return nil
}

// Lookup performs a lookup query and returns matching results.
// This method supports both legacy string queries ("findAll") and modern object-based queries.
// It validates query parameters and delegates to the appropriate storage methods.
//
// Supported query formats:
//   - String "findAll": Returns all SHIP records
//   - Object with SHIPQuery fields: Filters by domain, topics, identityKey with pagination
func (s *LookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	return shared.ExecuteLookup(ctx, question, s)
}

// ServiceName returns the SHIP service identifier for the shared lookup executor.
func (s *LookupService) ServiceName() string {
	return Service
}

// FindAll returns all SHIP records with optional pagination (implements shared.QueryExecutor).
func (s *LookupService) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	return s.storage.FindAll(ctx, limit, skip, sortOrder)
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

// GetDocumentation returns the service documentation.
// This method provides comprehensive documentation about the SHIP lookup service,
// including usage examples and best practices.
func (s *LookupService) GetDocumentation() string {
	return LookupDocumentation
}

// GetMetaData returns the service metadata.
// This method provides basic information about the SHIP lookup service
// including name and description.
func (s *LookupService) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SHIP Lookup Service",
		Description: "Provides lookup capabilities for SHIP tokens.",
	}
}
