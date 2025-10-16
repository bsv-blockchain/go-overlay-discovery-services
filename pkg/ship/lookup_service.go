// Package ship implements the SHIP (Service Host Interconnect Protocol) lookup service functionality.
// the BSV overlay LookupService interface.
package ship

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
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
	errPushDropDecodeFailed      = errors.New("failed to decode PushDrop locking script")
	errInvalidPushDropFields     = errors.New("invalid PushDrop result: expected at least 4 fields")
	errValidQueryMustBeProvided  = errors.New("a valid query must be provided")
	errLookupServiceNotSupported = errors.New("lookup service not supported")
	errInvalidStringQuery        = errors.New("invalid string query: only 'findAll' is supported")
	errQueryDomainInvalid        = errors.New("query.domain must be a string if provided")
	errQueryTopicsInvalid        = errors.New("query.topics must be an array of strings if provided")
	errQueryTopicElementInvalid  = errors.New("query.topics element must be a string")
	errQueryIdentityKeyInvalid   = errors.New("query.identityKey must be a string if provided")
	errQueryLimitInvalid         = errors.New("query.limit must be a positive number if provided")
	errQuerySkipInvalid          = errors.New("query.skip must be a non-negative number if provided")
	errQuerySortOrderInvalid     = errors.New("query.sortOrder must be 'asc' or 'desc' if provided")
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
	// Only process SHIP topic
	if payload.Topic != Topic {
		return nil // Silently ignore non-SHIP topics
	}

	// Use the locking script from payload
	scriptObj := payload.LockingScript

	// Decode the PushDrop locking script
	result := pushdrop.Decode(scriptObj)
	if result == nil {
		return errPushDropDecodeFailed
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return fmt.Errorf("%w: got %d", errInvalidPushDropFields, len(result.Fields))
	}

	// Extract and validate fields
	shipIdentifier := string(result.Fields[0])
	if shipIdentifier != Identifier {
		return nil // Silently ignore non-SHIP protocols
	}

	identityKey := hex.EncodeToString(result.Fields[1])
	domain := string(result.Fields[2])
	topicSupported := string(result.Fields[3])

	// Store the SHIP record
	txid := hex.EncodeToString(payload.Outpoint.Txid[:])
	return s.storage.StoreSHIPRecord(ctx, txid, int(payload.Outpoint.Index), identityKey, domain, topicSupported)
}

// OutputSpent handles an output being spent.
// This method removes the corresponding SHIP record when the UTXO is spent.
func (s *LookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	// Only process SHIP topic
	if payload.Topic != Topic {
		return nil // Silently ignore non-SHIP topics
	}

	// Delete the SHIP record
	txid := hex.EncodeToString(payload.Outpoint.Txid[:])
	return s.storage.DeleteSHIPRecord(ctx, txid, int(payload.Outpoint.Index))
}

// OutputEvicted handles an output being evicted.
// This method removes the corresponding SHIP record when the UTXO is evicted from the mempool.
func (s *LookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	// Delete the SHIP record
	txid := hex.EncodeToString(outpoint.Txid[:])
	return s.storage.DeleteSHIPRecord(ctx, txid, int(outpoint.Index))
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
	// Validate required fields
	if len(question.Query) == 0 {
		return nil, errValidQueryMustBeProvided
	}

	if question.Service != Service {
		return nil, fmt.Errorf("%w: expected '%s', got '%s'", errLookupServiceNotSupported, Service, question.Service)
	}

	// Parse the query from JSON
	var queryInterface interface{}
	if err := json.Unmarshal(question.Query, &queryInterface); err != nil {
		return nil, fmt.Errorf("failed to parse query JSON: %w", err)
	}

	// Handle legacy "findAll" string query
	if queryStr, ok := queryInterface.(string); ok {
		if queryStr == "findAll" {
			utxos, err := s.storage.FindAll(ctx, nil, nil, nil)
			if err != nil {
				return nil, err
			}
			return s.convertUTXOsToLookupAnswer(utxos), nil
		}
		return nil, fmt.Errorf("%w: got '%s'", errInvalidStringQuery, queryStr)
	}

	// Handle object-based query
	queryObj, err := s.parseQueryObject(queryInterface)
	if err != nil {
		return nil, fmt.Errorf("invalid query format: %w", err)
	}

	var utxos []types.UTXOReference
	// Handle findAll with pagination
	if queryObj.FindAll != nil && *queryObj.FindAll {
		utxos, err = s.storage.FindAll(ctx, queryObj.Limit, queryObj.Skip, queryObj.SortOrder)
	} else {
		// Handle specific query with filters
		utxos, err = s.storage.FindRecord(ctx, *queryObj)
	}

	if err != nil {
		return nil, err
	}

	return s.convertUTXOsToLookupAnswer(utxos), nil
}

// parseQueryObject parses and validates a query object
func (s *LookupService) parseQueryObject(query interface{}) (*types.SHIPQuery, error) {
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
func (s *LookupService) validateQuery(query *types.SHIPQuery) error {
	// Validate domain parameter
	if query.Domain != nil {
		if reflect.TypeOf(query.Domain).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.Domain).Elem().Kind() != reflect.String {
			return errQueryDomainInvalid
		}
	}

	// Validate topics parameter
	if query.Topics != nil {
		if reflect.TypeOf(query.Topics).Kind() != reflect.Slice {
			return errQueryTopicsInvalid
		}
		for i, topic := range query.Topics {
			if reflect.TypeOf(topic).Kind() != reflect.String {
				return fmt.Errorf("%w: at index %d", errQueryTopicElementInvalid, i)
			}
		}
	}

	// Validate identityKey parameter
	if query.IdentityKey != nil {
		if reflect.TypeOf(query.IdentityKey).Kind() != reflect.Ptr ||
			reflect.TypeOf(query.IdentityKey).Elem().Kind() != reflect.String {
			return errQueryIdentityKeyInvalid
		}
	}

	// Validate pagination parameters
	if query.Limit != nil {
		if *query.Limit < 0 {
			return errQueryLimitInvalid
		}
	}

	if query.Skip != nil {
		if *query.Skip < 0 {
			return errQuerySkipInvalid
		}
	}

	// Validate sort order parameter
	if query.SortOrder != nil {
		if *query.SortOrder != types.SortOrderAsc && *query.SortOrder != types.SortOrderDesc {
			return errQuerySortOrderInvalid
		}
	}

	return nil
}

// GetDocumentation returns the service documentation.
// This method provides comprehensive documentation about the SHIP lookup service,
// including usage examples and best practices.
func (s *LookupService) GetDocumentation() string {
	return LookupDocumentation
}

// convertUTXOsToLookupAnswer converts a slice of UTXO references to a LookupAnswer
func (s *LookupService) convertUTXOsToLookupAnswer(utxos []types.UTXOReference) *lookup.LookupAnswer {
	// For discovery services, we return the UTXOs as freeform result
	return &lookup.LookupAnswer{
		Type:   lookup.AnswerTypeFreeform,
		Result: utxos,
	}
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
