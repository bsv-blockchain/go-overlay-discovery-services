// Package ship implements the SHIP (Service Host Interconnect Protocol) topic manager functionality.
// This package provides Go equivalents for the TypeScript SHIPTopicManager class, enabling
// overlay network topic management and message routing for SHIP protocol.
package ship

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// TopicSubscription represents an active topic subscription
type TopicSubscription struct {
	// Topic is the name of the subscribed topic
	Topic string `json:"topic"`
	// SubscribedAt is when the subscription was created
	SubscribedAt time.Time `json:"subscribedAt"`
	// IsActive indicates if the subscription is currently active
	IsActive bool `json:"isActive"`
	// MessageCount is the number of messages received on this topic
	MessageCount int64 `json:"messageCount"`
}

// TopicMessage represents a message received on a topic
type TopicMessage struct {
	// Topic is the topic this message was received on
	Topic string `json:"topic"`
	// Payload contains the message data
	Payload interface{} `json:"payload"`
	// ReceivedAt is when the message was received
	ReceivedAt time.Time `json:"receivedAt"`
	// MessageID is a unique identifier for this message
	MessageID string `json:"messageId"`
}

// TopicMessageHandler is a function type for handling topic messages
type TopicMessageHandler func(ctx context.Context, message TopicMessage) error

// SHIPTopicManagerInterface defines the interface for SHIP topic management operations
type SHIPTopicManagerInterface interface {
	// SubscribeToTopic subscribes to a specific topic with a message handler
	SubscribeToTopic(ctx context.Context, topic string, handler TopicMessageHandler) error
	// UnsubscribeFromTopic unsubscribes from a specific topic
	UnsubscribeFromTopic(ctx context.Context, topic string) error
	// HandleTopicMessage processes an incoming topic message
	HandleTopicMessage(ctx context.Context, message TopicMessage) error
	// GetSubscribedTopics returns all current topic subscriptions
	GetSubscribedTopics() []TopicSubscription
	// CreateTopicSubscription creates a new topic subscription
	CreateTopicSubscription(ctx context.Context, topic string) (*TopicSubscription, error)
	// IsSubscribedToTopic checks if currently subscribed to a topic
	IsSubscribedToTopic(topic string) bool
	// GetTopicMessageCount returns the message count for a specific topic
	GetTopicMessageCount(topic string) int64
	// Close cleanly shuts down the topic manager
	Close(ctx context.Context) error
}

// SHIPTopicManager implements topic management functionality for SHIP protocol.
// It provides capabilities for subscribing to overlay network topics, handling messages,
// and managing topic lifecycle within the SHIP ecosystem.
type SHIPTopicManager struct {
	// subscriptions holds all active topic subscriptions
	subscriptions map[string]*TopicSubscription
	// handlers holds message handlers for each subscribed topic
	handlers map[string]TopicMessageHandler
	// mutex protects concurrent access to subscriptions and handlers
	mutex sync.RWMutex
	// storage provides access to SHIP storage operations
	storage SHIPStorageInterface
	// lookupService provides access to SHIP lookup operations (optional integration)
	lookupService *SHIPLookupService
}

// Compile-time verification that SHIPTopicManager implements SHIPTopicManagerInterface
var _ SHIPTopicManagerInterface = (*SHIPTopicManager)(nil)

// NewSHIPTopicManager creates a new SHIP topic manager instance.
// This constructor initializes the topic manager with the required dependencies
// for managing overlay network topic subscriptions and message routing.
//
// Parameters:
//   - storage: The SHIP storage implementation for data persistence
//   - lookupService: Optional SHIP lookup service for integration (can be nil)
//
// Returns:
//   - *SHIPTopicManager: A new SHIP topic manager instance
func NewSHIPTopicManager(storage SHIPStorageInterface, lookupService *SHIPLookupService) *SHIPTopicManager {
	return &SHIPTopicManager{
		subscriptions: make(map[string]*TopicSubscription),
		handlers:      make(map[string]TopicMessageHandler),
		storage:       storage,
		lookupService: lookupService,
	}
}

// SubscribeToTopic subscribes to a specific topic with a message handler.
// Creates a new subscription if one doesn't exist, or updates an existing one.
// The provided handler will be called for all messages received on this topic.
//
// Parameters:
//   - ctx: Context for the operation
//   - topic: The topic name to subscribe to
//   - handler: The message handler function to call for messages on this topic
//
// Returns:
//   - error: An error if the subscription fails, nil otherwise
func (tm *SHIPTopicManager) SubscribeToTopic(ctx context.Context, topic string, handler TopicMessageHandler) error {
	if topic == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	if handler == nil {
		return fmt.Errorf("message handler cannot be nil")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Create or update subscription
	subscription, exists := tm.subscriptions[topic]
	if !exists {
		subscription = &TopicSubscription{
			Topic:        topic,
			SubscribedAt: time.Now(),
			IsActive:     true,
			MessageCount: 0,
		}
		tm.subscriptions[topic] = subscription
	} else {
		// Reactivate existing subscription
		subscription.IsActive = true
	}

	// Set or update handler
	tm.handlers[topic] = handler

	return nil
}

// UnsubscribeFromTopic unsubscribes from a specific topic.
// Marks the subscription as inactive and removes the message handler.
// The subscription record is kept for historical purposes.
//
// Parameters:
//   - ctx: Context for the operation
//   - topic: The topic name to unsubscribe from
//
// Returns:
//   - error: An error if the unsubscription fails, nil otherwise
func (tm *SHIPTopicManager) UnsubscribeFromTopic(ctx context.Context, topic string) error {
	if topic == "" {
		return fmt.Errorf("topic name cannot be empty")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscription, exists := tm.subscriptions[topic]
	if !exists {
		return fmt.Errorf("not subscribed to topic: %s", topic)
	}

	// Mark subscription as inactive
	subscription.IsActive = false

	// Remove handler
	delete(tm.handlers, topic)

	return nil
}

// HandleTopicMessage processes an incoming topic message.
// Routes the message to the appropriate handler if one exists for the topic.
// Updates message statistics for the topic.
//
// Parameters:
//   - ctx: Context for the operation
//   - message: The topic message to process
//
// Returns:
//   - error: An error if message handling fails, nil otherwise
func (tm *SHIPTopicManager) HandleTopicMessage(ctx context.Context, message TopicMessage) error {
	if message.Topic == "" {
		return fmt.Errorf("message topic cannot be empty")
	}

	tm.mutex.RLock()
	subscription, subscriptionExists := tm.subscriptions[message.Topic]
	handler, handlerExists := tm.handlers[message.Topic]
	tm.mutex.RUnlock()

	// Check if we have an active subscription for this topic
	if !subscriptionExists || !subscription.IsActive {
		// Silently ignore messages for topics we're not subscribed to
		return nil
	}

	if !handlerExists {
		return fmt.Errorf("no handler found for topic: %s", message.Topic)
	}

	// Update message count
	tm.mutex.Lock()
	subscription.MessageCount++
	tm.mutex.Unlock()

	// Handle the message
	if err := handler(ctx, message); err != nil {
		return fmt.Errorf("failed to handle message for topic %s: %w", message.Topic, err)
	}

	return nil
}

// GetSubscribedTopics returns all current topic subscriptions.
// Returns a copy of subscription data to prevent external modification.
//
// Returns:
//   - []TopicSubscription: A slice of all topic subscriptions
func (tm *SHIPTopicManager) GetSubscribedTopics() []TopicSubscription {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscriptions := make([]TopicSubscription, 0, len(tm.subscriptions))
	for _, subscription := range tm.subscriptions {
		// Return a copy to prevent external modification
		subscriptions = append(subscriptions, *subscription)
	}

	return subscriptions
}

// CreateTopicSubscription creates a new topic subscription without a handler.
// This method is useful for creating subscription records before setting up handlers.
//
// Parameters:
//   - ctx: Context for the operation
//   - topic: The topic name to create a subscription for
//
// Returns:
//   - *TopicSubscription: The created subscription
//   - error: An error if creation fails, nil otherwise
func (tm *SHIPTopicManager) CreateTopicSubscription(ctx context.Context, topic string) (*TopicSubscription, error) {
	if topic == "" {
		return nil, fmt.Errorf("topic name cannot be empty")
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Check if subscription already exists
	if existing, exists := tm.subscriptions[topic]; exists {
		// Return existing subscription
		return existing, nil
	}

	// Create new subscription
	subscription := &TopicSubscription{
		Topic:        topic,
		SubscribedAt: time.Now(),
		IsActive:     false, // Not active until a handler is set
		MessageCount: 0,
	}

	tm.subscriptions[topic] = subscription
	return subscription, nil
}

// IsSubscribedToTopic checks if currently subscribed to a topic.
// Only returns true for active subscriptions.
//
// Parameters:
//   - topic: The topic name to check
//
// Returns:
//   - bool: True if actively subscribed to the topic, false otherwise
func (tm *SHIPTopicManager) IsSubscribedToTopic(topic string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscription, exists := tm.subscriptions[topic]
	return exists && subscription.IsActive
}

// GetTopicMessageCount returns the message count for a specific topic.
// Returns 0 if the topic is not subscribed to.
//
// Parameters:
//   - topic: The topic name to get the message count for
//
// Returns:
//   - int64: The number of messages received on this topic
func (tm *SHIPTopicManager) GetTopicMessageCount(topic string) int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if subscription, exists := tm.subscriptions[topic]; exists {
		return subscription.MessageCount
	}
	return 0
}

// Close cleanly shuts down the topic manager.
// Unsubscribes from all topics and cleans up resources.
//
// Parameters:
//   - ctx: Context for the shutdown operation
//
// Returns:
//   - error: An error if shutdown fails, nil otherwise
func (tm *SHIPTopicManager) Close(ctx context.Context) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Mark all subscriptions as inactive
	for _, subscription := range tm.subscriptions {
		subscription.IsActive = false
	}

	// Clear all handlers
	tm.handlers = make(map[string]TopicMessageHandler)

	return nil
}

// GetTopicManagerMetaData returns metadata information for the SHIP topic manager.
// This provides basic information about the topic manager service.
//
// Returns:
//   - types.MetaData: The topic manager metadata
func (tm *SHIPTopicManager) GetTopicManagerMetaData() types.MetaData {
	return types.MetaData{
		Name:             "SHIP Topic Manager",
		ShortDescription: "Manages overlay network topic subscriptions for SHIP protocol.",
	}
}

// GetActiveTopicCount returns the number of currently active topic subscriptions.
//
// Returns:
//   - int: The number of active subscriptions
func (tm *SHIPTopicManager) GetActiveTopicCount() int {
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

// GetTotalMessageCount returns the total number of messages processed across all topics.
//
// Returns:
//   - int64: The total message count
func (tm *SHIPTopicManager) GetTotalMessageCount() int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var total int64
	for _, subscription := range tm.subscriptions {
		total += subscription.MessageCount
	}
	return total
}
