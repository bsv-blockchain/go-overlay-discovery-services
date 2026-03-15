// Package shared provides common helpers for SHIP and SLAP storage and lookup operations.
package shared

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// UTXOProjection is the standard projection for returning UTXO references.
var UTXOProjection = bson.M{
	"txid":        1,
	"outputIndex": 1,
	"createdAt":   1,
}

// ApplyPaginationOpts configures sort, skip, and limit on find options.
func ApplyPaginationOpts(findOpts *options.FindOptions, sortOrder *types.SortOrder, skip *int, limit *int) {
	// Set sort order (default to descending by createdAt)
	mongoSortOrder := -1 // descending
	if sortOrder != nil && *sortOrder == types.SortOrderAsc {
		mongoSortOrder = 1 // ascending
	}
	findOpts.SetSort(bson.M{"createdAt": mongoSortOrder})

	// Apply pagination
	if skip != nil && *skip > 0 {
		findOpts.SetSkip(int64(*skip))
	}

	if limit != nil && *limit > 0 {
		findOpts.SetLimit(int64(*limit))
	}
}

// CollectUTXORefs iterates a mongo cursor and collects UTXO references.
func CollectUTXORefs(ctx context.Context, cursor *mongo.Cursor, recordType string) ([]types.UTXOReference, error) {
	var results []types.UTXOReference
	for cursor.Next(ctx) {
		var record struct {
			Txid        string `bson:"txid"`
			OutputIndex int    `bson:"outputIndex"`
		}

		if err := cursor.Decode(&record); err != nil {
			return nil, fmt.Errorf("failed to decode %s record: %w", recordType, err)
		}

		results = append(results, types.UTXOReference{
			Txid:        record.Txid,
			OutputIndex: record.OutputIndex,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while finding %s records: %w", recordType, err)
	}

	return results, nil
}

// FindAllRecords executes a find-all query with pagination on the given collection.
func FindAllRecords(ctx context.Context, collection *mongo.Collection, limit, skip *int, sortOrder *types.SortOrder, recordType string) ([]types.UTXOReference, error) {
	findOpts := options.Find()
	findOpts.SetProjection(UTXOProjection)
	ApplyPaginationOpts(findOpts, sortOrder, skip, limit)

	cursor, err := collection.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to find all %s records: %w", recordType, err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	return CollectUTXORefs(ctx, cursor, recordType)
}
