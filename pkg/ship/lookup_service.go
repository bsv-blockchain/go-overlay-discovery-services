// Package ship implements the SHIP (Service Host Interconnect Protocol) lookup service functionality.
// This package provides Go equivalents for the TypeScript SHIPLookupService class, implementing
// the BSV overlay LookupService interface.
package ship

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Constants for SHIP service configuration
const (
	// SHIPTopic is the topic manager topic for SHIP advertisements
	SHIPTopic = "tm_ship"
	// SHIPService is the lookup service identifier for SHIP
	SHIPService = "ls_ship"
	// SHIPIdentifier is the protocol identifier expected in PushDrop fields
	SHIPIdentifier = "SHIP"
)

// SHIPStorageInterface defines the interface for SHIP storage operations
type SHIPStorageInterface interface {
	StoreSHIPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, topic string) error
	DeleteSHIPRecord(ctx context.Context, txid string, outputIndex int) error
	FindRecord(ctx context.Context, query types.SHIPQuery) ([]types.UTXOReference, error)
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	EnsureIndexes(ctx context.Context) error
}

// SHIPLookupService implements the BSV overlay LookupService interface for SHIP protocol.
// It provides lookup capabilities for SHIP tokens within the overlay network,
// allowing discovery of nodes that host specific topics.
type SHIPLookupService struct {
	// storage is the SHIP storage implementation
	storage SHIPStorageInterface
	// pushDropDecoder handles PushDrop locking script decoding
	pushDropDecoder types.PushDropDecoder
	// utils provides utility functions for data conversion
	utils types.Utils
}

// Compile-time verification that SHIPLookupService implements types.LookupService
var _ types.LookupService = (*SHIPLookupService)(nil)

// Compile-time verification that SHIPStorage implements SHIPStorageInterface
var _ SHIPStorageInterface = (*SHIPStorage)(nil)

// NewSHIPLookupService creates a new SHIP lookup service instance.
//
// Parameters:
//   - storage: The SHIP storage implementation for data persistence
//   - pushDropDecoder: The PushDrop decoder for parsing locking scripts
//   - utils: Utility functions for data conversion
//
// Returns:
//   - *SHIPLookupService: A new SHIP lookup service instance
func NewSHIPLookupService(storage SHIPStorageInterface, pushDropDecoder types.PushDropDecoder, utils types.Utils) *SHIPLookupService {
	return &SHIPLookupService{
		storage:         storage,
		pushDropDecoder: pushDropDecoder,
		utils:           utils,
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
//
// Parameters:
//   - payload: The output admission payload containing topic, locking script, and UTXO reference
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SHIPLookupService) OutputAdmittedByTopic(ctx context.Context, payload types.OutputAdmittedByTopic) error {
	// Validate admission mode
	if payload.Mode != types.AdmissionModeLockingScript {
		return fmt.Errorf("invalid payload: expected admission mode 'locking-script', got '%s'", payload.Mode)
	}

	// Only process SHIP topic
	if payload.Topic != SHIPTopic {
		return nil // Silently ignore non-SHIP topics
	}

	// Decode the PushDrop locking script
	result, err := s.pushDropDecoder.Decode(payload.LockingScript)
	if err != nil {
		return fmt.Errorf("failed to decode PushDrop locking script: %w", err)
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return fmt.Errorf("invalid PushDrop result: expected at least 4 fields, got %d", len(result.Fields))
	}

	// Extract and validate fields
	shipIdentifier := s.utils.ToUTF8(result.Fields[0])
	if shipIdentifier != SHIPIdentifier {
		return nil // Silently ignore non-SHIP protocols
	}

	identityKey := s.utils.ToHex(result.Fields[1])
	domain := s.utils.ToUTF8(result.Fields[2])
	topicSupported := s.utils.ToUTF8(result.Fields[3])

	// Store the SHIP record
	return s.storage.StoreSHIPRecord(ctx, payload.Txid, payload.OutputIndex, identityKey, domain, topicSupported)
}

// OutputSpent handles an output being spent.
// This method removes the corresponding SHIP record when the UTXO is spent.
//
// Parameters:
//   - payload: The spent output payload containing topic and UTXO reference
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SHIPLookupService) OutputSpent(ctx context.Context, payload types.OutputSpent) error {
	// Validate spend notification mode
	if payload.Mode != types.SpendNotificationModeNone {
		return fmt.Errorf("invalid payload: expected spend notification mode 'none', got '%s'", payload.Mode)
	}

	// Only process SHIP topic
	if payload.Topic != SHIPTopic {
		return nil // Silently ignore non-SHIP topics
	}

	// Delete the SHIP record
	return s.storage.DeleteSHIPRecord(ctx, payload.Txid, payload.OutputIndex)
}

// OutputEvicted handles an output being evicted.
// This method removes the corresponding SHIP record when the UTXO is evicted from the mempool.
//
// Parameters:
//   - txid: The transaction ID of the evicted output
//   - outputIndex: The index of the evicted output within the transaction
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SHIPLookupService) OutputEvicted(ctx context.Context, txid string, outputIndex int) error {
	// Delete the SHIP record
	return s.storage.DeleteSHIPRecord(ctx, txid, outputIndex)
}

// Lookup performs a lookup query and returns matching results.
// This method supports both legacy string queries ("findAll") and modern object-based queries.
// It validates query parameters and delegates to the appropriate storage methods.
//
// Supported query formats:
//   - String "findAll": Returns all SHIP records
//   - Object with SHIPQuery fields: Filters by domain, topics, identityKey with pagination
//
// Parameters:
//   - question: The lookup question containing service identifier and query parameters
//
// Returns:
//   - types.LookupFormula: Matching UTXO references
//   - error: An error if the query fails or is invalid, nil otherwise
func (s *SHIPLookupService) Lookup(ctx context.Context, question types.LookupQuestion) (types.LookupFormula, error) {
	// Validate required fields
	if question.Query == nil {
		return nil, fmt.Errorf("a valid query must be provided")
	}

	if question.Service != SHIPService {
		return nil, fmt.Errorf("lookup service not supported: expected '%s', got '%s'", SHIPService, question.Service)
	}

	// Handle legacy "findAll" string query
	if queryStr, ok := question.Query.(string); ok {
		if queryStr == "findAll" {
			return s.storage.FindAll(ctx, nil, nil, nil)
		}
		return nil, fmt.Errorf("invalid string query: only 'findAll' is supported, got '%s'", queryStr)
	}

	// Handle object-based query
	queryObj, err := s.parseQueryObject(question.Query)
	if err != nil {
		return nil, fmt.Errorf("invalid query format: %w", err)
	}

	// Handle findAll with pagination
	if queryObj.FindAll != nil && *queryObj.FindAll {
		return s.storage.FindAll(ctx, queryObj.Limit, queryObj.Skip, queryObj.SortOrder)
	}

	// Handle specific query with filters
	return s.storage.FindRecord(ctx, *queryObj)
}

// parseQueryObject parses and validates a query object
func (s *SHIPLookupService) parseQueryObject(query interface{}) (*types.SHIPQuery, error) {
	// Convert to JSON and back to ensure proper type mapping
	jsonBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query object: %w", err)
	}

	var shipQuery types.SHIPQuery
	if err := json.Unmarshal(jsonBytes, &shipQuery); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query object: %w", err)
	}

	// Validate query parameters
	if err := s.validateQuery(&shipQuery); err != nil {
		return nil, err
	}

	return &shipQuery, nil
}

// validateQuery validates the query parameters
func (s *SHIPLookupService) validateQuery(query *types.SHIPQuery) error {
	// Validate domain parameter
	if query.Domain != nil {
		if reflect.TypeOf(query.Domain).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.Domain).Elem().Kind() != reflect.String {
			return fmt.Errorf("query.domain must be a string if provided")
		}
	}

	// Validate topics parameter
	if query.Topics != nil {
		if reflect.TypeOf(query.Topics).Kind() != reflect.Slice {
			return fmt.Errorf("query.topics must be an array of strings if provided")
		}
		for i, topic := range query.Topics {
			if reflect.TypeOf(topic).Kind() != reflect.String {
				return fmt.Errorf("query.topics[%d] must be a string", i)
			}
		}
	}

	// Validate identityKey parameter
	if query.IdentityKey != nil {
		if reflect.TypeOf(query.IdentityKey).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.IdentityKey).Elem().Kind() != reflect.String {
			return fmt.Errorf("query.identityKey must be a string if provided")
		}
	}

	// Validate pagination parameters
	if query.Limit != nil {
		if *query.Limit < 0 {
			return fmt.Errorf("query.limit must be a positive number if provided")
		}
	}

	if query.Skip != nil {
		if *query.Skip < 0 {
			return fmt.Errorf("query.skip must be a non-negative number if provided")
		}
	}

	// Validate sort order parameter
	if query.SortOrder != nil {
		if *query.SortOrder != types.SortOrderAsc && *query.SortOrder != types.SortOrderDesc {
			return fmt.Errorf("query.sortOrder must be 'asc' or 'desc' if provided")
		}
	}

	return nil
}

// GetDocumentation returns the service documentation.
// This method provides comprehensive documentation about the SHIP lookup service,
// including usage examples and best practices.
//
// Returns:
//   - string: The service documentation in markdown format
//   - error: Always nil (no errors expected)
func (s *SHIPLookupService) GetDocumentation() (string, error) {
	return SHIPDocumentation, nil
}

// GetMetaData returns the service metadata.
// This method provides basic information about the SHIP lookup service
// including name and description.
//
// Returns:
//   - types.MetaData: The service metadata
//   - error: Always nil (no errors expected)
func (s *SHIPLookupService) GetMetaData() (types.MetaData, error) {
	return types.MetaData{
		Name:             "SHIP Lookup Service",
		ShortDescription: "Provides lookup capabilities for SHIP tokens.",
	}, nil
}
