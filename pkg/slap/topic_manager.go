// Package slap implements the SLAP (Service Lookup Availability Protocol) topic manager functionality.
// Overlay network service subscription management and message routing for SLAP protocol.
package slap

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
)

// Static error variables for err113 compliance
var (
	errServiceNameEmpty         = errors.New("service name cannot be empty")
	errDomainEmpty              = errors.New("domain cannot be empty")
	errMessageHandlerNil        = errors.New("message handler cannot be nil")
	errNotSubscribedToService   = errors.New("not subscribed to service")
	errMessageServiceEmpty      = errors.New("message service cannot be empty")
	errMessageDomainEmpty       = errors.New("message domain cannot be empty")
	errNoHandlerFoundForService = errors.New("no handler found for service")
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

// TopicManager implements topic management functionality for SLAP protocol.
// It provides capabilities for subscribing to overlay network services, handling messages,
// and managing service lifecycle within the SLAP ecosystem.
type TopicManager struct {
	// subscriptions holds all active service subscriptions keyed by service+domain
	subscriptions map[string]*ServiceSubscription
	// handlers holds message handlers for each subscribed service
	handlers map[string]ServiceMessageHandler
	// mutex protects concurrent access to subscriptions and handlers
	mutex sync.RWMutex
	// storage provides access to SLAP storage operations
	storage StorageInterface
	// lookupService provides access to SLAP lookup operations (optional integration)
	lookupService *LookupService
}

// NewTopicManager creates a new SLAP topic manager instance.
// This constructor initializes the topic manager with the required dependencies
// for managing overlay network service subscriptions and message routing.
func NewTopicManager(storage StorageInterface, lookupService *LookupService) *TopicManager {
	return &TopicManager{
		subscriptions: make(map[string]*ServiceSubscription),
		handlers:      make(map[string]ServiceMessageHandler),
		storage:       storage,
		lookupService: lookupService,
	}
}

// getSubscriptionKey creates a unique key for service+domain combination
func (tm *TopicManager) getSubscriptionKey(service, domain string) string {
	return fmt.Sprintf("%s@%s", service, domain)
}

// SubscribeToService subscribes to a specific service with a message handler.
// Creates a new subscription if one doesn't exist, or updates an existing one.
// The provided handler will be called for all messages received for this service.
func (tm *TopicManager) SubscribeToService(_ context.Context, service, domain string, handler ServiceMessageHandler) error {
	if service == "" {
		return errServiceNameEmpty
	}

	if domain == "" {
		return errDomainEmpty
	}

	if handler == nil {
		return errMessageHandlerNil
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
func (tm *TopicManager) UnsubscribeFromService(_ context.Context, service, domain string) error {
	if service == "" {
		return errServiceNameEmpty
	}

	if domain == "" {
		return errDomainEmpty
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)

	subscription, exists := tm.subscriptions[subscriptionKey]
	if !exists {
		return fmt.Errorf("%w: %s@%s", errNotSubscribedToService, service, domain)
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
func (tm *TopicManager) HandleServiceMessage(ctx context.Context, message ServiceMessage) error {
	if message.Service == "" {
		return errMessageServiceEmpty
	}

	if message.Domain == "" {
		return errMessageDomainEmpty
	}

	subscriptionKey := tm.getSubscriptionKey(message.Service, message.Domain)

	tm.mutex.RLock()
	subscription, subscriptionExists := tm.subscriptions[subscriptionKey]
	handler, handlerExists := tm.handlers[subscriptionKey]
	isActive := subscriptionExists && subscription.IsActive
	tm.mutex.RUnlock()

	// Check if we have an active subscription for this service
	if !subscriptionExists || !isActive {
		// Silently ignore messages for services we're not subscribed to
		return nil
	}

	if !handlerExists {
		return fmt.Errorf("%w: %s@%s", errNoHandlerFoundForService, message.Service, message.Domain)
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
func (tm *TopicManager) GetSubscribedServices() []ServiceSubscription {
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
func (tm *TopicManager) CreateServiceSubscription(_ context.Context, service, domain string) (*ServiceSubscription, error) {
	if service == "" {
		return nil, errServiceNameEmpty
	}

	if domain == "" {
		return nil, errDomainEmpty
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
func (tm *TopicManager) IsSubscribedToService(service, domain string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscriptionKey := tm.getSubscriptionKey(service, domain)
	subscription, exists := tm.subscriptions[subscriptionKey]
	return exists && subscription.IsActive
}

// GetServiceMessageCount returns the message count for a specific service.
// Returns 0 if the service is not subscribed to.
func (tm *TopicManager) GetServiceMessageCount(service, domain string) int64 {
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
func (tm *TopicManager) Close(_ context.Context) error {
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
func (tm *TopicManager) GetTopicManagerMetaData() overlay.MetaData {
	return overlay.MetaData{
		Name:        "SLAP Topic Manager",
		Description: "Manages overlay network service subscriptions for SLAP protocol.",
	}
}

// GetActiveServiceCount returns the number of currently active service subscriptions.
func (tm *TopicManager) GetActiveServiceCount() int {
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
func (tm *TopicManager) GetTotalMessageCount() int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var total int64
	for _, subscription := range tm.subscriptions {
		total += subscription.MessageCount
	}
	return total
}

// GetServicesByDomain returns all active service subscriptions for a specific domain.
func (tm *TopicManager) GetServicesByDomain(domain string) []ServiceSubscription {
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
func (tm *TopicManager) GetAvailableServices() []string {
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

// IdentifyAdmissibleOutputs implements the engine.TopicManager interface
// For SLAP, this identifies outputs that should be admitted to the overlay
func (tm *TopicManager) IdentifyAdmissibleOutputs(ctx context.Context, beef []byte, previousCoins map[uint32]*transaction.TransactionOutput) (overlay.AdmittanceInstructions, error) {
	outputsToAdmit := []uint32{}

	// Parse transaction from BEEF format
	parsedTransaction, err := transaction.NewTransactionFromBEEF(beef)
	if err != nil {
		// Only log error if no outputs were admitted and no previous coins consumed
		if len(outputsToAdmit) == 0 && len(previousCoins) == 0 {
			log.Printf("🤚 Error identifying admissible outputs: %v", err)
		}
		return overlay.AdmittanceInstructions{
			OutputsToAdmit: outputsToAdmit,
			CoinsToRetain:  []uint32{},
		}, nil
	}

	// Check each output for SLAP token validity
	for i, output := range parsedTransaction.Outputs {
		// Decode PushDrop token
		result := pushdrop.Decode(output.LockingScript)
		if result == nil {
			continue // It's common for other outputs to be invalid; no need to log an error here
		}

		// SLAP tokens must have exactly 5 fields
		if len(result.Fields) != 5 {
			continue
		}

		// Check SLAP identifier (first field)
		slapIdentifier := utils.UTFBytesToString(result.Fields[0])
		if slapIdentifier != "SLAP" {
			continue
		}

		// Check advertised URI (third field)
		advertisedURI := utils.UTFBytesToString(result.Fields[2])
		if !utils.IsAdvertisableURI(advertisedURI) {
			continue
		}

		// Check topic name (fourth field)
		topic := utils.UTFBytesToString(result.Fields[3])
		if !utils.IsValidTopicOrServiceName(topic) {
			continue
		}

		// SLAP only accepts "ls_" (lookup service) advertisements
		if !strings.HasPrefix(topic, "ls_") {
			continue
		}

		// Check token signature linkage
		lockingPublicKey := result.LockingPublicKey.ToDERHex()
		tokenFields := make(utils.TokenFields, len(result.Fields))
		copy(tokenFields, result.Fields)

		// For now, use mock wallet - in production this should be the real wallet
		if valid, err := utils.IsTokenSignatureCorrectlyLinked(ctx, lockingPublicKey, tokenFields); err == nil && valid {
			if i >= 0 && i <= 0xFFFFFFFF {
				outputsToAdmit = append(outputsToAdmit, uint32(i))
			}
		} else if err == nil && !valid {
			slog.Info("Invalid token signature linkage", "outputIndex", i, "txid", parsedTransaction.TxID())
		}
	}

	// Friendly logging with slappy emojis
	if len(outputsToAdmit) > 0 {
		if len(outputsToAdmit) == 1 {
			log.Printf("👏 Admitted %d SLAP output!", len(outputsToAdmit))
		} else {
			log.Printf("👏 Admitted %d SLAP outputs!", len(outputsToAdmit))
		}
	}

	if len(previousCoins) > 0 {
		if len(previousCoins) == 1 {
			log.Printf("✋ Consumed %d previous SLAP coin!", len(previousCoins))
		} else {
			log.Printf("✋ Consumed %d previous SLAP coins!", len(previousCoins))
		}
	}

	if len(outputsToAdmit) == 0 && len(previousCoins) == 0 {
		log.Printf("😕 No SLAP outputs admitted and no previous SLAP coins consumed.")
	}

	return overlay.AdmittanceInstructions{
		OutputsToAdmit: outputsToAdmit,
		CoinsToRetain:  []uint32{},
	}, nil
}

// IdentifyNeededInputs implements the engine.TopicManager interface
// For SLAP, this identifies inputs needed for validation
func (tm *TopicManager) IdentifyNeededInputs(_ context.Context, _ []byte) ([]*transaction.Outpoint, error) {
	// SLAP doesn't require specific inputs for validation
	return []*transaction.Outpoint{}, nil
}

// GetDocumentation implements the engine.TopicManager interface
// Returns documentation for the SLAP topic manager
func (tm *TopicManager) GetDocumentation() string {
	return TopicManagerDocumentation
}

// GetMetaData implements the engine.TopicManager interface
// Returns metadata about the SLAP topic manager
func (tm *TopicManager) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SLAP Topic Manager",
		Description: "Manages SLAP protocol topics for service lookup and availability tracking",
	}
}
