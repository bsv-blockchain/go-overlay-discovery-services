package ship

import (
	"context"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

// SHIPTopicManager implements the TopicManager interface for SHIP (Service Host Interconnect Protocol) tokens
type SHIPTopicManager struct{}

// NewSHIPTopicManager creates a new SHIP topic manager
func NewSHIPTopicManager() *SHIPTopicManager {
	return &SHIPTopicManager{}
}

// IdentifyAdmissibleOutputs identifies which outputs should be admitted to the overlay
func (tm *SHIPTopicManager) IdentifyAdmissibleOutputs(ctx context.Context, beef []byte, previousCoins map[uint32]*transaction.TransactionOutput) (overlay.AdmittanceInstructions, error) {
	var outputsToAdmit []uint32
	var coinsToRetain []uint32

	// Parse the transaction from BEEF
	tx, err := transaction.NewTransactionFromBEEF(beef)
	if err != nil {
		return overlay.AdmittanceInstructions{}, err
	}

	// Check each output for SHIP compliance
	for i, output := range tx.Outputs {
		if tm.isValidSHIPOutput(output) {
			outputsToAdmit = append(outputsToAdmit, uint32(i))
		}
	}


	return overlay.AdmittanceInstructions{
		OutputsToAdmit: outputsToAdmit,
		CoinsToRetain:  coinsToRetain,
	}, nil
}

// IdentifyNeededInputs identifies which inputs are needed for the transaction
func (tm *SHIPTopicManager) IdentifyNeededInputs(ctx context.Context, beef []byte) ([]*transaction.Outpoint, error) {
	// For SHIP, we don't typically need additional inputs beyond what's provided
	return nil, nil
}

// GetDocumentation returns documentation specific to the SHIP topic manager
func (tm *SHIPTopicManager) GetDocumentation() string {
	return `SHIP Topic Manager

Manages SHIP tokens for service host interconnect. The SHIP Topic Manager identifies admissible outputs based on SHIP protocol requirements.

SHIP tokens facilitate the advertisement of nodes hosting specific topics within the overlay network.

Requirements for admission:
- Must be a valid PushDrop token with 5 fields
- First field must be "SHIP" identifier
- Advertised URI must be acceptable (http/https/ws/wss/tcp/udp)
- Topic must be valid and start with "tm_" (topic manager)
- Token signature must be properly linked`
}

// GetMetaData returns metadata associated with this topic manager
func (tm *SHIPTopicManager) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SHIP Topic Manager",
		Description: "Manages SHIP tokens for service host interconnect.",
	}
}

// isValidSHIPOutput checks if an output is a valid SHIP advertisement
func (tm *SHIPTopicManager) isValidSHIPOutput(output *transaction.TransactionOutput) bool {
	// Decode the PushDrop data
	result := pushdrop.Decode(output.LockingScript)
	if result == nil {
		return false
	}

	// SHIP tokens must have exactly 5 fields
	if len(result.Fields) != 5 {
		return false
	}

	// First field must be "SHIP" identifier
	shipIdentifier := string(result.Fields[0])
	if shipIdentifier != "SHIP" {
		return false
	}

	// Third field is the advertised URI - must be acceptable
	advertisedURI := string(result.Fields[2])
	if !utils.IsAdvertisableURI(advertisedURI) {
		return false
	}

	// Fourth field is the topic - must be valid and start with "tm_"
	topic := string(result.Fields[3])
	if !utils.IsValidTopicOrServiceName(topic) {
		return false
	}
	if !tm.isTopicManagerTopic(topic) {
		return false
	}

	// Verify token signature is correctly linked
	if !utils.IsTokenSignatureCorrectlyLinked(result.LockingPublicKey, result.Fields) {
		return false
	}

	return true
}

// isTopicManagerTopic checks if the topic is a topic manager topic (starts with "tm_")
func (tm *SHIPTopicManager) isTopicManagerTopic(topic string) bool {
	return len(topic) >= 3 && topic[:3] == "tm_"
}

// Verify that SHIPTopicManager implements engine.TopicManager
var _ engine.TopicManager = (*SHIPTopicManager)(nil)
