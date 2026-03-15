// Package slap implements the SLAP (Service Lookup Availability Protocol) storage functionality.
// MongoDB-based storage and retrieval of SLAP records.
package slap

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/shared"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// StorageInterface defines the interface for SLAP storage operations.
type StorageInterface interface {
	StoreSLAPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, service string) error
	DeleteSLAPRecord(ctx context.Context, txid string, outputIndex int) error
	FindRecord(ctx context.Context, query types.SLAPQuery) ([]types.UTXOReference, error)
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	EnsureIndexes(ctx context.Context) error
}

// Storage implements a storage engine for SLAP protocol records.
// It provides MongoDB-based storage with methods for storing, deleting,
// and querying SLAP records with support for pagination and filtering.
type Storage struct {
	db          *mongo.Database
	slapRecords *mongo.Collection
}

// NewStorage constructs a new Storage instance with the provided MongoDB database.
// The storage uses a collection named "slapRecords" to store SLAP protocol records.
func NewStorage(db *mongo.Database) *Storage {
	return &Storage{
		db:          db,
		slapRecords: db.Collection("slapRecords"),
	}
}

// EnsureIndexes creates the necessary indexes for the SLAP records collection.
// This method should be called once during application initialization to optimize
// query performance. It creates a compound index on domain and service fields.
func (s *Storage) EnsureIndexes(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			{Key: "domain", Value: 1},
			{Key: "service", Value: 1},
		},
	}

	_, err := s.slapRecords.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("failed to create indexes for SLAP records: %w", err)
	}

	return nil
}

// StoreSLAPRecord stores a new SLAP record in the database.
// The record includes transaction information, identity key, domain, service,
// and an automatically generated creation timestamp.
func (s *Storage) StoreSLAPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, service string) error {
	record := types.SLAPRecord{
		Txid:        txid,
		OutputIndex: outputIndex,
		IdentityKey: identityKey,
		Domain:      domain,
		Service:     service,
		CreatedAt:   time.Now(),
	}

	_, err := s.slapRecords.InsertOne(ctx, record)
	if err != nil {
		return fmt.Errorf("failed to store SLAP record: %w", err)
	}

	return nil
}

// DeleteSLAPRecord deletes a SLAP record from the database based on transaction ID and output index.
// This method is typically used when a UTXO is spent and the associated SLAP record should be removed.
func (s *Storage) DeleteSLAPRecord(ctx context.Context, txid string, outputIndex int) error {
	filter := bson.M{
		"txid":        txid,
		"outputIndex": outputIndex,
	}

	_, err := s.slapRecords.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete SLAP record: %w", err)
	}

	return nil
}

// FindRecord finds SLAP records based on the provided query parameters.
// It supports filtering by domain, service, and identity key, with pagination and sorting options.
// Returns only UTXO references (txid and outputIndex) as projection for efficient querying.
func (s *Storage) FindRecord(ctx context.Context, query types.SLAPQuery) ([]types.UTXOReference, error) {
	mongoQuery := bson.M{}

	// Add domain filter if provided
	if query.Domain != nil {
		mongoQuery["domain"] = *query.Domain
	}

	// Add service filter if provided
	if query.Service != nil {
		mongoQuery["service"] = *query.Service
	}

	// Add identity key filter if provided
	if query.IdentityKey != nil {
		mongoQuery["identityKey"] = *query.IdentityKey
	}

	// Set up the find options
	findOpts := options.Find()
	findOpts.SetProjection(shared.UTXOProjection)
	shared.ApplyPaginationOpts(findOpts, query.SortOrder, query.Skip, query.Limit)

	// Execute the query
	cursor, err := s.slapRecords.Find(ctx, mongoQuery, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to find SLAP records: %w", err)
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	return shared.CollectUTXORefs(ctx, cursor, "SLAP")
}

// FindAll returns all SLAP records in the database with optional pagination and sorting.
// This method ignores all filtering criteria and returns all available records.
// Returns only UTXO references (txid and outputIndex) as projection for efficient querying.
func (s *Storage) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	return shared.FindAllRecords(ctx, s.slapRecords, limit, skip, sortOrder, "SLAP")
}
