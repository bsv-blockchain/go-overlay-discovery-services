package ship

import (
	"context"

	"github.com/bsv-blockchain/go-sdk/transaction"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Storage defines the interface for SHIP protocol storage operations
type Storage interface {
	// EnsureIndexes ensures the necessary indexes are created for the collections
	EnsureIndexes(ctx context.Context) error

	// StoreSHIPRecord stores a SHIP record
	StoreSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey, domain, topic string) error

	// DeleteSHIPRecord deletes a SHIP record
	DeleteSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint) error

	// FindRecord finds SHIP records based on a given query object
	FindRecord(ctx context.Context, query *types.SHIPQuery) ([]*transaction.Outpoint, error)

	// FindAll returns all results tracked by the overlay
	FindAll(ctx context.Context) ([]*transaction.Outpoint, error)
}