// Package ship implements the SHIP (Service Host Interconnect Protocol) storage functionality.
// MongoDB-based storage and retrieval of SHIP records.
package ship

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// StorageInterface defines the interface for SHIP storage operations.
type StorageInterface interface {
	StoreSHIPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, topic string) error
	DeleteSHIPRecord(ctx context.Context, txid string, outputIndex int) error
	FindRecord(ctx context.Context, query types.SHIPQuery) ([]types.UTXOReference, error)
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	EnsureIndexes(ctx context.Context) error
}

// Storage implements a storage engine for SHIP protocol records.
// It provides MongoDB-based storage with methods for storing, deleting,
// and querying SHIP records with support for pagination and filtering.
type Storage struct {
	db          *mongo.Database
	shipRecords *mongo.Collection
}

// Compile-time verification that Storage implements SHIPStorageInterface
// is performed in lookup_service.go to avoid circular dependencies

// NewStorage constructs a new Storage instance with the provided MongoDB database.
// The storage uses a collection named "shipRecords" to store SHIP protocol records.
func NewStorage(db *mongo.Database) *Storage {
	return &Storage{
		db:          db,
		shipRecords: db.Collection("shipRecords"),
	}
}

// EnsureIndexes creates the necessary indexes for the SHIP records collection.
// This method should be called once during application initialization to optimize
// query performance. It creates a compound index on domain and topic fields.
func (s *Storage) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "domain", Value: 1},
			{Key: "topic", Value: 1},
		},
	}

	_, err := s.shipRecords.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create indexes for SHIP records: %w", err)
	}

	return nil
}

// StoreSHIPRecord stores a new SHIP record in the database.
// The record includes transaction information, identity key, domain, topic,
// and an automatically generated creation timestamp.
func (s *Storage) StoreSHIPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, topic string) error {
	record := types.SHIPRecord{
		Txid:        txid,
		OutputIndex: outputIndex,
		IdentityKey: identityKey,
		Domain:      domain,
		Topic:       topic,
		CreatedAt:   time.Now(),
	}

	_, err := s.shipRecords.InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to store SHIP record: %w", err)
	}

	return nil
}

// DeleteSHIPRecord deletes a SHIP record from the database based on transaction ID and output index.
// This method is typically used when a UTXO is spent and the associated SHIP record should be removed.
func (s *Storage) DeleteSHIPRecord(ctx context.Context, txid string, outputIndex int) error {
	filter := bson.M{
		"txid":        txid,
		"outputIndex": outputIndex,
	}

	_, err := s.shipRecords.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete SHIP record: %w", err)
	}

	return nil
}

// FindRecord finds SHIP records based on the provided query parameters.
// It supports filtering by domain, topics, and identity key, with pagination and sorting options.
// Returns only UTXO references (txid and outputIndex) as projection for efficient querying.
func (s *Storage) FindRecord(ctx context.Context, query types.SHIPQuery) ([]types.UTXOReference, error) {
	mongoQuery := bson.M{}

	// Add domain filter if provided
	if query.Domain != nil {
		mongoQuery["domain"] = *query.Domain
	}

	// Add topics filter using $in operator if provided
	if len(query.Topics) > 0 {
		mongoQuery["topic"] = bson.M{"$in": query.Topics}
	}

	// Add identity key filter if provided
	if query.IdentityKey != nil {
		mongoQuery["identityKey"] = *query.IdentityKey
	}

	// Set up the find options
	findOpts := options.Find()

	// Set projection to return only txid, outputIndex, and createdAt
	findOpts.SetProjection(bson.M{
		"txid":        1,
		"outputIndex": 1,
		"createdAt":   1,
	})

	// Set sort order (default to descending by createdAt)
	sortOrder := -1 // descending
	if query.SortOrder != nil && *query.SortOrder == types.SortOrderAsc {
		sortOrder = 1 // ascending
	}
	findOpts.SetSort(bson.M{"createdAt": sortOrder})

	// Apply pagination
	if query.Skip != nil && *query.Skip > 0 {
		findOpts.SetSkip(int64(*query.Skip))
	}

	if query.Limit != nil && *query.Limit > 0 {
		findOpts.SetLimit(int64(*query.Limit))
	}

	// Execute the query
	cursor, err := s.shipRecords.Find(ctx, mongoQuery, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to find SHIP records: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	// Collect results
	var results []types.UTXOReference
	for cursor.Next(ctx) {
		var record struct {
			Txid        string `bson:"txid"`
			OutputIndex int    `bson:"outputIndex"`
		}

		if err := cursor.Decode(&record); err != nil {
			return nil, fmt.Errorf("failed to decode SHIP record: %w", err)
		}

		results = append(results, types.UTXOReference{
			Txid:        record.Txid,
			OutputIndex: record.OutputIndex,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while finding SHIP records: %w", err)
	}

	return results, nil
}

// FindAll returns all SHIP records in the database with optional pagination and sorting.
// This method ignores all filtering criteria and returns all available records.
// Returns only UTXO references (txid and outputIndex) as projection for efficient querying.
func (s *Storage) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	// Set up the find options
	findOpts := options.Find()

	// Set projection to return only txid, outputIndex, and createdAt
	findOpts.SetProjection(bson.M{
		"txid":        1,
		"outputIndex": 1,
		"createdAt":   1,
	})

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

	// Execute the query (empty filter to get all records)
	cursor, err := s.shipRecords.Find(ctx, bson.M{}, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to find all SHIP records: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	// Collect results
	var results []types.UTXOReference
	for cursor.Next(ctx) {
		var record struct {
			Txid        string `bson:"txid"`
			OutputIndex int    `bson:"outputIndex"`
		}

		if err := cursor.Decode(&record); err != nil {
			return nil, fmt.Errorf("failed to decode SHIP record: %w", err)
		}

		results = append(results, types.UTXOReference{
			Txid:        record.Txid,
			OutputIndex: record.OutputIndex,
		})
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error while finding all SHIP records: %w", err)
	}

	return results, nil
}
