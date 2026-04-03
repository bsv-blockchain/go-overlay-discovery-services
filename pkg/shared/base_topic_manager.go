package shared

import (
	"context"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// BaseTopicManagerConfig holds protocol-specific configuration for BaseTopicManagerOps.
type BaseTopicManagerConfig struct {
	// Admittance is the protocol-specific admittance configuration
	Admittance AdmittanceConfig
	// MetaDataName is the human-readable name for GetMetaData
	MetaDataName string
	// MetaDataDescription is the description for GetMetaData
	MetaDataDescription string
	// Documentation is the documentation string returned by GetDocumentation
	Documentation *string
}

// BaseTopicManagerOps provides shared implementations for engine.TopicManager interface methods
// that are structurally identical between SHIP and SLAP. Embed this in protocol-specific
// TopicManager types to eliminate code duplication.
type BaseTopicManagerOps struct {
	Cfg BaseTopicManagerConfig
}

// NewBaseTopicManagerOps creates a new BaseTopicManagerOps with the given configuration.
func NewBaseTopicManagerOps(cfg BaseTopicManagerConfig) BaseTopicManagerOps {
	return BaseTopicManagerOps{Cfg: cfg}
}

// IdentifyAdmissibleOutputs implements the engine.TopicManager interface.
// It delegates to the shared IdentifyAdmissibleOutputs function with protocol-specific config.
func (b *BaseTopicManagerOps) IdentifyAdmissibleOutputs(ctx context.Context, beef *transaction.Beef, txid *chainhash.Hash, previousCoins []uint32) (overlay.AdmittanceInstructions, error) {
	return IdentifyAdmissibleOutputs(ctx, beef, txid, previousCoins, b.Cfg.Admittance)
}

// IdentifyNeededInputs implements the engine.TopicManager interface.
// Discovery protocols don't require specific inputs for validation.
func (b *BaseTopicManagerOps) IdentifyNeededInputs(_ context.Context, _ *transaction.Beef, _ *chainhash.Hash) ([]*transaction.Outpoint, error) {
	return IdentifyNeededInputsNoOp()
}

// GetDocumentation implements the engine.TopicManager interface.
func (b *BaseTopicManagerOps) GetDocumentation() string {
	if b.Cfg.Documentation != nil {
		return *b.Cfg.Documentation
	}
	return ""
}

// GetMetaData implements the engine.TopicManager interface.
func (b *BaseTopicManagerOps) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        b.Cfg.MetaDataName,
		Description: b.Cfg.MetaDataDescription,
	}
}
