package slap

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

const TxId = "bdf1e48e845a65ba8c139c9b94844de30716f38d53787ba0a435e8705c4216d5"

// Mock implementations for testing

// MockSLAPStorageInterface is a mock implementation of SLAPStorage interface methods
type MockSLAPStorageInterface struct {
	mock.Mock
}

func (m *MockSLAPStorageInterface) StoreSLAPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, service string) error {
	args := m.Called(ctx, txid, outputIndex, identityKey, domain, service)
	return args.Error(0)
}

func (m *MockSLAPStorageInterface) DeleteSLAPRecord(ctx context.Context, txid string, outputIndex int) error {
	args := m.Called(ctx, txid, outputIndex)
	return args.Error(0)
}

func (m *MockSLAPStorageInterface) FindRecord(ctx context.Context, query types.SLAPQuery) ([]types.UTXOReference, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockSLAPStorageInterface) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	args := m.Called(ctx, limit, skip, sortOrder)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockSLAPStorageInterface) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Note: Mock PushDropDecoder and Utils are no longer needed since we use real implementations

// Test helper functions

func createTestSLAPLookupService() (*SLAPLookupService, *MockSLAPStorageInterface) {
	mockStorage := new(MockSLAPStorageInterface)
	service := NewSLAPLookupService(mockStorage)
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
	s.AppendPushData(pubKeyBytes)

	// Add OP_CHECKSIG
	s.AppendOpcodes(script.OpCHECKSIG)

	// Add fields using PushData
	for _, field := range fields {
		s.AppendPushData(field)
	}

	// Add DROP operations to remove fields from stack
	notYetDropped := len(fields)
	for notYetDropped > 1 {
		s.AppendOpcodes(script.Op2DROP)
		notYetDropped -= 2
	}
	if notYetDropped != 0 {
		s.AppendOpcodes(script.OpDROP)
	}

	return s.String()
}

// createValidPushDropResult helper removed - using real PushDrop scripts instead

// Test NewSLAPLookupService

func TestNewSLAPLookupService(t *testing.T) {
	mockStorage := new(MockSLAPStorageInterface)

	service := NewSLAPLookupService(mockStorage)

	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

// Test OutputAdmittedByTopic

func TestOutputAdmittedByTopic_Success(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	// Create valid PushDrop script with SLAP data
	fields := [][]byte{
		[]byte("SLAP"),                 // Protocol identifier
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("ls_treasury"),          // Service
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         SLAPTopic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	// Set up mock for storage (txid is now hex-encoded from outpoint)
	mockStorage.On("StoreSLAPRecord", mock.Anything, TxId, 0, "01020304", "https://example.com", "ls_treasury").Return(nil)

	// Execute
	err = service.OutputAdmittedByTopic(context.Background(), payload)

	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputAdmittedByTopic_IgnoreNonSLAPTopic(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	// Create invalid script that can't be decoded
	scriptObj, err := script.NewFromHex("deadbeef")
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
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
	assert.NoError(t, err) // Should silently ignore non-SLAP topics
}

func TestOutputAdmittedByTopic_PushDropDecodeError(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	// Create invalid script that can't be decoded
	scriptObj, err := script.NewFromHex("deadbeef")
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         SLAPTopic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PushDrop locking script")
}

func TestOutputAdmittedByTopic_InsufficientFields(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	// Create PushDrop script with only 2 fields instead of required 4
	fields := [][]byte{
		[]byte("SLAP"),
		[]byte{0x01, 0x02, 0x03, 0x04},
	}
	invalidScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(invalidScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         SLAPTopic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected at least 4 fields, got 2")
}

func TestOutputAdmittedByTopic_IgnoreNonSLAPProtocol(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	// Create valid PushDrop script with SHIP protocol instead of SLAP
	fields := [][]byte{
		[]byte("SHIP"),                 // Different protocol
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("ls_treasury"),          // Service
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         SLAPTopic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	assert.NoError(t, err) // Should silently ignore non-SLAP protocols
}

// Test OutputSpent

func TestOutputSpent_Success(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputSpent{
		Topic:    SLAPTopic,
		Outpoint: outpoint,
	}

	mockStorage.On("DeleteSLAPRecord", mock.Anything, TxId, 0).Return(nil)

	err = service.OutputSpent(context.Background(), payload)
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputSpent_IgnoreNonSLAPTopic(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
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
	assert.NoError(t, err) // Should silently ignore non-SLAP topics
}

// Test OutputEvicted

func TestOutputEvicted_Success(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	mockStorage.On("DeleteSLAPRecord", mock.Anything, TxId, 0).Return(nil)

	err = service.OutputEvicted(context.Background(), outpoint)
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputEvicted_StorageError(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	mockStorage.On("DeleteSLAPRecord", mock.Anything, TxId, 0).Return(errors.New("storage error"))

	err = service.OutputEvicted(context.Background(), outpoint)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test Lookup

func TestLookup_LegacyFindAll(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   json.RawMessage(`"findAll"`),
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
		{Txid: "def456", OutputIndex: 1},
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_NilQuery(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   json.RawMessage{},
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "a valid query must be provided")
}

func TestLookup_WrongService(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	question := &lookup.LookupQuestion{
		Service: "ls_other",
		Query:   json.RawMessage(`"findAll"`),
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup service not supported")
}

func TestLookup_InvalidStringQuery(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   json.RawMessage(`"invalid"`),
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid string query: only 'findAll' is supported")
}

func TestLookup_ObjectQuery_FindAll(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

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

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindAll", mock.Anything, &limit, &skip, &sortOrder).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_ObjectQuery_SpecificQuery(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	domain := "https://example.com"
	service_name := "ls_treasury"
	identityKey := "01020304"

	query := map[string]interface{}{
		"domain":      domain,
		"service":     service_name,
		"identityKey": identityKey,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	expectedQuery := types.SLAPQuery{
		Domain:      &domain,
		Service:     &service_name,
		IdentityKey: &identityKey,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindRecord", mock.Anything, expectedQuery).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

func TestLookup_ValidationError_NegativeLimit(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	query := map[string]interface{}{
		"limit": -1,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.limit must be a positive number")
}

func TestLookup_ValidationError_NegativeSkip(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	query := map[string]interface{}{
		"skip": -1,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.skip must be a non-negative number")
}

func TestLookup_ValidationError_InvalidSortOrder(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	query := map[string]interface{}{
		"sortOrder": "invalid",
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.sortOrder must be 'asc' or 'desc'")
}

// Test GetDocumentation

func TestGetDocumentation(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	doc := service.GetDocumentation()
	assert.Equal(t, LookupDocumentation, doc)
	assert.Contains(t, doc, "# SLAP Lookup Service")
	assert.Contains(t, doc, "Service Lookup Availability Protocol")
}

// Test GetMetaData

func TestGetMetaData(t *testing.T) {
	service, _ := createTestSLAPLookupService()

	metadata := service.GetMetaData()
	assert.Equal(t, "SLAP Lookup Service", metadata.Name)
	assert.Equal(t, "Provides lookup capabilities for SLAP tokens.", metadata.Description)
}

// Test edge cases and error scenarios

func TestLookup_StorageError(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   json.RawMessage(`"findAll"`),
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return([]types.UTXOReference{}, errors.New("storage error"))

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

func TestOutputAdmittedByTopic_StorageError(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	// Create valid PushDrop script with SLAP data
	fields := [][]byte{
		[]byte("SLAP"),                 // Protocol identifier
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("ls_treasury"),          // Service
	}
	validScriptHex := createValidPushDropScript(fields)
	scriptObj, err := script.NewFromHex(validScriptHex)
	require.NoError(t, err)

	// Create outpoint
	txidBytes, err := hex.DecodeString(TxId)
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	payload := &engine.OutputAdmittedByTopic{
		Topic:         SLAPTopic,
		Outpoint:      outpoint,
		LockingScript: scriptObj,
	}

	mockStorage.On("StoreSLAPRecord", mock.Anything, TxId, 0, "01020304", "https://example.com", "ls_treasury").Return(errors.New("storage error"))

	err = service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test complex query scenarios

func TestLookup_ComplexObjectQuery(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	domain := "https://example.com"
	service_name := "ls_treasury"
	identityKey := "deadbeef01020304"
	limit := 50
	skip := 10
	sortOrder := types.SortOrderDesc

	query := map[string]interface{}{
		"domain":      domain,
		"service":     service_name,
		"identityKey": identityKey,
		"limit":       limit,
		"skip":        skip,
		"sortOrder":   sortOrder,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	expectedQuery := types.SLAPQuery{
		Domain:      &domain,
		Service:     &service_name,
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
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
		assert.Len(t, utxos, 2)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	mockStorage.AssertExpectations(t)
}

// Test SLAP-specific scenarios

func TestLookup_ServiceOnlyQuery(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	service_name := "ls_treasury"

	query := map[string]interface{}{
		"service": service_name,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	expectedQuery := types.SLAPQuery{
		Service: &service_name,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
		{Txid: "def456", OutputIndex: 1},
		{Txid: "ghi789", OutputIndex: 0},
	}

	mockStorage.On("FindRecord", mock.Anything, expectedQuery).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	assert.Len(t, results.Result, 3)
	mockStorage.AssertExpectations(t)
}

func TestLookup_DomainOnlyQuery(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	domain := "https://example.com"

	query := map[string]interface{}{
		"domain": domain,
	}

	queryJSON, _ := json.Marshal(query)
	question := &lookup.LookupQuestion{
		Service: SLAPService,
		Query:   queryJSON,
	}

	expectedQuery := types.SLAPQuery{
		Domain: &domain,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindRecord", mock.Anything, expectedQuery).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, lookup.AnswerTypeFreeform, results.Type)
	if utxos, ok := results.Result.([]types.UTXOReference); ok {
		assert.Equal(t, expectedResults, utxos)
	} else {
		t.Errorf("Expected UTXOReference slice, got %T", results.Result)
	}
	assert.Len(t, results.Result, 1)
	mockStorage.AssertExpectations(t)
}

// Test different service types

func TestOutputAdmittedByTopic_DifferentServices(t *testing.T) {
	service, mockStorage := createTestSLAPLookupService()

	testCases := []struct {
		name        string
		serviceName string
	}{
		{"Treasury Service", "ls_treasury"},
		{"Bridge Service", "ls_bridge"},
		{"Custom Service", "ls_custom_service"},
		{"Token Service", "ls_token"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create valid PushDrop script with SLAP data
			fields := [][]byte{
				[]byte("SLAP"),
				[]byte{0x01, 0x02, 0x03, 0x04},
				[]byte("https://example.com"),
				[]byte(tc.serviceName),
			}
			validScriptHex := createValidPushDropScript(fields)
			scriptObj, err := script.NewFromHex(validScriptHex)
			require.NoError(t, err)

			// Create outpoint
			txidBytes, err := hex.DecodeString(TxId)
			require.NoError(t, err)
			var txidArray [32]byte
			copy(txidArray[:], txidBytes)

			outpoint := &transaction.Outpoint{
				Txid:  txidArray,
				Index: 0,
			}

			payload := &engine.OutputAdmittedByTopic{
				Topic:         SLAPTopic,
				Outpoint:      outpoint,
				LockingScript: scriptObj,
			}

			mockStorage.On("StoreSLAPRecord", mock.Anything, TxId, 0, "01020304", "https://example.com", tc.serviceName).Return(nil)

			err = service.OutputAdmittedByTopic(context.Background(), payload)
			assert.NoError(t, err)

			// Clear mocks for next iteration
			mockStorage.ExpectedCalls = nil
		})
	}
}

func TestSLAPLookupService_OutputNoLongerRetainedInHistory(t *testing.T) {
	mockStorage := &MockSLAPStorageInterface{}
	service := NewSLAPLookupService(mockStorage)

	// Create test outpoint
	outpoint := &transaction.Outpoint{
		Txid:  [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32},
		Index: 0,
	}

	// Test that OutputNoLongerRetainedInHistory does nothing (no-op)
	err := service.OutputNoLongerRetainedInHistory(context.Background(), outpoint, "tm_slap")
	assert.NoError(t, err)

	// Verify no storage methods were called
	mockStorage.AssertExpectations(t)
}

func TestSLAPLookupService_OutputBlockHeightUpdated(t *testing.T) {
	mockStorage := &MockSLAPStorageInterface{}
	service := NewSLAPLookupService(mockStorage)

	// Create test transaction ID
	txidBytes, err := hex.DecodeString("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	require.NoError(t, err)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)
	txid := &chainhash.Hash{}
	copy(txid[:], txidArray[:])

	// Test that OutputBlockHeightUpdated does nothing (no-op)
	err = service.OutputBlockHeightUpdated(context.Background(), txid, 12345, 0)
	assert.NoError(t, err)

	// Verify no storage methods were called
	mockStorage.AssertExpectations(t)
}
