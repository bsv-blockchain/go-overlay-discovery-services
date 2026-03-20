package shared

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bsv-blockchain/go-sdk/overlay/lookup"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Common error variables for lookup execution.
var (
	ErrValidQueryMustBeProvided  = errors.New("a valid query must be provided")
	ErrLookupServiceNotSupported = errors.New("lookup service not supported")
	ErrInvalidStringQuery        = errors.New("invalid string query: only 'findAll' is supported")
)

// QueryExecutor defines the operations needed to execute a lookup query.
// Both SHIP and SLAP lookup services implement this interface to share
// the common Lookup method logic.
type QueryExecutor interface {
	// ServiceName returns the expected service identifier (e.g. "ls_ship" or "ls_slap").
	ServiceName() string
	// FindAll returns all records with optional pagination.
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	// ParseAndExecuteQuery parses a raw query interface into a typed query,
	// validates it, and executes the appropriate storage call.
	ParseAndExecuteQuery(ctx context.Context, queryInterface interface{}) ([]types.UTXOReference, error)
}

// ExecuteLookup implements the common Lookup logic shared by SHIP and SLAP lookup services.
// It validates the question, parses the query JSON, handles the "findAll" string shortcut,
// and delegates typed query execution to the provided QueryExecutor.
func ExecuteLookup(ctx context.Context, question *lookup.LookupQuestion, executor QueryExecutor) (*lookup.LookupAnswer, error) {
	// Validate required fields
	if len(question.Query) == 0 {
		return nil, ErrValidQueryMustBeProvided
	}

	if question.Service != executor.ServiceName() {
		return nil, fmt.Errorf("%w: expected '%s', got '%s'", ErrLookupServiceNotSupported, executor.ServiceName(), question.Service)
	}

	// Parse the query from JSON
	var queryInterface interface{}
	if err := json.Unmarshal(question.Query, &queryInterface); err != nil {
		return nil, fmt.Errorf("failed to parse query JSON: %w", err)
	}

	// Handle legacy "findAll" string query
	if queryStr, ok := queryInterface.(string); ok {
		if queryStr == "findAll" {
			utxos, err := executor.FindAll(ctx, nil, nil, nil)
			if err != nil {
				return nil, err
			}
			return ConvertUTXOsToLookupAnswer(utxos), nil
		}
		return nil, fmt.Errorf("%w: got '%s'", ErrInvalidStringQuery, queryStr)
	}

	// Handle object-based query via the executor
	utxos, err := executor.ParseAndExecuteQuery(ctx, queryInterface)
	if err != nil {
		return nil, fmt.Errorf("invalid query format: %w", err)
	}

	return ConvertUTXOsToLookupAnswer(utxos), nil
}

// ParseQueryJSON is a helper that marshals a raw query interface to JSON and
// unmarshals it into the target type. Used by both SHIP and SLAP parseQueryObject methods.
func ParseQueryJSON(query, target interface{}) error {
	jsonBytes, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query object: %w", err)
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal query object: %w", err)
	}

	return nil
}
