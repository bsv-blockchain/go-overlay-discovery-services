package slap

import (
	"context"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

// SLAPTopicManager implements the TopicManager interface for SLAP (Service Lookup Availability Protocol) tokens
type SLAPTopicManager struct{}

// NewSLAPTopicManager creates a new SLAP topic manager
func NewSLAPTopicManager() *SLAPTopicManager {
	return &SLAPTopicManager{}
}

// IdentifyAdmissibleOutputs identifies which outputs should be admitted to the overlay
func (tm *SLAPTopicManager) IdentifyAdmissibleOutputs(ctx context.Context, beef []byte, previousCoins map[uint32]*transaction.TransactionOutput) (overlay.AdmittanceInstructions, error) {
	var outputsToAdmit []uint32
	var coinsToRetain []uint32

	// Parse the transaction from BEEF
	tx, err := transaction.NewTransactionFromBEEF(beef)
	if err != nil {
		return overlay.AdmittanceInstructions{}, err
	}

	// Check each output for SLAP compliance
	for i, output := range tx.Outputs {
		if tm.isValidSLAPOutput(output) {
			outputsToAdmit = append(outputsToAdmit, uint32(i))
		}
	}


	return overlay.AdmittanceInstructions{
		OutputsToAdmit: outputsToAdmit,
		CoinsToRetain:  coinsToRetain,
	}, nil
}

// IdentifyNeededInputs identifies which inputs are needed for the transaction
func (tm *SLAPTopicManager) IdentifyNeededInputs(ctx context.Context, beef []byte) ([]*transaction.Outpoint, error) {
	// For SLAP, we don't typically need additional inputs beyond what's provided
	return nil, nil
}

// GetDocumentation returns documentation specific to the SLAP topic manager
func (tm *SLAPTopicManager) GetDocumentation() string {
	return `SLAP Topic Manager

Manages SLAP tokens for service lookup availability. The SLAP Topic Manager identifies admissible outputs based on SLAP protocol requirements.

SLAP tokens facilitate the advertisement of service availability within the overlay network.

Requirements for admission:
- Must be a valid PushDrop token with 5 fields
- First field must be "SLAP" identifier
- Advertised URI must be acceptable (http/https/ws/wss/tcp/udp)
- Service name must be valid and start with "ls_" (lookup service)
- Token signature must be properly linked`
}

// GetMetaData returns metadata associated with this topic manager
func (tm *SLAPTopicManager) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SLAP Topic Manager",
		Description: "Manages SLAP tokens for service lookup availability.",
	}
}

// isValidSLAPOutput checks if an output is a valid SLAP advertisement
func (tm *SLAPTopicManager) isValidSLAPOutput(output *transaction.TransactionOutput) bool {
	// Decode the PushDrop data
	result := pushdrop.Decode(output.LockingScript)
	if result == nil {
		return false
	}

	// SLAP tokens must have exactly 5 fields
	if len(result.Fields) != 5 {
		return false
	}

	// First field must be "SLAP" identifier
	slapIdentifier := string(result.Fields[0])
	if slapIdentifier != "SLAP" {
		return false
	}

	// Third field is the advertised URI - must be acceptable
	advertisedURI := string(result.Fields[2])
	if !utils.IsAdvertisableURI(advertisedURI) {
		return false
	}

	// Fourth field is the service name - must be valid and start with "ls_"
	serviceName := string(result.Fields[3])
	if !utils.IsValidTopicOrServiceName(serviceName) {
		return false
	}
	if !tm.isLookupService(serviceName) {
		return false
	}

	// Verify token signature is correctly linked
	if !utils.IsTokenSignatureCorrectlyLinked(result.LockingPublicKey, result.Fields) {
		return false
	}

	return true
}

// isLookupService checks if the service name is a lookup service (starts with "ls_")
func (tm *SLAPTopicManager) isLookupService(serviceName string) bool {
	return len(serviceName) >= 3 && serviceName[:3] == "ls_"
}

// Verify that SLAPTopicManager implements engine.TopicManager
var _ engine.TopicManager = (*SLAPTopicManager)(nil)
