package ship

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

// MockSHIPStorageInterface is a mock implementation of SHIPStorage interface methods
type MockSHIPStorageInterface struct {
	mock.Mock
}

func (m *MockSHIPStorageInterface) StoreSHIPRecord(ctx context.Context, txid string, outputIndex int, identityKey, domain, topic string) error {
	args := m.Called(ctx, txid, outputIndex, identityKey, domain, topic)
	return args.Error(0)
}

func (m *MockSHIPStorageInterface) DeleteSHIPRecord(ctx context.Context, txid string, outputIndex int) error {
	args := m.Called(ctx, txid, outputIndex)
	return args.Error(0)
}

func (m *MockSHIPStorageInterface) FindRecord(ctx context.Context, query types.SHIPQuery) ([]types.UTXOReference, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockSHIPStorageInterface) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	args := m.Called(ctx, limit, skip, sortOrder)
	return args.Get(0).([]types.UTXOReference), args.Error(1)
}

func (m *MockSHIPStorageInterface) EnsureIndexes(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Note: Mock PushDropDecoder and Utils are no longer needed since we use real implementations

// Test helper functions

func createTestSHIPLookupService() (*SHIPLookupService, *MockSHIPStorageInterface) {
	mockStorage := new(MockSHIPStorageInterface)
	service := NewSHIPLookupService(mockStorage)
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

// Test NewSHIPLookupService

func TestNewSHIPLookupService(t *testing.T) {
	mockStorage := new(MockSHIPStorageInterface)

	service := NewSHIPLookupService(mockStorage)

	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
}

// Test OutputAdmittedByTopic

func TestOutputAdmittedByTopic_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create valid PushDrop script with SHIP data
	fields := [][]byte{
		[]byte("SHIP"),                 // Protocol identifier
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("tm_bridge"),            // Topic
	}
	validScript := createValidPushDropScript(fields)

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: validScript,
		Txid:          "abc123",
		OutputIndex:   0,
	}

	// Set up mock for storage
	mockStorage.On("StoreSHIPRecord", mock.Anything, "abc123", 0, "01020304", "https://example.com", "tm_bridge").Return(nil)

	// Execute
	err := service.OutputAdmittedByTopic(context.Background(), payload)

	// Assert
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputAdmittedByTopic_InvalidMode(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          "invalid-mode",
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload: expected admission mode 'locking-script'")
}

func TestOutputAdmittedByTopic_IgnoreNonSHIPTopic(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         "tm_other",
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.NoError(t, err) // Should silently ignore non-SHIP topics
}

func TestOutputAdmittedByTopic_PushDropDecodeError(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef", // Invalid script that can't be decoded
		Txid:          "abc123",
		OutputIndex:   0,
	}

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PushDrop locking script")
}

func TestOutputAdmittedByTopic_InsufficientFields(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create PushDrop script with only 2 fields instead of required 4
	fields := [][]byte{
		[]byte("SHIP"),
		[]byte{0x01, 0x02, 0x03, 0x04},
	}
	invalidScript := createValidPushDropScript(fields)

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: invalidScript,
		Txid:          "abc123",
		OutputIndex:   0,
	}

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected at least 4 fields, got 2")
}

func TestOutputAdmittedByTopic_IgnoreNonSHIPProtocol(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	// Create valid PushDrop script with SLAP protocol instead of SHIP
	fields := [][]byte{
		[]byte("SLAP"),                 // Different protocol
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("tm_bridge"),            // Topic
	}
	validScript := createValidPushDropScript(fields)

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: validScript,
		Txid:          "abc123",
		OutputIndex:   0,
	}

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.NoError(t, err) // Should silently ignore non-SHIP protocols
}

// Test OutputSpent

func TestOutputSpent_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	payload := types.OutputSpent{
		Mode:        types.SpendNotificationModeNone,
		Topic:       SHIPTopic,
		Txid:        "abc123",
		OutputIndex: 0,
	}

	mockStorage.On("DeleteSHIPRecord", mock.Anything, "abc123", 0).Return(nil)

	err := service.OutputSpent(context.Background(), payload)
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputSpent_InvalidMode(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	payload := types.OutputSpent{
		Mode:        "invalid-mode",
		Topic:       SHIPTopic,
		Txid:        "abc123",
		OutputIndex: 0,
	}

	err := service.OutputSpent(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid payload: expected spend notification mode 'none'")
}

func TestOutputSpent_IgnoreNonSHIPTopic(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	payload := types.OutputSpent{
		Mode:        types.SpendNotificationModeNone,
		Topic:       "tm_other",
		Txid:        "abc123",
		OutputIndex: 0,
	}

	err := service.OutputSpent(context.Background(), payload)
	assert.NoError(t, err) // Should silently ignore non-SHIP topics
}

// Test OutputEvicted

func TestOutputEvicted_Success(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	mockStorage.On("DeleteSHIPRecord", mock.Anything, "abc123", 0).Return(nil)

	err := service.OutputEvicted(context.Background(), "abc123", 0)
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputEvicted_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	mockStorage.On("DeleteSHIPRecord", mock.Anything, "abc123", 0).Return(errors.New("storage error"))

	err := service.OutputEvicted(context.Background(), "abc123", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test Lookup

func TestLookup_LegacyFindAll(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   "findAll",
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
		{Txid: "def456", OutputIndex: 1},
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, types.LookupFormula(expectedResults), results)
	mockStorage.AssertExpectations(t)
}

func TestLookup_NilQuery(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   nil,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "a valid query must be provided")
}

func TestLookup_WrongService(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: "ls_other",
		Query:   "findAll",
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup service not supported")
}

func TestLookup_InvalidStringQuery(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   "invalid",
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
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

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
	}

	expectedResults := []types.UTXOReference{
		{Txid: "abc123", OutputIndex: 0},
	}

	mockStorage.On("FindAll", mock.Anything, &limit, &skip, &sortOrder).Return(expectedResults, nil)

	results, err := service.Lookup(context.Background(), question)
	assert.NoError(t, err)
	assert.Equal(t, types.LookupFormula(expectedResults), results)
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

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
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
	assert.NoError(t, err)
	assert.Equal(t, types.LookupFormula(expectedResults), results)
	mockStorage.AssertExpectations(t)
}

func TestLookup_ValidationError_NegativeLimit(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"limit": -1,
	}

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.limit must be a positive number")
}

func TestLookup_ValidationError_NegativeSkip(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"skip": -1,
	}

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.skip must be a non-negative number")
}

func TestLookup_ValidationError_InvalidSortOrder(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	query := map[string]interface{}{
		"sortOrder": "invalid",
	}

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "query.sortOrder must be 'asc' or 'desc'")
}

// Test GetDocumentation

func TestGetDocumentation(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	doc, err := service.GetDocumentation()
	assert.NoError(t, err)
	assert.Equal(t, LookupDocumentation, doc)
	assert.Contains(t, doc, "# SHIP Lookup Service")
	assert.Contains(t, doc, "Service Host Interconnect Protocol")
}

// Test GetMetaData

func TestGetMetaData(t *testing.T) {
	service, _ := createTestSHIPLookupService()

	metadata, err := service.GetMetaData()
	assert.NoError(t, err)
	assert.Equal(t, "SHIP Lookup Service", metadata.Name)
	assert.Equal(t, "Provides lookup capabilities for SHIP tokens.", metadata.ShortDescription)
	assert.Nil(t, metadata.IconURL)
	assert.Nil(t, metadata.Version)
	assert.Nil(t, metadata.InformationURL)
}

// Test edge cases and error scenarios

func TestLookup_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   "findAll",
	}

	mockStorage.On("FindAll", mock.Anything, (*int)(nil), (*int)(nil), (*types.SortOrder)(nil)).Return([]types.UTXOReference{}, errors.New("storage error"))

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

func TestOutputAdmittedByTopic_StorageError(t *testing.T) {
	service, mockStorage := createTestSHIPLookupService()

	// Create valid PushDrop script with SHIP data
	fields := [][]byte{
		[]byte("SHIP"),                 // Protocol identifier
		[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
		[]byte("https://example.com"),  // Domain
		[]byte("tm_bridge"),            // Topic
	}
	validScript := createValidPushDropScript(fields)

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: validScript,
		Txid:          "abc123",
		OutputIndex:   0,
	}

	mockStorage.On("StoreSHIPRecord", mock.Anything, "abc123", 0, "01020304", "https://example.com", "tm_bridge").Return(errors.New("storage error"))

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
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

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   query,
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
	assert.NoError(t, err)
	assert.Equal(t, types.LookupFormula(expectedResults), results)
	assert.Len(t, results, 2)
	mockStorage.AssertExpectations(t)
}
