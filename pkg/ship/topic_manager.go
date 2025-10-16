// Package ship implements the SHIP (Service Host Interconnect Protocol) topic manager functionality.
// Overlay network topic management and message routing for SHIP protocol.
package ship

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
	errTopicNameEmpty         = errors.New("topic name cannot be empty")
	errMessageHandlerNil      = errors.New("message handler cannot be nil")
	errNotSubscribedToTopic   = errors.New("not subscribed to topic")
	errMessageTopicEmpty      = errors.New("message topic cannot be empty")
	errNoHandlerFoundForTopic = errors.New("no handler found for topic")
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

// TopicManager implements topic management functionality for SHIP protocol.
// It provides capabilities for subscribing to overlay network topics, handling messages,
// and managing topic lifecycle within the SHIP ecosystem.
type TopicManager struct {
	// subscriptions holds all active topic subscriptions
	subscriptions map[string]*TopicSubscription
	// handlers holds message handlers for each subscribed topic
	handlers map[string]TopicMessageHandler
	// mutex protects concurrent access to subscriptions and handlers
	mutex sync.RWMutex
	// storage provides access to SHIP storage operations
	storage StorageInterface
	// lookupService provides access to SHIP lookup operations (optional integration)
	lookupService *LookupService
}

// NewTopicManager creates a new SHIP topic manager instance.
// This constructor initializes the topic manager with the required dependencies
// for managing overlay network topic subscriptions and message routing.
func NewTopicManager(storage StorageInterface, lookupService *LookupService) *TopicManager {
	return &TopicManager{
		subscriptions: make(map[string]*TopicSubscription),
		handlers:      make(map[string]TopicMessageHandler),
		storage:       storage,
		lookupService: lookupService,
	}
}

// SubscribeToTopic subscribes to a specific topic with a message handler.
// Creates a new subscription if one doesn't exist, or updates an existing one.
// The provided handler will be called for all messages received on this topic.
func (tm *TopicManager) SubscribeToTopic(_ context.Context, topic string, handler TopicMessageHandler) error {
	if topic == "" {
		return errTopicNameEmpty
	}

	if handler == nil {
		return errMessageHandlerNil
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
func (tm *TopicManager) UnsubscribeFromTopic(_ context.Context, topic string) error {
	if topic == "" {
		return errTopicNameEmpty
	}

	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	subscription, exists := tm.subscriptions[topic]
	if !exists {
		return fmt.Errorf("%w: %s", errNotSubscribedToTopic, topic)
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
func (tm *TopicManager) HandleTopicMessage(ctx context.Context, message TopicMessage) error {
	if message.Topic == "" {
		return errMessageTopicEmpty
	}

	tm.mutex.RLock()
	subscription, subscriptionExists := tm.subscriptions[message.Topic]
	handler, handlerExists := tm.handlers[message.Topic]
	isActive := subscriptionExists && subscription.IsActive
	tm.mutex.RUnlock()

	// Check if we have an active subscription for this topic
	if !subscriptionExists || !isActive {
		// Silently ignore messages for topics we're not subscribed to
		return nil
	}

	if !handlerExists {
		return fmt.Errorf("%w: %s", errNoHandlerFoundForTopic, message.Topic)
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
func (tm *TopicManager) GetSubscribedTopics() []TopicSubscription {
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
func (tm *TopicManager) CreateTopicSubscription(_ context.Context, topic string) (*TopicSubscription, error) {
	if topic == "" {
		return nil, errTopicNameEmpty
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
func (tm *TopicManager) IsSubscribedToTopic(topic string) bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	subscription, exists := tm.subscriptions[topic]
	return exists && subscription.IsActive
}

// GetTopicMessageCount returns the message count for a specific topic.
// Returns 0 if the topic is not subscribed to.
func (tm *TopicManager) GetTopicMessageCount(topic string) int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	if subscription, exists := tm.subscriptions[topic]; exists {
		return subscription.MessageCount
	}
	return 0
}

// Close cleanly shuts down the topic manager.
// Unsubscribes from all topics and cleans up resources.
func (tm *TopicManager) Close(_ context.Context) error {
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
func (tm *TopicManager) GetTopicManagerMetaData() overlay.MetaData {
	return overlay.MetaData{
		Name:        "SHIP Topic Manager",
		Description: "Manages overlay network topic subscriptions for SHIP protocol.",
	}
}

// GetActiveTopicCount returns the number of currently active topic subscriptions.
func (tm *TopicManager) GetActiveTopicCount() int {
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
func (tm *TopicManager) GetTotalMessageCount() int64 {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	var total int64
	for _, subscription := range tm.subscriptions {
		total += subscription.MessageCount
	}
	return total
}

// IdentifyAdmissibleOutputs implements the engine.TopicManager interface
// For SHIP, this identifies outputs that should be admitted to the overlay
func (tm *TopicManager) IdentifyAdmissibleOutputs(ctx context.Context, beef []byte, previousCoins map[uint32]*transaction.TransactionOutput) (overlay.AdmittanceInstructions, error) {
	outputsToAdmit := []uint32{}

	// Parse transaction from BEEF format
	parsedTransaction, err := transaction.NewTransactionFromBEEF(beef)
	if err != nil {
		// Only log error if no outputs were admitted and no previous coins consumed
		if len(outputsToAdmit) == 0 && len(previousCoins) == 0 {
			log.Printf("⛴️ Error identifying admissible outputs: %v", err)
		}
		return overlay.AdmittanceInstructions{
			OutputsToAdmit: outputsToAdmit,
			CoinsToRetain:  []uint32{},
		}, nil
	}

	// Check each output for SHIP token validity
	for i, output := range parsedTransaction.Outputs {
		// Decode PushDrop token
		result := pushdrop.Decode(output.LockingScript)
		if result == nil {
			continue // It's common for other outputs to be invalid; no need to log an error here
		}

		// SHIP tokens must have exactly 5 fields
		if len(result.Fields) != 5 {
			continue
		}

		// Check SHIP identifier (first field)
		shipIdentifier := utils.UTFBytesToString(result.Fields[0])
		if shipIdentifier != "SHIP" {
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

		// SHIP only accepts "tm_" (topic manager) advertisements
		if !strings.HasPrefix(topic, "tm_") {
			continue
		}

		// Check token signature linkage
		lockingPublicKey := result.LockingPublicKey.ToDERHex()
		tokenFields := make(utils.TokenFields, len(result.Fields))
		copy(tokenFields, result.Fields)

		if valid, err := utils.IsTokenSignatureCorrectlyLinked(ctx, lockingPublicKey, tokenFields); err == nil && valid {
			if i >= 0 && i <= 0xFFFFFFFF {
				outputsToAdmit = append(outputsToAdmit, uint32(i))
			}
		} else if err == nil && !valid {
			slog.Info("Invalid token signature linkage", "outputIndex", i, "txid", parsedTransaction.TxID())
		}
	}

	// Friendly logging with ship emojis
	if len(outputsToAdmit) > 0 {
		if len(outputsToAdmit) == 1 {
			log.Printf("🛳️ Ahoy! Admitted %d SHIP output!", len(outputsToAdmit))
		} else {
			log.Printf("🛳️ Ahoy! Admitted %d SHIP outputs!", len(outputsToAdmit))
		}
	}

	if len(previousCoins) > 0 {
		if len(previousCoins) == 1 {
			log.Printf("🚢 Consumed %d previous SHIP coin!", len(previousCoins))
		} else {
			log.Printf("🚢 Consumed %d previous SHIP coins!", len(previousCoins))
		}
	}

	if len(outputsToAdmit) == 0 && len(previousCoins) == 0 {
		log.Printf("⚓ No SHIP outputs admitted and no previous SHIP coins consumed.")
	}

	return overlay.AdmittanceInstructions{
		OutputsToAdmit: outputsToAdmit,
		CoinsToRetain:  []uint32{},
	}, nil
}

// IdentifyNeededInputs implements the engine.TopicManager interface
// For SHIP, this identifies inputs needed for validation
func (tm *TopicManager) IdentifyNeededInputs(_ context.Context, _ []byte) ([]*transaction.Outpoint, error) {
	// SHIP doesn't require specific inputs for validation
	return []*transaction.Outpoint{}, nil
}

// GetDocumentation implements the engine.TopicManager interface
// Returns documentation for the SHIP topic manager
func (tm *TopicManager) GetDocumentation() string {
	return TopicManagerDocumentation
}

// GetMetaData implements the engine.TopicManager interface
// Returns metadata about the SHIP topic manager
func (tm *TopicManager) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SHIP Topic Manager",
		Description: "Manages SHIP protocol topics for service host interconnection and discovery",
	}
}
