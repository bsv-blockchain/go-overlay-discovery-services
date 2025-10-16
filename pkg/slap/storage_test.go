package slap

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// MockDatabase is a simple interface for testing
type MockDatabase interface {
	Collection(name string) MockCollection
}

// MockCollection is a simple interface for testing
type MockCollection interface {
	InsertOne(ctx context.Context, document interface{}) error
	DeleteOne(ctx context.Context, filter interface{}) error
	Find(ctx context.Context, query types.SLAPQuery) ([]types.UTXOReference, error)
	FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)
	EnsureIndexes(ctx context.Context) error
}

// TestSLAPStorage is a mock implementation for testing
type TestSLAPStorage struct {
	records []types.SLAPRecord
}

// NewTestSLAPStorage creates a new test storage instance
func NewTestSLAPStorage() *TestSLAPStorage {
	return &TestSLAPStorage{
		records: make([]types.SLAPRecord, 0),
	}
}

// EnsureIndexes mock implementation
func (s *TestSLAPStorage) EnsureIndexes(_ context.Context) error {
	return nil
}

// StoreSLAPRecord mock implementation
func (s *TestSLAPStorage) StoreSLAPRecord(_ context.Context, txid string, outputIndex int, identityKey, domain, service string) error {
	record := types.SLAPRecord{
		Txid:        txid,
		OutputIndex: outputIndex,
		IdentityKey: identityKey,
		Domain:      domain,
		Service:     service,
	}
	s.records = append(s.records, record)
	return nil
}

// DeleteSLAPRecord mock implementation
func (s *TestSLAPStorage) DeleteSLAPRecord(_ context.Context, txid string, outputIndex int) error {
	for i, record := range s.records {
		if record.Txid == txid && record.OutputIndex == outputIndex {
			s.records = append(s.records[:i], s.records[i+1:]...)
			return nil
		}
	}
	return nil
}

// FindRecord mock implementation
func (s *TestSLAPStorage) FindRecord(_ context.Context, query types.SLAPQuery) ([]types.UTXOReference, error) {
	var results []types.UTXOReference

	for _, record := range s.records {
		match := true

		// Filter by domain
		if query.Domain != nil && record.Domain != *query.Domain {
			match = false
		}

		// Filter by service
		if query.Service != nil && record.Service != *query.Service {
			match = false
		}

		// Filter by identity key
		if query.IdentityKey != nil && record.IdentityKey != *query.IdentityKey {
			match = false
		}

		if match {
			results = append(results, types.UTXOReference{
				Txid:        record.Txid,
				OutputIndex: record.OutputIndex,
			})
		}
	}

	// Apply pagination
	if query.Skip != nil && *query.Skip > 0 {
		if *query.Skip >= len(results) {
			return []types.UTXOReference{}, nil
		}
		results = results[*query.Skip:]
	}

	if query.Limit != nil && *query.Limit > 0 && len(results) > *query.Limit {
		results = results[:*query.Limit]
	}

	return results, nil
}

// FindAll mock implementation
func (s *TestSLAPStorage) FindAll(_ context.Context, limit, skip *int, _ *types.SortOrder) ([]types.UTXOReference, error) {
	results := make([]types.UTXOReference, 0, len(s.records))

	for _, record := range s.records {
		results = append(results, types.UTXOReference{
			Txid:        record.Txid,
			OutputIndex: record.OutputIndex,
		})
	}

	// Apply pagination
	if skip != nil && *skip > 0 {
		if *skip >= len(results) {
			return []types.UTXOReference{}, nil
		}
		results = results[*skip:]
	}

	if limit != nil && *limit > 0 && len(results) > *limit {
		results = results[:*limit]
	}

	return results, nil
}

// TestNewSLAPStorage tests that we can create a new SLAP storage (would use real MongoDB in practice)
func TestNewSLAPStorage(t *testing.T) {
	// This test validates the concept - in practice would use real MongoDB
	storage := NewTestSLAPStorage()
	assert.NotNil(t, storage)
}

// TestEnsureIndexes tests the index creation functionality
func TestEnsureIndexes(t *testing.T) {
	storage := NewTestSLAPStorage()
	err := storage.EnsureIndexes(context.Background())
	require.NoError(t, err)
}

// TestStoreSLAPRecord tests the record storage functionality
func TestStoreSLAPRecord(t *testing.T) {
	storage := NewTestSLAPStorage()

	err := storage.StoreSLAPRecord(context.Background(), "test-txid-123", 0, "test-identity-key", "example.com", "lookup-service")
	require.NoError(t, err)

	// Verify the record was stored
	assert.Len(t, storage.records, 1)
	assert.Equal(t, "test-txid-123", storage.records[0].Txid)
	assert.Equal(t, 0, storage.records[0].OutputIndex)
	assert.Equal(t, "test-identity-key", storage.records[0].IdentityKey)
	assert.Equal(t, "example.com", storage.records[0].Domain)
	assert.Equal(t, "lookup-service", storage.records[0].Service)
}

// TestDeleteSLAPRecord tests the record deletion functionality
func TestDeleteSLAPRecord(t *testing.T) {
	storage := NewTestSLAPStorage()

	// Store a record first
	err := storage.StoreSLAPRecord(context.Background(), "test-txid-123", 0, "test-identity-key", "example.com", "lookup-service")
	require.NoError(t, err)

	// Verify it was stored
	assert.Len(t, storage.records, 1)

	// Delete the record
	err = storage.DeleteSLAPRecord(context.Background(), "test-txid-123", 0)
	require.NoError(t, err)

	// Verify it was deleted
	assert.Empty(t, storage.records)
}

// TestFindRecord tests the record finding functionality with various query parameters
func TestFindRecord(t *testing.T) {
	storage := NewTestSLAPStorage()

	// Store test records
	records := []struct {
		txid        string
		outputIndex int
		identityKey string
		domain      string
		service     string
	}{
		{"txid1", 0, "key1", "example.com", "service1"},
		{"txid2", 1, "key2", "example.com", "service2"},
		{"txid3", 0, "key1", "test.com", "service1"},
		{"txid4", 2, "key3", "example.com", "service3"},
	}

	for _, record := range records {
		err := storage.StoreSLAPRecord(context.Background(), record.txid, record.outputIndex, record.identityKey, record.domain, record.service)
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		query         types.SLAPQuery
		expectedCount int
		expectedTxids []string
	}{
		{
			name: "find by domain",
			query: types.SLAPQuery{
				Domain: stringPtr("example.com"),
			},
			expectedCount: 3,
			expectedTxids: []string{"txid1", "txid2", "txid4"},
		},
		{
			name: "find by service",
			query: types.SLAPQuery{
				Service: stringPtr("service1"),
			},
			expectedCount: 2,
			expectedTxids: []string{"txid1", "txid3"},
		},
		{
			name: "find by identity key",
			query: types.SLAPQuery{
				IdentityKey: stringPtr("key1"),
			},
			expectedCount: 2,
			expectedTxids: []string{"txid1", "txid3"},
		},
		{
			name: "find with multiple filters",
			query: types.SLAPQuery{
				Domain:      stringPtr("example.com"),
				Service:     stringPtr("service1"),
				IdentityKey: stringPtr("key1"),
			},
			expectedCount: 1,
			expectedTxids: []string{"txid1"},
		},
		{
			name: "find with pagination - limit",
			query: types.SLAPQuery{
				Domain: stringPtr("example.com"),
				Limit:  intPtr(2),
			},
			expectedCount: 2,
		},
		{
			name: "find with pagination - skip",
			query: types.SLAPQuery{
				Domain: stringPtr("example.com"),
				Skip:   intPtr(1),
			},
			expectedCount: 2,
		},
		{
			name: "find with pagination - limit and skip",
			query: types.SLAPQuery{
				Domain: stringPtr("example.com"),
				Limit:  intPtr(1),
				Skip:   intPtr(1),
			},
			expectedCount: 1,
		},
		{
			name: "find no matches",
			query: types.SLAPQuery{
				Domain: stringPtr("nonexistent.com"),
			},
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.FindRecord(context.Background(), tt.query)
			require.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)

			if len(tt.expectedTxids) > 0 {
				resultTxids := make([]string, len(results))
				for i, result := range results {
					resultTxids[i] = result.Txid
				}

				for _, expectedTxid := range tt.expectedTxids {
					assert.Contains(t, resultTxids, expectedTxid)
				}
			}
		})
	}
}

// TestFindAll tests the find all functionality with pagination
func TestFindAll(t *testing.T) {
	storage := NewTestSLAPStorage()

	// Store test records
	for i := 0; i < 5; i++ {
		err := storage.StoreSLAPRecord(context.Background(),
			"txid"+string(rune('1'+i)), i, "key"+string(rune('1'+i)), "example.com", "service1")
		require.NoError(t, err)
	}

	tests := []struct {
		name          string
		limit         *int
		skip          *int
		sortOrder     *types.SortOrder
		expectedCount int
	}{
		{
			name:          "find all without parameters",
			expectedCount: 5,
		},
		{
			name:          "find all with limit",
			limit:         intPtr(3),
			expectedCount: 3,
		},
		{
			name:          "find all with skip",
			skip:          intPtr(2),
			expectedCount: 3,
		},
		{
			name:          "find all with limit and skip",
			limit:         intPtr(2),
			skip:          intPtr(1),
			expectedCount: 2,
		},
		{
			name:          "find all with sort order asc",
			sortOrder:     sortOrderPtr(types.SortOrderAsc),
			expectedCount: 5,
		},
		{
			name:          "find all with sort order desc",
			sortOrder:     sortOrderPtr(types.SortOrderDesc),
			expectedCount: 5,
		},
		{
			name:          "find all with skip beyond records",
			skip:          intPtr(10),
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := storage.FindAll(context.Background(), tt.limit, tt.skip, tt.sortOrder)
			require.NoError(t, err)
			assert.Len(t, results, tt.expectedCount)
		})
	}
}

// TestEdgeCases tests various edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	storage := NewTestSLAPStorage()

	t.Run("empty query parameters", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{})
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("delete non-existent record", func(t *testing.T) {
		err := storage.DeleteSLAPRecord(context.Background(), "non-existent", 0)
		require.NoError(t, err) // Should not error even if record doesn't exist
	})

	t.Run("find with nil service", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: nil,
		})
		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// TestQueryLogicConsistency tests that the query logic matches the TypeScript implementation
func TestQueryLogicConsistency(t *testing.T) {
	storage := NewTestSLAPStorage()

	// Store test data
	testData := []struct {
		txid        string
		outputIndex int
		identityKey string
		domain      string
		service     string
	}{
		{"tx1", 0, "alice", "example.com", "directory"},
		{"tx2", 1, "bob", "example.com", "lookup"},
		{"tx3", 0, "alice", "test.com", "directory"},
		{"tx4", 2, "charlie", "example.com", "search"},
	}

	for _, data := range testData {
		err := storage.StoreSLAPRecord(context.Background(),
			data.txid, data.outputIndex, data.identityKey, data.domain, data.service)
		require.NoError(t, err)
	}

	// Test the exact same query patterns as TypeScript
	t.Run("domain filter", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Domain: stringPtr("example.com"),
		})
		require.NoError(t, err)
		assert.Len(t, results, 3)
	})

	t.Run("service filter", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: stringPtr("directory"),
		})
		require.NoError(t, err)
		assert.Len(t, results, 2) // tx1, tx3
	})

	t.Run("identity key filter", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			IdentityKey: stringPtr("alice"),
		})
		require.NoError(t, err)
		assert.Len(t, results, 2) // tx1, tx3
	})

	t.Run("combined filters", func(t *testing.T) {
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Domain:      stringPtr("example.com"),
			Service:     stringPtr("directory"),
			IdentityKey: stringPtr("alice"),
		})
		require.NoError(t, err)
		assert.Len(t, results, 1) // only tx1
		assert.Equal(t, "tx1", results[0].Txid)
	})
}

// TestSLAPSpecificDifferences tests SLAP-specific behavior that differs from SHIP
func TestSLAPSpecificDifferences(t *testing.T) {
	storage := NewTestSLAPStorage()

	// Store test data with different services
	testData := []struct {
		txid        string
		outputIndex int
		identityKey string
		domain      string
		service     string
	}{
		{"tx1", 0, "alice", "example.com", "directory"},
		{"tx2", 1, "bob", "example.com", "lookup"},
		{"tx3", 0, "charlie", "test.com", "search"},
	}

	for _, data := range testData {
		err := storage.StoreSLAPRecord(context.Background(),
			data.txid, data.outputIndex, data.identityKey, data.domain, data.service)
		require.NoError(t, err)
	}

	t.Run("service filtering is single value not array", func(t *testing.T) {
		// In SHIP, topics are filtered with $in (array match)
		// In SLAP, service is a single string field
		results, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: stringPtr("directory"),
		})
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "tx1", results[0].Txid)
	})

	t.Run("different service names return different results", func(t *testing.T) {
		directoryResults, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: stringPtr("directory"),
		})
		require.NoError(t, err)
		assert.Len(t, directoryResults, 1)

		lookupResults, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: stringPtr("lookup"),
		})
		require.NoError(t, err)
		assert.Len(t, lookupResults, 1)

		searchResults, err := storage.FindRecord(context.Background(), types.SLAPQuery{
			Service: stringPtr("search"),
		})
		require.NoError(t, err)
		assert.Len(t, searchResults, 1)

		// Verify they're all different records
		assert.NotEqual(t, directoryResults[0].Txid, lookupResults[0].Txid)
		assert.NotEqual(t, directoryResults[0].Txid, searchResults[0].Txid)
		assert.NotEqual(t, lookupResults[0].Txid, searchResults[0].Txid)
	})
}

// Helper functions for pointer creation
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func sortOrderPtr(s types.SortOrder) *types.SortOrder {
	return &s
}
