package slap

import (
	"context"

	"github.com/bsv-blockchain/go-sdk/transaction"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Storage defines the interface for SLAP protocol storage operations
type Storage interface {
	// EnsureIndexes ensures the necessary indexes are created for the collections
	EnsureIndexes(ctx context.Context) error

	// StoreSLAPRecord stores a SLAP record
	StoreSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey, domain, service string) error

	// DeleteSLAPRecord deletes a SLAP record
	DeleteSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint) error

	// FindRecord finds SLAP records based on a given query object
	FindRecord(ctx context.Context, query *types.SLAPQuery) ([]*transaction.Outpoint, error)

	// FindAll returns all results tracked by the overlay
	FindAll(ctx context.Context) ([]*transaction.Outpoint, error)
}