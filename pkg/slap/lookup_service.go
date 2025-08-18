// Package slap implements the SLAP (Service Lookup Availability Protocol) lookup service functionality.
// This package provides Go equivalents for the TypeScript SLAPLookupService class, implementing
// the BSV overlay LookupService interface.
package slap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
)

// Constants for SLAP service configuration
const (
	// SLAPTopic is the topic manager topic for SLAP advertisements
	SLAPTopic = "tm_slap"
	// SLAPService is the lookup service identifier for SLAP
	SLAPService = "ls_slap"
	// SLAPIdentifier is the protocol identifier expected in PushDrop fields
	SLAPIdentifier = "SLAP"
)

// SLAPStorageInterface defines the interface for SLAP storage operations
type SLAPStorageInterface interface {
	StoreSLAPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, service string) error
	DeleteSLAPRecord(ctx context.Context, txid string, outputIndex int) error
	FindRecord(ctx context.Context, query types.SLAPQuery) ([]types.UTXOReference, error)
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	EnsureIndexes(ctx context.Context) error
}

// SLAPLookupService implements the BSV overlay LookupService interface for SLAP protocol.
// It provides lookup capabilities for SLAP tokens within the overlay network,
// allowing discovery of nodes that offer specific services.
type SLAPLookupService struct {
	// storage is the SLAP storage implementation
	storage SLAPStorageInterface
}

// Compile-time verification that SLAPLookupService implements types.LookupService
var _ types.LookupService = (*SLAPLookupService)(nil)

// Compile-time verification that SLAPStorage implements SLAPStorageInterface
var _ SLAPStorageInterface = (*SLAPStorage)(nil)

// NewSLAPLookupService creates a new SLAP lookup service instance.
//
// Parameters:
//   - storage: The SLAP storage implementation for data persistence
//
// Returns:
//   - *SLAPLookupService: A new SLAP lookup service instance
func NewSLAPLookupService(storage SLAPStorageInterface) *SLAPLookupService {
	return &SLAPLookupService{
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
//
// Parameters:
//   - payload: The output admission payload containing topic, locking script, and UTXO reference
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SLAPLookupService) OutputAdmittedByTopic(ctx context.Context, payload types.OutputAdmittedByTopic) error {
	// Validate admission mode
	if payload.Mode != types.AdmissionModeLockingScript {
		return fmt.Errorf("invalid payload: expected admission mode 'locking-script', got '%s'", payload.Mode)
	}

	// Only process SLAP topic
	if payload.Topic != SLAPTopic {
		return nil // Silently ignore non-SLAP topics
	}

	// Create script from hex string
	scriptObj, err := script.NewFromHex(payload.LockingScript)
	if err != nil {
		return fmt.Errorf("failed to create script from hex: %w", err)
	}

	// Decode the PushDrop locking script
	result := pushdrop.Decode(scriptObj)
	if result == nil {
		return fmt.Errorf("failed to decode PushDrop locking script")
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return fmt.Errorf("invalid PushDrop result: expected at least 4 fields, got %d", len(result.Fields))
	}

	// Extract and validate fields
	slapIdentifier := string(result.Fields[0])
	if slapIdentifier != SLAPIdentifier {
		return nil // Silently ignore non-SLAP protocols
	}

	identityKey := hex.EncodeToString(result.Fields[1])
	domain := string(result.Fields[2])
	serviceSupported := string(result.Fields[3])

	// Store the SLAP record
	return s.storage.StoreSLAPRecord(ctx, payload.Txid, payload.OutputIndex, identityKey, domain, serviceSupported)
}

// OutputSpent handles an output being spent.
// This method removes the corresponding SLAP record when the UTXO is spent.
//
// Parameters:
//   - payload: The spent output payload containing topic and UTXO reference
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SLAPLookupService) OutputSpent(ctx context.Context, payload types.OutputSpent) error {
	// Validate spend notification mode
	if payload.Mode != types.SpendNotificationModeNone {
		return fmt.Errorf("invalid payload: expected spend notification mode 'none', got '%s'", payload.Mode)
	}

	// Only process SLAP topic
	if payload.Topic != SLAPTopic {
		return nil // Silently ignore non-SLAP topics
	}

	// Delete the SLAP record
	return s.storage.DeleteSLAPRecord(ctx, payload.Txid, payload.OutputIndex)
}

// OutputEvicted handles an output being evicted.
// This method removes the corresponding SLAP record when the UTXO is evicted from the mempool.
//
// Parameters:
//   - txid: The transaction ID of the evicted output
//   - outputIndex: The index of the evicted output within the transaction
//
// Returns:
//   - error: An error if processing fails, nil otherwise
func (s *SLAPLookupService) OutputEvicted(ctx context.Context, txid string, outputIndex int) error {
	// Delete the SLAP record
	return s.storage.DeleteSLAPRecord(ctx, txid, outputIndex)
}

// Lookup performs a lookup query and returns matching results.
// This method supports both legacy string queries ("findAll") and modern object-based queries.
// It validates query parameters and delegates to the appropriate storage methods.
//
// Supported query formats:
//   - String "findAll": Returns all SLAP records
//   - Object with SLAPQuery fields: Filters by domain, service, identityKey with pagination
//
// Parameters:
//   - question: The lookup question containing service identifier and query parameters
//
// Returns:
//   - types.LookupFormula: Matching UTXO references
//   - error: An error if the query fails or is invalid, nil otherwise
func (s *SLAPLookupService) Lookup(ctx context.Context, question types.LookupQuestion) (types.LookupFormula, error) {
	// Validate required fields
	if question.Query == nil {
		return nil, fmt.Errorf("a valid query must be provided")
	}

	if question.Service != SLAPService {
		return nil, fmt.Errorf("lookup service not supported: expected '%s', got '%s'", SLAPService, question.Service)
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
func (s *SLAPLookupService) parseQueryObject(query interface{}) (*types.SLAPQuery, error) {
	// Convert to JSON and back to ensure proper type mapping
	jsonBytes, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query object: %w", err)
	}

	var slapQuery types.SLAPQuery
	if err := json.Unmarshal(jsonBytes, &slapQuery); err != nil {
		return nil, fmt.Errorf("failed to unmarshal query object: %w", err)
	}

	// Validate query parameters
	if err := s.validateQuery(&slapQuery); err != nil {
		return nil, err
	}

	return &slapQuery, nil
}

// validateQuery validates the query parameters
func (s *SLAPLookupService) validateQuery(query *types.SLAPQuery) error {
	// Validate domain parameter
	if query.Domain != nil {
		if reflect.TypeOf(query.Domain).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.Domain).Elem().Kind() != reflect.String {
			return fmt.Errorf("query.domain must be a string if provided")
		}
	}

	// Validate service parameter
	if query.Service != nil {
		if reflect.TypeOf(query.Service).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.Service).Elem().Kind() != reflect.String {
			return fmt.Errorf("query.service must be a string if provided")
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
// This method provides comprehensive documentation about the SLAP lookup service,
// including usage examples and best practices.
//
// Returns:
//   - string: The service documentation in markdown format
//   - error: Always nil (no errors expected)
func (s *SLAPLookupService) GetDocumentation() (string, error) {
	return LookupDocumentation, nil
}

// GetMetaData returns the service metadata.
// This method provides basic information about the SLAP lookup service
// including name and description.
//
// Returns:
//   - types.MetaData: The service metadata
//   - error: Always nil (no errors expected)
func (s *SLAPLookupService) GetMetaData() (types.MetaData, error) {
	return types.MetaData{
		Name:             "SLAP Lookup Service",
		ShortDescription: "Provides lookup capabilities for SLAP tokens.",
	}, nil
}
