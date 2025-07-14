package slap

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// SLAPMongoStorage implements MongoDB storage for SLAP protocol
type SLAPMongoStorage struct {
	db          *mongo.Database
	slapRecords *mongo.Collection
}

// SLAPStorageRecord represents a SLAP record in MongoDB
type SLAPStorageRecord struct {
	Outpoint    string    `bson:"outpoint"`
	IdentityKey string    `bson:"identityKey"`
	Domain      string    `bson:"domain"`
	Service     string    `bson:"service"`
	CreatedAt   time.Time `bson:"createdAt"`
}

// NewSLAPMongoStorage creates a new SLAPMongoStorage instance
func NewSLAPMongoStorage(db *mongo.Database) *SLAPMongoStorage {
	return &SLAPMongoStorage{
		db:          db,
		slapRecords: db.Collection("slapRecords"),
	}
}

// EnsureIndexes creates necessary indexes for the collections
func (s *SLAPMongoStorage) EnsureIndexes(ctx context.Context) error {
	// Create unique index on outpoint
	outpointIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "outpoint", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	}
	
	// Create index for queries
	queryIndex := mongo.IndexModel{
		Keys: bson.D{
			{Key: "domain", Value: 1},
			{Key: "service", Value: 1},
		},
	}
	
	_, err := s.slapRecords.Indexes().CreateMany(ctx, []mongo.IndexModel{outpointIndex, queryIndex})
	return err
}

// StoreSLAPRecord stores a SLAP record
func (s *SLAPMongoStorage) StoreSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey string, domain string, service string) error {
	record := SLAPStorageRecord{
		Outpoint:    outpoint.String(),
		IdentityKey: identityKey,
		Domain:      domain,
		Service:     service,
		CreatedAt:   time.Now(),
	}
	
	_, err := s.slapRecords.InsertOne(ctx, record)
	return err
}

// DeleteSLAPRecord deletes a SLAP record
func (s *SLAPMongoStorage) DeleteSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint) error {
	filter := bson.M{
		"outpoint": outpoint.String(),
	}
	
	_, err := s.slapRecords.DeleteOne(ctx, filter)
	return err
}

// FindRecord finds SLAP records based on a given query
func (s *SLAPMongoStorage) FindRecord(ctx context.Context, query *types.SLAPQuery) ([]*transaction.Outpoint, error) {
	filter := bson.M{}
	
	// Add domain to the query if provided
	if query.Domain != nil && *query.Domain != "" {
		filter["domain"] = *query.Domain
	}
	
	// Add service to the query if provided
	if query.Service != nil && *query.Service != "" {
		filter["service"] = *query.Service
	}
	
	// Add identityKey to the query if provided
	if query.IdentityKey != nil && *query.IdentityKey != "" {
		filter["identityKey"] = *query.IdentityKey
	}
	
	// Project only outpoint
	projection := bson.M{
		"outpoint": 1,
		"_id":      0,
	}
	
	cursor, err := s.slapRecords.Find(ctx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var results []*transaction.Outpoint
	for cursor.Next(ctx) {
		var record struct {
			Outpoint string `bson:"outpoint"`
		}
		
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		
		// Parse outpoint string
		outpoint, err := transaction.OutpointFromString(record.Outpoint)
		if err != nil {
			return nil, err
		}
		
		results = append(results, outpoint)
	}
	
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	
	return results, nil
}

// FindAll returns all results tracked by the overlay
func (s *SLAPMongoStorage) FindAll(ctx context.Context) ([]*transaction.Outpoint, error) {
	// Project only outpoint
	projection := bson.M{
		"outpoint": 1,
		"_id":      0,
	}
	
	cursor, err := s.slapRecords.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	var results []*transaction.Outpoint
	for cursor.Next(ctx) {
		var record struct {
			Outpoint string `bson:"outpoint"`
		}
		
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		
		// Parse outpoint string
		outpoint, err := transaction.OutpointFromString(record.Outpoint)
		if err != nil {
			return nil, err
		}
		
		results = append(results, outpoint)
	}
	
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	
	return results, nil
}