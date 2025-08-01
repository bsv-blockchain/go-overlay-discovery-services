package ship

import (
	"context"
	"errors"
	"testing"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
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

// TestMockPushDropDecoder is a mock implementation of PushDropDecoder for testing
type TestMockPushDropDecoder struct {
	mock.Mock
}

func (m *TestMockPushDropDecoder) Decode(lockingScript string) (*types.PushDropResult, error) {
	args := m.Called(lockingScript)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.PushDropResult), args.Error(1)
}

// TestMockUtils is a mock implementation of Utils for testing
type TestMockUtils struct {
	mock.Mock
}

func (m *TestMockUtils) ToUTF8(data []byte) string {
	args := m.Called(data)
	return args.String(0)
}

func (m *TestMockUtils) ToHex(data []byte) string {
	args := m.Called(data)
	return args.String(0)
}

// Test helper functions

func createTestSHIPLookupService() (*SHIPLookupService, *MockSHIPStorageInterface, *TestMockPushDropDecoder, *TestMockUtils) {
	mockStorage := new(MockSHIPStorageInterface)
	mockPushDrop := new(TestMockPushDropDecoder)
	mockUtils := new(TestMockUtils)

	service := NewSHIPLookupService(mockStorage, mockPushDrop, mockUtils)

	return service, mockStorage, mockPushDrop, mockUtils
}

func createValidPushDropResult() *types.PushDropResult {
	return &types.PushDropResult{
		Fields: [][]byte{
			[]byte("SHIP"),                 // Protocol identifier
			[]byte{0x01, 0x02, 0x03, 0x04}, // Identity key bytes
			[]byte("https://example.com"),  // Domain
			[]byte("tm_bridge"),            // Topic
		},
	}
}

// Test NewSHIPLookupService

func TestNewSHIPLookupService(t *testing.T) {
	mockStorage := new(MockSHIPStorageInterface)
	mockPushDrop := new(TestMockPushDropDecoder)
	mockUtils := new(TestMockUtils)

	service := NewSHIPLookupService(mockStorage, mockPushDrop, mockUtils)

	assert.NotNil(t, service)
	assert.Equal(t, mockStorage, service.storage)
	assert.Equal(t, mockPushDrop, service.pushDropDecoder)
	assert.Equal(t, mockUtils, service.utils)
}

// Test OutputAdmittedByTopic

func TestOutputAdmittedByTopic_Success(t *testing.T) {
	service, mockStorage, mockPushDrop, mockUtils := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	pushDropResult := createValidPushDropResult()

	// Set up mocks
	mockPushDrop.On("Decode", "deadbeef").Return(pushDropResult, nil)
	mockUtils.On("ToUTF8", []byte("SHIP")).Return("SHIP")
	mockUtils.On("ToHex", []byte{0x01, 0x02, 0x03, 0x04}).Return("01020304")
	mockUtils.On("ToUTF8", []byte("https://example.com")).Return("https://example.com")
	mockUtils.On("ToUTF8", []byte("tm_bridge")).Return("tm_bridge")
	mockStorage.On("StoreSHIPRecord", mock.Anything, "abc123", 0, "01020304", "https://example.com", "tm_bridge").Return(nil)

	// Execute
	err := service.OutputAdmittedByTopic(context.Background(), payload)

	// Assert
	assert.NoError(t, err)
	mockPushDrop.AssertExpectations(t)
	mockUtils.AssertExpectations(t)
	mockStorage.AssertExpectations(t)
}

func TestOutputAdmittedByTopic_InvalidMode(t *testing.T) {
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, mockPushDrop, _ := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	mockPushDrop.On("Decode", "deadbeef").Return(nil, errors.New("decode error"))

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode PushDrop locking script")
}

func TestOutputAdmittedByTopic_InsufficientFields(t *testing.T) {
	service, _, mockPushDrop, _ := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	// Only 2 fields instead of required 4
	pushDropResult := &types.PushDropResult{
		Fields: [][]byte{
			[]byte("SHIP"),
			[]byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	mockPushDrop.On("Decode", "deadbeef").Return(pushDropResult, nil)

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected at least 4 fields, got 2")
}

func TestOutputAdmittedByTopic_IgnoreNonSHIPProtocol(t *testing.T) {
	service, _, mockPushDrop, mockUtils := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	pushDropResult := createValidPushDropResult()
	pushDropResult.Fields[0] = []byte("SLAP") // Different protocol

	mockPushDrop.On("Decode", "deadbeef").Return(pushDropResult, nil)
	mockUtils.On("ToUTF8", []byte("SLAP")).Return("SLAP")

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.NoError(t, err) // Should silently ignore non-SHIP protocols
}

// Test OutputSpent

func TestOutputSpent_Success(t *testing.T) {
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, mockStorage, _, _ := createTestSHIPLookupService()

	mockStorage.On("DeleteSHIPRecord", mock.Anything, "abc123", 0).Return(nil)

	err := service.OutputEvicted(context.Background(), "abc123", 0)
	assert.NoError(t, err)
	mockStorage.AssertExpectations(t)
}

func TestOutputEvicted_StorageError(t *testing.T) {
	service, mockStorage, _, _ := createTestSHIPLookupService()

	mockStorage.On("DeleteSHIPRecord", mock.Anything, "abc123", 0).Return(errors.New("storage error"))

	err := service.OutputEvicted(context.Background(), "abc123", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test Lookup

func TestLookup_LegacyFindAll(t *testing.T) {
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   nil,
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "a valid query must be provided")
}

func TestLookup_WrongService(t *testing.T) {
	service, _, _, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: "ls_other",
		Query:   "findAll",
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lookup service not supported")
}

func TestLookup_InvalidStringQuery(t *testing.T) {
	service, _, _, _ := createTestSHIPLookupService()

	question := types.LookupQuestion{
		Service: SHIPService,
		Query:   "invalid",
	}

	_, err := service.Lookup(context.Background(), question)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid string query: only 'findAll' is supported")
}

func TestLookup_ObjectQuery_FindAll(t *testing.T) {
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

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
	service, _, _, _ := createTestSHIPLookupService()

	doc, err := service.GetDocumentation()
	assert.NoError(t, err)
	assert.Equal(t, SHIPDocumentation, doc)
	assert.Contains(t, doc, "# SHIP Lookup Service")
	assert.Contains(t, doc, "Service Host Interconnect Protocol")
}

// Test GetMetaData

func TestGetMetaData(t *testing.T) {
	service, _, _, _ := createTestSHIPLookupService()

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
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
	service, mockStorage, mockPushDrop, mockUtils := createTestSHIPLookupService()

	payload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         SHIPTopic,
		LockingScript: "deadbeef",
		Txid:          "abc123",
		OutputIndex:   0,
	}

	pushDropResult := createValidPushDropResult()

	mockPushDrop.On("Decode", "deadbeef").Return(pushDropResult, nil)
	mockUtils.On("ToUTF8", []byte("SHIP")).Return("SHIP")
	mockUtils.On("ToHex", []byte{0x01, 0x02, 0x03, 0x04}).Return("01020304")
	mockUtils.On("ToUTF8", []byte("https://example.com")).Return("https://example.com")
	mockUtils.On("ToUTF8", []byte("tm_bridge")).Return("tm_bridge")
	mockStorage.On("StoreSHIPRecord", mock.Anything, "abc123", 0, "01020304", "https://example.com", "tm_bridge").Return(errors.New("storage error"))

	err := service.OutputAdmittedByTopic(context.Background(), payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage error")
}

// Test complex query scenarios

func TestLookup_ComplexObjectQuery(t *testing.T) {
	service, mockStorage, _, _ := createTestSHIPLookupService()

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
