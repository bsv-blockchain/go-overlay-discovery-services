package shared

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
)

// Common error variables for PushDrop processing.
var (
	ErrPushDropDecodeFailed  = errors.New("failed to decode PushDrop locking script")
	ErrInvalidPushDropFields = errors.New("invalid PushDrop result: expected at least 4 fields")
	ErrBEEFParseFailed       = errors.New("failed to parse atomic BEEF")
	ErrOutputIndexOutOfRange = errors.New("output index out of range")
)

// PushDropFields holds the parsed fields from a PushDrop locking script.
type PushDropFields struct {
	IdentityKey string
	Domain      string
	FourthField string // topic (SHIP) or service (SLAP)
	Txid        string
	OutputIndex int
}

// ParsePushDropOutput decodes a PushDrop locking script from an OutputAdmittedByTopic payload,
// validates the protocol identifier, and returns the parsed fields.
// Returns nil (no error) if the topic or identifier doesn't match, indicating the output
// should be silently ignored.
func ParsePushDropOutput(payload *engine.OutputAdmittedByTopic, expectedTopic, expectedIdentifier string) (*PushDropFields, error) {
	// Only process the expected topic
	if payload.Topic != expectedTopic {
		return nil, nil //nolint:nilnil // nil,nil means silently skip
	}

	// Parse the atomic BEEF to extract the transaction
	tx, err := transaction.NewTransactionFromBEEF(payload.AtomicBEEF)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrBEEFParseFailed, err)
	}
	if tx == nil {
		return nil, ErrBEEFParseFailed
	}
	if int(payload.OutputIndex) >= len(tx.Outputs) {
		return nil, fmt.Errorf("%w: index %d, outputs %d", ErrOutputIndexOutOfRange, payload.OutputIndex, len(tx.Outputs))
	}

	// Decode the PushDrop locking script
	result := pushdrop.Decode(tx.Outputs[payload.OutputIndex].LockingScript)
	if result == nil {
		return nil, ErrPushDropDecodeFailed
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return nil, fmt.Errorf("%w: got %d", ErrInvalidPushDropFields, len(result.Fields))
	}

	// Extract and validate protocol identifier
	identifier := string(result.Fields[0])
	if identifier != expectedIdentifier {
		return nil, nil //nolint:nilnil // nil,nil means silently skip
	}

	return &PushDropFields{
		IdentityKey: hex.EncodeToString(result.Fields[1]),
		Domain:      string(result.Fields[2]),
		FourthField: string(result.Fields[3]),
		Txid:        hex.EncodeToString(tx.TxID().CloneBytes()),
		OutputIndex: int(payload.OutputIndex),
	}, nil
}
