package shared

import (
	"context"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// DiscoveryNoOps provides no-op implementations of engine.LookupService methods
// that are not relevant for discovery services. Embed this struct in SHIP and SLAP
// LookupService types to eliminate duplicate no-op method implementations.
type DiscoveryNoOps struct{}

// OutputNoLongerRetainedInHistory is a no-op for discovery services.
// Discovery services don't have the concept of historical retention, so this is ignored.
func (DiscoveryNoOps) OutputNoLongerRetainedInHistory(_ context.Context, _ *transaction.Outpoint, _ string) error {
	return nil
}

// OutputBlockHeightUpdated is a no-op for discovery services.
// Discovery services don't track block heights, so this is ignored.
func (DiscoveryNoOps) OutputBlockHeightUpdated(_ context.Context, _ *chainhash.Hash, _ uint32, _ uint64) error {
	return nil
}

// IdentifyNeededInputsNoOp returns an empty outpoint slice.
// Both SHIP and SLAP don't require specific inputs for validation.
func IdentifyNeededInputsNoOp() ([]*transaction.Outpoint, error) {
	return []*transaction.Outpoint{}, nil
}
