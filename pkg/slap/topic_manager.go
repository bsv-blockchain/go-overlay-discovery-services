// Package slap implements the SLAP (Service Lookup Availability Protocol) topic manager functionality.
// This package provides Go equivalents for the TypeScript SLAPTopicManager class, enabling
// overlay network service subscription management and message routing for SLAP protocol.
package slap

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// ServiceSubscription represents an active service subscription for SLAP protocol
type ServiceSubscription struct {
	// Service is the name of the subscribed service
	Service string `json:"service"`
	// Domain is the domain associated with the service subscription
	Domain string `json:"domain"`
	// SubscribedAt is when the subscription was created
	SubscribedAt time.Time `json:"subscribedAt"`
	// IsActive indicates if the subscription is currently active
	IsActive bool `json:"isActive"`
	// MessageCount is the number of messages received for this service
	MessageCount int64 `json:"messageCount"`
}

// ServiceMessage represents a message received for a service subscription
type ServiceMessage struct {
	// Service is the service this message was received for
	Service string `json:"service"`
	// Domain is the domain associated with the service
	Domain string `json:"domain"`
	// Payload contains the message data
	Payload interface{} `json:"payload"`
	// ReceivedAt is when the message was received
	ReceivedAt time.Time `json:"receivedAt"`
	// MessageID is a unique identifier for this message
	MessageID string `json:"messageId"`
	// IdentityKey identifies the service provider (optional)
	IdentityKey string `json:"identityKey,omitempty"`
}

// ServiceMessageHandler is a function type for handling service messages
type ServiceMessageHandler func(ctx context.Context, message ServiceMessage) error

// SLAPTopicManagerInterface defines the interface for SLAP topic management operations
type SLAPTopicManagerInterface interface {
	// SubscribeToService subscribes to a specific service with a message handler
	SubscribeToService(ctx context.Context, service, domain string, handler ServiceMessageHandler) error
	// UnsubscribeFromService unsubscribes from a specific service
	UnsubscribeFromService(ctx context.Context, service, domain string) error
	// HandleServiceMessage processes an incoming service message
	HandleServiceMessage(ctx context.Context, message ServiceMessage) error
	// GetSubscribedServices returns all current service subscriptions
	GetSubscribedServices() []ServiceSubscription
	// CreateServiceSubscription creates a new service subscription
	CreateServiceSubscription(ctx context.Context, service, domain string) (*ServiceSubscription, error)
	// IsSubscribedToService checks if currently subscribed to a service
	IsSubscribedToService(service, domain string) bool
	// GetServiceMessageCount returns the message count for a specific service
	GetServiceMessageCount(service, domain string) int64
	// Close cleanly shuts down the topic manager
	Close(ctx context.Context) error
}

// SLAPTopicManager implements topic management functionality for SLAP protocol.
// It provides capabilities for subscribing to overlay network services, handling messages,
// and managing service lifecycle within the SLAP ecosystem.
type SLAPTopicManager struct {
	// subscriptions holds all active service subscriptions keyed by service+domain
	subscriptions map[string]*ServiceSubscription
	// handlers holds message handlers for each subscribed service
	handlers map[string]ServiceMessageHandler
	// mutex protects concurrent access to subscriptions and handlers
	mutex sync.RWMutex
	// storage provides access to SLAP storage operations
	storage SLAPStorageInterface
	// lookupService provides access to SLAP lookup operations (optional integration)
	lookupService *SLAPLookupService
}

// Compile-time verification that SLAPTopicManager implements SLAPTopicManagerInterface
var _ SLAPTopicManagerInterface = (*SLAPTopicManager)(nil)

// NewSLAPTopicManager creates a new SLAP topic manager instance.
// This constructor initializes the topic manager with the required dependencies
// for managing overlay network service subscriptions and message routing.
//
// Parameters:
//   - storage: The SLAP storage implementation for data persistence
//   - lookupService: Optional SLAP lookup service for integration (can be nil)
//
// Returns:
//   - *SLAPTopicManager: A new SLAP topic manager instance
func NewSLAPTopicManager(storage SLAPStorageInterface, lookupService *SLAPLookupService) *SLAPTopicManager {
	return &SLAPTopicManager{
		subscriptions: make(map[string]*ServiceSubscription),
		handlers:      make(map[string]ServiceMessageHandler),
		storage:       storage,
		lookupService: lookupService,
	}
}

// getSubscriptionKey creates a unique key for service+domain combination
func (tm *SLAPTopicManager) getSubscriptionKey(service, domain string) string {
	return fmt.Sprintf("%s@%s", service, domain)
}

// SubscribeToService subscribes to a specific service with a message handler.
// Creates a new subscription if one doesn't exist, or updates an existing one.
// The provided handler will be called for all messages received for this service.
//
// Parameters:
//   - ctx: Context for the operation
//   - service: The service name to subscribe to
//   - domain: The domain associated with the service
//   - handler: The message handler function to call for messages on this service
//
// Returns:
//   - error: An error if the subscription fails, nil otherwise
func (tm *SLAPTopicManager) SubscribeToService(ctx context.Context, service, domain string, handler ServiceMessageHandler) error {
	if service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("message handler cannot be nil")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)

	// Create or update subscription
	subscription, exists := tm.subscriptions[subscriptionKey]
	if !exists {
		subscription = &ServiceSubscription{
			Service:      service,
			Domain:       domain,
			SubscribedAt: time.Now(),
			IsActive:     true,
			MessageCount: 0,
		}
		tm.subscriptions[subscriptionKey] = subscription
	} else {
		// Reactivate existing subscription
		subscription.IsActive = true
	}

	// Set or update handler
	tm.handlers[subscriptionKey] = handler

	return nil
}

// UnsubscribeFromService unsubscribes from a specific service.
// Marks the subscription as inactive and removes the message handler.
// The subscription record is kept for historical purposes.
//
// Parameters:
//   - ctx: Context for the operation
//   - service: The service name to unsubscribe from
//   - domain: The domain associated with the service
//
// Returns:
//   - error: An error if the unsubscription fails, nil otherwise
func (tm *SLAPTopicManager) UnsubscribeFromService(ctx context.Context, service, domain string) error {
	if service == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)

	subscription, exists := tm.subscriptions[subscriptionKey]
	if !exists {
		return fmt.Errorf("not subscribed to service: %s@%s", service, domain)
	}

	// Mark subscription as inactive
	subscription.IsActive = false

	// Remove handler
	delete(tm.handlers, subscriptionKey)

	return nil
}

// HandleServiceMessage processes an incoming service message.
// Routes the message to the appropriate handler if one exists for the service.
// Updates message statistics for the service.
//
// Parameters:
//   - ctx: Context for the operation
//   - message: The service message to process
//
// Returns:
//   - error: An error if message handling fails, nil otherwise
func (tm *SLAPTopicManager) HandleServiceMessage(ctx context.Context, message ServiceMessage) error {
	if message.Service == "" {
		return fmt.Errorf("message service cannot be empty")
	}

	if message.Domain == "" {
		return fmt.Errorf("message domain cannot be empty")
	}

	subscriptionKey := tm.getSubscriptionKey(message.Service, message.Domain)

	tm.mutex.RLock()
	subscription, subscriptionExists := tm.subscriptions[subscriptionKey]
	handler, handlerExists := tm.handlers[subscriptionKey]
	tm.mutex.RUnlock()

	// Check if we have an active subscription for this service
	if !subscriptionExists || !subscription.IsActive {
		// Silently ignore messages for services we're not subscribed to
		return nil
	}

	if !handlerExists {
		return fmt.Errorf("no handler found for service: %s@%s", message.Service, message.Domain)
	}

	// Update message count
	tm.mutex.Lock()
	subscription.MessageCount++
	tm.mutex.Unlock()

	// Handle the message
	if err := handler(ctx, message); err != nil {
		return fmt.Errorf("failed to handle message for service %s@%s: %w", message.Service, message.Domain, err)
	}

	return nil
}

// GetSubscribedServices returns all current service subscriptions.
// Returns a copy of subscription data to prevent external modification.
//
// Returns:
//   - []ServiceSubscription: A slice of all service subscriptions
func (tm *SLAPTopicManager) GetSubscribedServices() []ServiceSubscription {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscriptions := make([]ServiceSubscription, 0, len(tm.subscriptions))
	for _, subscription := range tm.subscriptions {
		// Return a copy to prevent external modification
		subscriptions = append(subscriptions, *subscription)
	}

	return subscriptions
}

// CreateServiceSubscription creates a new service subscription without a handler.
// This method is useful for creating subscription records before setting up handlers.
//
// Parameters:
//   - ctx: Context for the operation
//   - service: The service name to create a subscription for
//   - domain: The domain associated with the service
//
// Returns:
//   - *ServiceSubscription: The created subscription
//   - error: An error if creation fails, nil otherwise
func (tm *SLAPTopicManager) CreateServiceSubscription(ctx context.Context, service, domain string) (*ServiceSubscription, error) {
	if service == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)

	// Check if subscription already exists
	if existing, exists := tm.subscriptions[subscriptionKey]; exists {
		// Return existing subscription
		return existing, nil
	}

	// Create new subscription
	subscription := &ServiceSubscription{
		Service:      service,
		Domain:       domain,
		SubscribedAt: time.Now(),
		IsActive:     false, // Not active until a handler is set
		MessageCount: 0,
	}

	tm.subscriptions[subscriptionKey] = subscription
	return subscription, nil
}

// IsSubscribedToService checks if currently subscribed to a service.
// Only returns true for active subscriptions.
//
// Parameters:
//   - service: The service name to check
//   - domain: The domain associated with the service
//
// Returns:
//   - bool: True if actively subscribed to the service, false otherwise
func (tm *SLAPTopicManager) IsSubscribedToService(service, domain string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)
	subscription, exists := tm.subscriptions[subscriptionKey]
	return exists && subscription.IsActive
}

// GetServiceMessageCount returns the message count for a specific service.
// Returns 0 if the service is not subscribed to.
//
// Parameters:
//   - service: The service name to get the message count for
//   - domain: The domain associated with the service
//
// Returns:
//   - int64: The number of messages received for this service
func (tm *SLAPTopicManager) GetServiceMessageCount(service, domain string) int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)
	if subscription, exists := tm.subscriptions[subscriptionKey]; exists {
		return subscription.MessageCount
	}
	return 0
}

// Close cleanly shuts down the topic manager.
// Unsubscribes from all services and cleans up resources.
//
// Parameters:
//   - ctx: Context for the shutdown operation
//
// Returns:
//   - error: An error if shutdown fails, nil otherwise
func (tm *SLAPTopicManager) Close(ctx context.Context) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Mark all subscriptions as inactive
	for _, subscription := range tm.subscriptions {
		subscription.IsActive = false
	}

	// Clear all handlers
	tm.handlers = make(map[string]ServiceMessageHandler)

	return nil
}

// GetTopicManagerMetaData returns metadata information for the SLAP topic manager.
// This provides basic information about the topic manager service.
//
// Returns:
//   - types.MetaData: The topic manager metadata
func (tm *SLAPTopicManager) GetTopicManagerMetaData() types.MetaData {
	return types.MetaData{
		Name:             "SLAP Topic Manager",
		ShortDescription: "Manages overlay network service subscriptions for SLAP protocol.",
	}
}

// GetActiveServiceCount returns the number of currently active service subscriptions.
//
// Returns:
//   - int: The number of active subscriptions
func (tm *SLAPTopicManager) GetActiveServiceCount() int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	count := 0
	for _, subscription := range tm.subscriptions {
		if subscription.IsActive {
			count++
		}
	}
	return count
}

// GetTotalMessageCount returns the total number of messages processed across all services.
//
// Returns:
//   - int64: The total message count
func (tm *SLAPTopicManager) GetTotalMessageCount() int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var total int64
	for _, subscription := range tm.subscriptions {
		total += subscription.MessageCount
	}
	return total
}

// GetServicesByDomain returns all active service subscriptions for a specific domain.
//
// Parameters:
//   - domain: The domain to filter by
//
// Returns:
//   - []ServiceSubscription: Service subscriptions for the specified domain
func (tm *SLAPTopicManager) GetServicesByDomain(domain string) []ServiceSubscription {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var domainServices []ServiceSubscription
	for _, subscription := range tm.subscriptions {
		if subscription.Domain == domain && subscription.IsActive {
			domainServices = append(domainServices, *subscription)
		}
	}

	return domainServices
}

// GetAvailableServices returns a list of unique service names that are currently subscribed to.
//
// Returns:
//   - []string: List of unique service names
func (tm *SLAPTopicManager) GetAvailableServices() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	serviceSet := make(map[string]bool)
	for _, subscription := range tm.subscriptions {
		if subscription.IsActive {
			serviceSet[subscription.Service] = true
		}
	}

	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}

	return services
}