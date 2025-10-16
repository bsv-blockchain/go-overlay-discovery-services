package ship

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const TxID = "bdf1e48e845a65ba8c139c9b94844de30716f38d53787ba0a435e8705c4216d5"

// Static error variables for testing
var (
	errTestStorage = errors.New("storage error")
)

// Mock implementations for testing

// MockStorage is a mock implementation of Storage interface methods
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) StoreSHIPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, topic string) error {
	args := m.Called(ctx, txid, outputIndex, identityKey, domain, topic)
	return args.Error(0)
}

func (m *MockStorage) DeleteSHIPRecord(ctx context.Context, txid string, outputIndex int) error {
	args := m.Called(ctx, txid, outputIndex)
	return args.Error(0)
}

func (m *MockStorage) FindRecord(ctx context.Context, query types.SHIPQuery) ([]types.UTXOReference, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockStorage) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	args := m.Called(ctx, limit, skip, sortOrder)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockStorage) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Mock PushDropDecoder and Utils are no longer needed since we use real implementations

// Test helper functions

func createTestSHIPLookupService() (*LookupService, *MockStorage) {
	mockStorage := new(MockStorage)
	service := NewLookupService(mockStorage)
	return service, mockStorage
}

// createValidPushDropScript creates a valid PushDrop script with the specified fields
func createValidPushDropScript(fields [][]byte) string {
	// Create a valid public key (33 bytes) - this is a known valid public key
	pubKeyHex := "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	pubKeyBytes, _ := hex.DecodeString(pubKeyHex)

	// Start building the script
	s := &script.Script{}

	// Add public key
	if err := s.AppendPushData(pubKeyBytes); err != nil {
		return ""
	}

	// Add OP_CHECKSIG
	if err := s.AppendOpcodes(script.OpCHECKSIG); err != nil {
		return ""
	}

	// Add fields using PushData
	for _, field := range fields {
		if err := s.AppendPushData(field); err != nil {
			return ""
		}
	}

	// Add DROP operations to remove fields from stack
	notYetDropped := len(fields)
	for notYetDropped > 1 {
		if err := s.AppendOpcodes(script.Op2DROP); err != nil {
			return ""
		}
		notYetDropped -= 2
	}
	if notYetDropped != 0 {
		if err := s.AppendOpcodes(script.OpDROP); err != nil {
			return ""
		}
	}

	return s.String()
}

// createValidPushDropResult helper removed - using real PushDrop scripts instead

// Test NewLookupService

func TestNewSHIPLookupService(t *testing.T) {
	mockStorage := new(MockStorage)

	service := NewLookupService(mockStorage)

	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

// Test OutputAdmittedByTopic

func TestOutputAdmittedByTopic_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create valid PushDrop script with SHIP data
	fields := [][]byte{
		[]byte("SHIP"),                // Protocol identifier
		{0x01, 0x02, 0x03, 0x04},      // Identity key bytes
		[]byte("https://example.com"), // Domain
		[]byte("tm_bridge"),           // Topic
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         Topic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	// Set up mock for storage (txid is now hex-encoded from outpoint)
	mockStorage.On("StoreSHIPRecord", mock.Anything, TxID, 0, "01020304", "https://example.com", "tm_bridge").Return(nil)

	// Execute
	err = service.OutputAdmittedByTopic(context.Background(), payload)

	// Assert
	require.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputAdmittedByTopic_IgnoreNonSHIPTopic(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create invalid script that can't be decoded
	scriptObj, err := script.NewFromHex("deadbeef")
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         "tm_other",
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	require.NoError(t, err) // Should silently ignore non-SHIP topics
}

func TestOutputAdmittedByTopic_PushDropDecodeError(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create invalid script that can't be decoded
	scriptObj, err := script.NewFromHex("deadbeef")
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         Topic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PushDrop locking script")
}

func TestOutputAdmittedByTopic_InsufficientFields(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create PushDrop script with only 2 fields instead of required 4
	fields := [][]byte{
		[]byte("SHIP"),
		{0x01, 0x02, 0x03, 0x04},
	}
	invalidScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(invalidScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         Topic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected at least 4 fields")
	assert.Contains(t, err.Error(), "got 2")
}

func TestOutputAdmittedByTopic_IgnoreNonSHIPProtocol(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create valid PushDrop script with SLAP protocol instead of SHIP
	fields := [][]byte{
		[]byte("SLAP"),                // Different protocol
		{0x01, 0x02, 0x03, 0x04},      // Identity key bytes
		[]byte("https://example.com"), // Domain
		[]byte("tm_bridge"),           // Topic
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         Topic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	require.NoError(t, err) // Should silently ignore non-SHIP protocols
}

// Test OutputSpent

func TestOutputSpent_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputSpent{
		Topic:    Topic,
		Outpoint: outpoint,
	}

	mockStorage.On("DeleteSHIPRecord", mock.Anything, TxID, 0).Return(nil)

	err = service.OutputSpent(context.Background(), payload)
	require.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputSpent_IgnoreNonSHIPTopic(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputSpent{
		Topic:    "tm_other",
		Outpoint: outpoint,
	}

	err = service.OutputSpent(context.Background(), payload)
	require.NoError(t, err) // Should silently ignore non-SHIP topics
}

// Test OutputEvicted

func TestOutputEvicted_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	mockStorage.On("DeleteSHIPRecord", mock.Anything, TxID, 0).Return(nil)

	err = service.OutputEvicted(context.Background(), outpoint)
	require.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputEvicted_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	mockStorage.On("DeleteSHIPRecord", mock.Anything, TxID, 0).Return(errTestStorage)

	err = service.OutputEvicted(context.Background(), outpoint)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test Lookup

func TestLookup_LegacyFindAll(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   json.RawMessage(`"findAll"`),
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
		{Txid: "def456", OutputIndex: 1},
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	require.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_NilQuery(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   json.RawMessage{},
	}

	_, err := service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a valid query must be provided")
}

func TestLookup_WrongService(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := &lookup.LookupQuestion{
		Service: "ls_other",
		Query:   json.RawMessage(`"findAll"`),
	}

	_, err := service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "lookup service not supported")
}

func TestLookup_InvalidStringQuery(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   json.RawMessage(`"invalid"`),
	}

	_, err := service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid string query: only 'findAll' is supported")
}

func TestLookup_ObjectQuery_FindAll(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	findAll := true
	limit := 10
	skip := 5
	sortOrder := types.SortOrderAsc

	query := map[string]interface{}{
		"findAll":   findAll,
		"limit":     limit,
		"skip":      skip,
		"sortOrder": sortOrder,
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindAll", mock.Anything, &limit, &skip, &sortOrder).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	require.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_ObjectQuery_SpecificQuery(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	domain := "https://example.com"
	topics := []string{"tm_bridge", "tm_sync"}
	identityKey := "01020304"

	query := map[string]interface{}{
		"domain":      domain,
		"topics":      topics,
		"identityKey": identityKey,
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	expectedQuery := types.SHIPQuery{
		Domain:      &domain,
		Topics:      topics,
		IdentityKey: &identityKey,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindRecord", mock.Anything, expectedQuery).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	require.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_ValidationError_NegativeLimit(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"limit": -1,
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	_, err = service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query.limit must be a positive number")
}

func TestLookup_ValidationError_NegativeSkip(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"skip": -1,
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	_, err = service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query.skip must be a non-negative number")
}

func TestLookup_ValidationError_InvalidSortOrder(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"sortOrder": "invalid",
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	_, err = service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query.sortOrder must be 'asc' or 'desc'")
}

// Test GetDocumentation

func TestGetDocumentation(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	doc := service.GetDocumentation()
	assert.Equal(t, LookupDocumentation, doc)
	assert.Contains(t, doc, "# SHIP Lookup Service")
	assert.Contains(t, doc, "Service Host Interconnect Protocol")
}

// Test GetMetaData

func TestGetMetaData(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	metadata := service.GetMetaData()
	assert.Equal(t, "SHIP Lookup Service", metadata.Name)
	assert.Equal(t, "Provides lookup capabilities for SHIP tokens.", metadata.Description)
	assert.Empty(t, metadata.Icon)
	assert.Empty(t, metadata.Version)
	assert.Empty(t, metadata.InfoUrl)
}

// Test edge cases and error scenarios

func TestLookup_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   json.RawMessage(`"findAll"`),
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return([]types.UTXOReference{}, errTestStorage)

	_, err := service.Lookup(context.Background(), question)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

func TestOutputAdmittedByTopic_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create valid PushDrop script with SHIP data
	fields := [][]byte{
		[]byte("SHIP"),                // Protocol identifier
		{0x01, 0x02, 0x03, 0x04},      // Identity key bytes
		[]byte("https://example.com"), // Domain
		[]byte("tm_bridge"),           // Topic
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxID)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         Topic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	mockStorage.On("StoreSHIPRecord", mock.Anything, TxID, 0, "01020304", "https://example.com", "tm_bridge").Return(errTestStorage)

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test complex query scenarios

func TestLookup_ComplexObjectQuery(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	domain := "https://example.com"
	topics := []string{"tm_bridge", "tm_sync", "tm_token"}
	identityKey := "deadbeef01020304"
	limit := 50
	skip := 10
	sortOrder := types.SortOrderDesc

	query := map[string]interface{}{
		"domain":      domain,
		"topics":      topics,
		"identityKey": identityKey,
		"limit":       limit,
		"skip":        skip,
		"sortOrder":   sortOrder,
	}

	queryJSON, err := json.Marshal(query)
	require.NoError(t, err)
	question := &lookup.LookupQuestion{
		Service: Service,
		Query:   queryJSON,
	}

	expectedQuery := types.SHIPQuery{
		Domain:      &domain,
		Topics:      topics,
		IdentityKey: &identityKey,
		Limit:       &limit,
		Skip:        &skip,
		SortOrder:   &sortOrder,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
		{Txid: "def456", OutputIndex: 1},
	}

	mockStorage.On("FindRecord", mock.Anything, expectedQuery).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	require.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
		assert.Len(t, utxos, 2)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestSHIPLookupService_OutputNoLongerRetainedInHistory(t *testing.T) {
	mockStorage := &MockStorage{}
	service := NewLookupService(mockStorage)

	// Create test outpoint
	outpoint := &transaction.Outpoint{
		Txid:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Index: 0,
	}

	// Test that OutputNoLongerRetainedInHistory does nothing (no-op)
	err := service.OutputNoLongerRetainedInHistory(context.Background(), outpoint, "tm_ship")
	require.NoError(t, err)

	// Verify no storage methods were called
	mockStorage.AssertExpectations(t)
}

func TestSHIPLookupService_OutputBlockHeightUpdated(t *testing.T) {
	mockStorage := &MockStorage{}
	service := NewLookupService(mockStorage)

	// Create test transaction ID
	txidBytes, err := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)
	txid := &chainhash.Hash{}
	copy(txid[:], txidArray[:])

	// Test that OutputBlockHeightUpdated does nothing (no-op)
	err = service.OutputBlockHeightUpdated(context.Background(), txid, 12345, 0)
	require.NoError(t, err)

	// Verify no storage methods were called
	mockStorage.AssertExpectations(t)
}
