package shared

import (
	"context"
	"encoding/hex"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// DeleteRecordFunc is the function signature for deleting a record by txid and output index.
type DeleteRecordFunc func(ctx context.Context, txid string, outputIndex int) error

// HandleOutputSpent processes an OutputSpent event by deleting the corresponding record
// if the payload matches the expected topic. Returns nil for non-matching topics.
func HandleOutputSpent(ctx context.Context, payload *engine.OutputSpent, expectedTopic string, deleteFn DeleteRecordFunc) error {
	if payload.Topic != expectedTopic {
		return nil
	}
	txid := hex.EncodeToString(payload.Outpoint.Txid[:])
	return deleteFn(ctx, txid, int(payload.Outpoint.Index))
}

// HandleOutputEvicted processes an OutputEvicted event by deleting the corresponding record.
func HandleOutputEvicted(ctx context.Context, outpoint *transaction.Outpoint, deleteFn DeleteRecordFunc) error {
	txid := hex.EncodeToString(outpoint.Txid[:])
	return deleteFn(ctx, txid, int(outpoint.Index))
}
