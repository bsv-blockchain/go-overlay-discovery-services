package ship

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// SHIPMongoStorage implements MongoDB storage for SHIP protocol
type SHIPMongoStorage struct {
	db           *mongo.Database
	shipRecords  *mongo.Collection
}

// SHIPStorageRecord represents a SHIP record in MongoDB
type SHIPStorageRecord struct {
	Outpoint    string    `bson:"outpoint"`
	IdentityKey string    `bson:"identityKey"`
	Domain      string    `bson:"domain"`
	Topic       string    `bson:"topic"`
	CreatedAt   time.Time `bson:"createdAt"`
}

// NewSHIPMongoStorage creates a new SHIPMongoStorage instance
func NewSHIPMongoStorage(db *mongo.Database) *SHIPMongoStorage {
	return &SHIPMongoStorage{
		db:          db,
		shipRecords: db.Collection("shipRecords"),
	}
}

// EnsureIndexes creates necessary indexes for the collections
func (s *SHIPMongoStorage) EnsureIndexes(ctx context.Context) error {
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
			{Key: "topic", Value: 1},
		},
	}
	
	_, err := s.shipRecords.Indexes().CreateMany(ctx, []mongo.IndexModel{outpointIndex, queryIndex})
	return err
}

// StoreSHIPRecord stores a SHIP record
func (s *SHIPMongoStorage) StoreSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey string, domain string, topic string) error {
	record := SHIPStorageRecord{
		Outpoint:    outpoint.String(),
		IdentityKey: identityKey,
		Domain:      domain,
		Topic:       topic,
		CreatedAt:   time.Now(),
	}
	
	_, err := s.shipRecords.InsertOne(ctx, record)
	return err
}

// DeleteSHIPRecord deletes a SHIP record
func (s *SHIPMongoStorage) DeleteSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint) error {
	filter := bson.M{
		"outpoint": outpoint.String(),
	}
	
	_, err := s.shipRecords.DeleteOne(ctx, filter)
	return err
}

// FindRecord finds SHIP records based on a given query
func (s *SHIPMongoStorage) FindRecord(ctx context.Context, query *types.SHIPQuery) ([]*transaction.Outpoint, error) {
	filter := bson.M{}
	
	// Add domain to the query if provided
	if query.Domain != nil && *query.Domain != "" {
		filter["domain"] = *query.Domain
	}
	
	// Add topics to the query if provided
	if len(query.Topics) > 0 {
		filter["topic"] = bson.M{"$in": query.Topics}
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
	
	cursor, err := s.shipRecords.Find(ctx, filter, options.Find().SetProjection(projection))
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
func (s *SHIPMongoStorage) FindAll(ctx context.Context) ([]*transaction.Outpoint, error) {
	// Project only outpoint
	projection := bson.M{
		"outpoint": 1,
		"_id":      0,
	}
	
	cursor, err := s.shipRecords.Find(ctx, bson.M{}, options.Find().SetProjection(projection))
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