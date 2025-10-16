package slap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static error variables for testing
var (
	errTestHandler = errors.New("handler error")
)

// Test helper functions

func createTestSLAPTopicManager() *TopicManager {
	mockStorage := new(MockStorage)

	topicManager := NewTopicManager(mockStorage, nil)

	return topicManager
}

func createTestSLAPTopicManagerWithLookupService() (*TopicManager, *MockStorage, *LookupService) {
	mockStorage := new(MockStorage)

	lookupService := NewLookupService(mockStorage)
	topicManager := NewTopicManager(mockStorage, lookupService)

	return topicManager, mockStorage, lookupService
}

func createTestServiceMessage(service, domain, messageID string, payload interface{}) ServiceMessage {
	return ServiceMessage{
		Service:     service,
		Domain:      domain,
		Payload:     payload,
		ReceivedAt:  time.Now(),
		MessageID:   messageID,
		IdentityKey: "test-identity-key",
	}
}

// Mock message handler for testing
func createMockServiceHandler(called *bool, shouldError bool) ServiceMessageHandler {
	return func(_ context.Context, _ ServiceMessage) error {
		*called = true
		if shouldError {
			return errTestHandler
		}
		return nil
	}
}

// Test NewTopicManager

func TestNewSLAPTopicManager(t *testing.T) {
	mockStorage := new(MockStorage)

	topicManager := NewTopicManager(mockStorage, nil)

	assert.NotNil(t, topicManager)
	assert.Equal(t, mockStorage, topicManager.storage)
	assert.Nil(t, topicManager.lookupService)
	assert.NotNil(t, topicManager.subscriptions)
	assert.NotNil(t, topicManager.handlers)
	assert.Empty(t, topicManager.subscriptions)
	assert.Empty(t, topicManager.handlers)
}

func TestNewSLAPTopicManagerWithLookupService(t *testing.T) {
	topicManager, mockStorage, lookupService := createTestSLAPTopicManagerWithLookupService()

	assert.NotNil(t, topicManager)
	assert.Equal(t, mockStorage, topicManager.storage)
	assert.Equal(t, lookupService, topicManager.lookupService)
	assert.NotNil(t, topicManager.subscriptions)
	assert.NotNil(t, topicManager.handlers)
}

// Test SubscribeToService

func TestSubscribeToService_Success(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)

	require.NoError(t, err)
	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
	assert.Len(t, topicManager.subscriptions, 1)
	assert.Len(t, topicManager.handlers, 1)

	// Check subscription details
	subscriptions := topicManager.GetSubscribedServices()
	assert.Len(t, subscriptions, 1)
	assert.Equal(t, "ls_treasury", subscriptions[0].Service)
	assert.Equal(t, "example.com", subscriptions[0].Domain)
	assert.True(t, subscriptions[0].IsActive)
	assert.Equal(t, int64(0), subscriptions[0].MessageCount)
}

func TestSubscribeToService_EmptyService(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "", "example.com", handler)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service name cannot be empty")
	assert.False(t, topicManager.IsSubscribedToService("", "example.com"))
}

func TestSubscribeToService_EmptyDomain(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "", handler)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain cannot be empty")
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", ""))
}

func TestSubscribeToService_NilHandler(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message handler cannot be nil")
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
}

func TestSubscribeToService_UpdateExistingSubscription(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Create initial subscription
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	// Unsubscribe
	err = topicManager.UnsubscribeFromService(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Resubscribe with new handler
	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	err = topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler2)
	require.NoError(t, err)
	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Should still have only one subscription
	assert.Len(t, topicManager.subscriptions, 1)
	assert.Len(t, topicManager.handlers, 1)
}

// Test UnsubscribeFromService

func TestUnsubscribeFromService_Success(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// First subscribe
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)
	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Then unsubscribe
	err = topicManager.UnsubscribeFromService(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Subscription should still exist but be inactive
	assert.Len(t, topicManager.subscriptions, 1)
	assert.Empty(t, topicManager.handlers)

	subscriptions := topicManager.GetSubscribedServices()
	assert.Len(t, subscriptions, 1)
	assert.False(t, subscriptions[0].IsActive)
}

func TestUnsubscribeFromService_EmptyService(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	err := topicManager.UnsubscribeFromService(context.Background(), "", "example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "service name cannot be empty")
}

func TestUnsubscribeFromService_EmptyDomain(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	err := topicManager.UnsubscribeFromService(context.Background(), "ls_treasury", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "domain cannot be empty")
}

func TestUnsubscribeFromService_NotSubscribed(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	err := topicManager.UnsubscribeFromService(context.Background(), "ls_nonexistent", "example.com")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not subscribed to service: ls_nonexistent@example.com")
}

// Test HandleServiceMessage

func TestHandleServiceMessage_Success(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe to service
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	// Handle message
	message := createTestServiceMessage("ls_treasury", "example.com", "msg-1", "test payload")
	err = topicManager.HandleServiceMessage(context.Background(), message)

	require.NoError(t, err)
	assert.True(t, handlerCalled)
	assert.Equal(t, int64(1), topicManager.GetServiceMessageCount("ls_treasury", "example.com"))
}

func TestHandleServiceMessage_EmptyService(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	message := createTestServiceMessage("", "example.com", "msg-1", "test payload")
	err := topicManager.HandleServiceMessage(context.Background(), message)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message service cannot be empty")
}

func TestHandleServiceMessage_EmptyDomain(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	message := createTestServiceMessage("ls_treasury", "", "msg-1", "test payload")
	err := topicManager.HandleServiceMessage(context.Background(), message)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message domain cannot be empty")
}

func TestHandleServiceMessage_NotSubscribed(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	message := createTestServiceMessage("ls_nonexistent", "example.com", "msg-1", "test payload")
	err := topicManager.HandleServiceMessage(context.Background(), message)

	// Should silently ignore messages for services we're not subscribed to
	require.NoError(t, err)
}

func TestHandleServiceMessage_InactiveSubscription(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe and then unsubscribe
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	err = topicManager.UnsubscribeFromService(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)

	// Try to handle message
	message := createTestServiceMessage("ls_treasury", "example.com", "msg-1", "test payload")
	err = topicManager.HandleServiceMessage(context.Background(), message)

	// Should silently ignore inactive subscriptions
	require.NoError(t, err)
	assert.False(t, handlerCalled)
}

func TestHandleServiceMessage_HandlerError(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe with error handler
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, true)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	// Handle message
	message := createTestServiceMessage("ls_treasury", "example.com", "msg-1", "test payload")
	err = topicManager.HandleServiceMessage(context.Background(), message)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to handle message for service ls_treasury@example.com")
	assert.True(t, handlerCalled)

	// Message count should still be incremented
	assert.Equal(t, int64(1), topicManager.GetServiceMessageCount("ls_treasury", "example.com"))
}

// Test CreateServiceSubscription

func TestCreateServiceSubscription_Success(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	subscription, err := topicManager.CreateServiceSubscription(context.Background(), "ls_treasury", "example.com")

	require.NoError(t, err)
	assert.NotNil(t, subscription)
	assert.Equal(t, "ls_treasury", subscription.Service)
	assert.Equal(t, "example.com", subscription.Domain)
	assert.False(t, subscription.IsActive) // Should not be active without handler
	assert.Equal(t, int64(0), subscription.MessageCount)
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Should exist in subscriptions
	assert.Len(t, topicManager.subscriptions, 1)
}

func TestCreateServiceSubscription_EmptyService(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	subscription, err := topicManager.CreateServiceSubscription(context.Background(), "", "example.com")

	require.Error(t, err)
	assert.Nil(t, subscription)
	assert.Contains(t, err.Error(), "service name cannot be empty")
}

func TestCreateServiceSubscription_EmptyDomain(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	subscription, err := topicManager.CreateServiceSubscription(context.Background(), "ls_treasury", "")

	require.Error(t, err)
	assert.Nil(t, subscription)
	assert.Contains(t, err.Error(), "domain cannot be empty")
}

func TestCreateServiceSubscription_ExistingSubscription(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Create first subscription
	subscription1, err := topicManager.CreateServiceSubscription(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)

	// Try to create again
	subscription2, err := topicManager.CreateServiceSubscription(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)

	// Should return the same subscription
	assert.Equal(t, subscription1, subscription2)
	assert.Len(t, topicManager.subscriptions, 1)
}

// Test GetSubscribedServices

func TestGetSubscribedServices_Empty(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	subscriptions := topicManager.GetSubscribedServices()

	assert.NotNil(t, subscriptions)
	assert.Empty(t, subscriptions)
}

func TestGetSubscribedServices_Multiple(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Create multiple subscriptions
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler2)
	require.NoError(t, err)

	// Create an inactive subscription
	_, err = topicManager.CreateServiceSubscription(context.Background(), "ls_storage", "storage.com")
	require.NoError(t, err)

	subscriptions := topicManager.GetSubscribedServices()

	assert.Len(t, subscriptions, 3)

	// Count active and inactive
	activeCount := 0
	inactiveCount := 0
	for _, sub := range subscriptions {
		if sub.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	assert.Equal(t, 2, activeCount)
	assert.Equal(t, 1, inactiveCount)
}

// Test IsSubscribedToService

func TestIsSubscribedToService_True(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
}

func TestIsSubscribedToService_False(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	assert.False(t, topicManager.IsSubscribedToService("ls_nonexistent", "example.com"))
}

func TestIsSubscribedToService_Inactive(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Create inactive subscription
	_, err := topicManager.CreateServiceSubscription(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)

	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
}

// Test GetServiceMessageCount

func TestGetServiceMessageCount_Zero(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	count := topicManager.GetServiceMessageCount("ls_nonexistent", "example.com")
	assert.Equal(t, int64(0), count)
}

func TestGetServiceMessageCount_WithMessages(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe and handle messages
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	// Handle multiple messages
	for i := 0; i < 5; i++ {
		message := createTestServiceMessage("ls_treasury", "example.com", fmt.Sprintf("msg-%d", i), "test payload")
		err = topicManager.HandleServiceMessage(context.Background(), message)
		require.NoError(t, err)
	}

	count := topicManager.GetServiceMessageCount("ls_treasury", "example.com")
	assert.Equal(t, int64(5), count)
}

// Test Close

func TestClose_Success(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Create multiple subscriptions
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler2)
	require.NoError(t, err)

	// Close the topic manager
	err = topicManager.Close(context.Background())
	require.NoError(t, err)

	// All subscriptions should be inactive
	assert.False(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
	assert.False(t, topicManager.IsSubscribedToService("ls_bank", "bank.com"))

	// All handlers should be cleared
	assert.Empty(t, topicManager.handlers)

	// Subscriptions should still exist but be inactive
	subscriptions := topicManager.GetSubscribedServices()
	assert.Len(t, subscriptions, 2)
	for _, sub := range subscriptions {
		assert.False(t, sub.IsActive)
	}
}

// Test metadata and statistics methods

func TestGetTopicManagerMetaData(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	metadata := topicManager.GetTopicManagerMetaData()

	assert.Equal(t, "SLAP Topic Manager", metadata.Name)
	assert.Equal(t, "Manages overlay network service subscriptions for SLAP protocol.", metadata.Description)
}

func TestGetActiveServiceCount(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Initially should be 0
	assert.Equal(t, 0, topicManager.GetActiveServiceCount())

	// Add active subscriptions
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler2)
	require.NoError(t, err)

	assert.Equal(t, 2, topicManager.GetActiveServiceCount())

	// Add inactive subscription
	_, err = topicManager.CreateServiceSubscription(context.Background(), "ls_storage", "storage.com")
	require.NoError(t, err)

	assert.Equal(t, 2, topicManager.GetActiveServiceCount())

	// Unsubscribe from one
	err = topicManager.UnsubscribeFromService(context.Background(), "ls_treasury", "example.com")
	require.NoError(t, err)

	assert.Equal(t, 1, topicManager.GetActiveServiceCount())
}

func TestGetTotalMessageCount(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Initially should be 0
	assert.Equal(t, int64(0), topicManager.GetTotalMessageCount())

	// Subscribe to services
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler2)
	require.NoError(t, err)

	// Handle messages on both services
	for i := 0; i < 3; i++ {
		message := createTestServiceMessage("ls_treasury", "example.com", fmt.Sprintf("msg-%d", i), "payload")
		err = topicManager.HandleServiceMessage(context.Background(), message)
		require.NoError(t, err)
	}

	for i := 0; i < 2; i++ {
		message := createTestServiceMessage("ls_bank", "bank.com", fmt.Sprintf("msg-%d", i), "payload")
		err = topicManager.HandleServiceMessage(context.Background(), message)
		require.NoError(t, err)
	}

	assert.Equal(t, int64(5), topicManager.GetTotalMessageCount())
}

// Test SLAP-specific methods

func TestGetServicesByDomain(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe to services in different domains
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	handlerCalled3 := false
	handler3 := createMockServiceHandler(&handlerCalled3, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "example.com", handler2)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_storage", "storage.com", handler3)
	require.NoError(t, err)

	// Get services for example.com
	exampleServices := topicManager.GetServicesByDomain("example.com")
	assert.Len(t, exampleServices, 2)

	serviceNames := make([]string, len(exampleServices))
	for i, service := range exampleServices {
		serviceNames[i] = service.Service
		assert.Equal(t, "example.com", service.Domain)
		assert.True(t, service.IsActive)
	}

	assert.Contains(t, serviceNames, "ls_treasury")
	assert.Contains(t, serviceNames, "ls_bank")

	// Get services for storage.com
	storageServices := topicManager.GetServicesByDomain("storage.com")
	assert.Len(t, storageServices, 1)
	assert.Equal(t, "ls_storage", storageServices[0].Service)

	// Get services for non-existent domain
	nonExistentServices := topicManager.GetServicesByDomain("nonexistent.com")
	assert.Empty(t, nonExistentServices)
}

func TestGetAvailableServices(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Initially should be empty
	services := topicManager.GetAvailableServices()
	assert.Empty(t, services)

	// Subscribe to services
	handlerCalled1 := false
	handler1 := createMockServiceHandler(&handlerCalled1, false)

	handlerCalled2 := false
	handler2 := createMockServiceHandler(&handlerCalled2, false)

	handlerCalled3 := false
	handler3 := createMockServiceHandler(&handlerCalled3, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler1)
	require.NoError(t, err)

	err = topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler2)
	require.NoError(t, err)

	// Subscribe to same service on different domain
	err = topicManager.SubscribeToService(context.Background(), "ls_treasury", "treasury.com", handler3)
	require.NoError(t, err)

	services = topicManager.GetAvailableServices()
	assert.Len(t, services, 2) // Should be unique service names

	assert.Contains(t, services, "ls_treasury")
	assert.Contains(t, services, "ls_bank")

	// Add inactive subscription - should not be included
	_, err = topicManager.CreateServiceSubscription(context.Background(), "ls_storage", "storage.com")
	require.NoError(t, err)

	services = topicManager.GetAvailableServices()
	assert.Len(t, services, 2) // Should still be 2
}

// Test concurrent access scenarios

func TestConcurrentServiceSubscription(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Test concurrent subscription to different services
	done := make(chan bool, 2)

	go func() {
		handlerCalled := false
		handler := createMockServiceHandler(&handlerCalled, false)
		err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
		assert.NoError(t, err)
		done <- true
	}()

	go func() {
		handlerCalled := false
		handler := createMockServiceHandler(&handlerCalled, false)
		err := topicManager.SubscribeToService(context.Background(), "ls_bank", "bank.com", handler)
		assert.NoError(t, err)
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	assert.Equal(t, 2, topicManager.GetActiveServiceCount())
	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))
	assert.True(t, topicManager.IsSubscribedToService("ls_bank", "bank.com"))
}

func TestConcurrentServiceMessageHandling(t *testing.T) {
	topicManager := createTestSLAPTopicManager()

	// Subscribe to service with a thread-safe handler
	handler := func(_ context.Context, _ ServiceMessage) error {
		return nil
	}

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)

	// Handle messages concurrently
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func(messageID int) {
			message := createTestServiceMessage("ls_treasury", "example.com", fmt.Sprintf("msg-%d", messageID), "payload")
			err := topicManager.HandleServiceMessage(context.Background(), message)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	assert.Equal(t, int64(5), topicManager.GetServiceMessageCount("ls_treasury", "example.com"))
}

// Integration test with lookup service

func TestIntegrationWithLookupService(t *testing.T) {
	topicManager, mockStorage, lookupService := createTestSLAPTopicManagerWithLookupService()

	// Verify integration
	assert.NotNil(t, topicManager.lookupService)
	assert.Equal(t, lookupService, topicManager.lookupService)

	// Topic manager should still work normally
	handlerCalled := false
	handler := createMockServiceHandler(&handlerCalled, false)

	err := topicManager.SubscribeToService(context.Background(), "ls_treasury", "example.com", handler)
	require.NoError(t, err)
	assert.True(t, topicManager.IsSubscribedToService("ls_treasury", "example.com"))

	// Storage should be the same instance
	assert.Equal(t, mockStorage, topicManager.storage)
}
