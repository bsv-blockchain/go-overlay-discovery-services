package shared

import (
	"context"
	"log"
	"log/slog"
	"strings"

	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

// AdmittanceConfig configures protocol-specific behavior for IdentifyAdmissibleOutputs.
type AdmittanceConfig struct {
	// Identifier is the protocol identifier (e.g. "SHIP" or "SLAP").
	Identifier string
	// TopicPrefix is the required prefix for the topic/service field (e.g. "tm_" or "ls_").
	TopicPrefix string
	// EmojiAdmit is the emoji used for admit log messages.
	EmojiAdmit string
	// EmojiConsume is the emoji used for consume log messages.
	EmojiConsume string
	// EmojiNone is the emoji used when nothing was admitted/consumed.
	EmojiNone string
}

// IdentifyAdmissibleOutputs is the shared implementation for SHIP and SLAP topic managers.
// It parses the BEEF transaction, validates PushDrop tokens, and returns admittance instructions.
func IdentifyAdmissibleOutputs(ctx context.Context, beef []byte, previousCoins map[uint32]*transaction.TransactionOutput, cfg AdmittanceConfig) (overlay.AdmittanceInstructions, error) {
	outputsToAdmit := []uint32{}

	// Parse transaction from BEEF format
	parsedTransaction, err := transaction.NewTransactionFromBEEF(beef)
	if err != nil {
		if len(previousCoins) == 0 {
			log.Printf("%s Error identifying admissible outputs: %v", cfg.EmojiNone, err)
		}
		return overlay.AdmittanceInstructions{
			OutputsToAdmit: outputsToAdmit,
			CoinsToRetain:  []uint32{},
		}, nil
	}

	// Check each output for token validity
	for i, output := range parsedTransaction.Outputs {
		if idx, ok := validateOutput(ctx, i, output, parsedTransaction, cfg); ok {
			outputsToAdmit = append(outputsToAdmit, idx)
		}
	}

	logAdmittanceResults(outputsToAdmit, previousCoins, cfg)

	return overlay.AdmittanceInstructions{
		OutputsToAdmit: outputsToAdmit,
		CoinsToRetain:  []uint32{},
	}, nil
}

// validateOutput checks whether a single transaction output contains a valid PushDrop token
// matching the protocol configuration. Returns the output index and true if valid.
func validateOutput(ctx context.Context, i int, output *transaction.TransactionOutput, parsedTransaction *transaction.Transaction, cfg AdmittanceConfig) (uint32, bool) {
	result := pushdrop.Decode(output.LockingScript)
	if result == nil || len(result.Fields) != 5 {
		return 0, false
	}

	if utils.UTFBytesToString(result.Fields[0]) != cfg.Identifier {
		return 0, false
	}

	if !utils.IsAdvertisableURI(utils.UTFBytesToString(result.Fields[2])) {
		return 0, false
	}

	topic := utils.UTFBytesToString(result.Fields[3])
	if !utils.IsValidTopicOrServiceName(topic) || !strings.HasPrefix(topic, cfg.TopicPrefix) {
		return 0, false
	}

	lockingPublicKey := result.LockingPublicKey.ToDERHex()
	tokenFields := make(utils.TokenFields, len(result.Fields))
	copy(tokenFields, result.Fields)

	valid, sigErr := utils.IsTokenSignatureCorrectlyLinked(ctx, lockingPublicKey, tokenFields)
	if sigErr != nil || !valid {
		if sigErr == nil {
			slog.Info("Invalid token signature linkage", "outputIndex", i, "txid", parsedTransaction.TxID())
		}
		return 0, false
	}

	if i < 0 || i > 0xFFFFFFFF {
		return 0, false
	}
	return uint32(i), true
}

// logAdmittanceResults logs the outcome of the admittance check.
func logAdmittanceResults(outputsToAdmit []uint32, previousCoins map[uint32]*transaction.TransactionOutput, cfg AdmittanceConfig) {
	if len(outputsToAdmit) > 0 {
		suffix := pluralSuffix(len(outputsToAdmit))
		log.Printf("%s Admitted %d %s output%s!", cfg.EmojiAdmit, len(outputsToAdmit), cfg.Identifier, suffix)
	}

	if len(previousCoins) > 0 {
		suffix := pluralSuffix(len(previousCoins))
		log.Printf("%s Consumed %d previous %s coin%s!", cfg.EmojiConsume, len(previousCoins), cfg.Identifier, suffix)
	}

	if len(outputsToAdmit) == 0 && len(previousCoins) == 0 {
		log.Printf("%s No %s outputs admitted and no previous %s coins consumed.", cfg.EmojiNone, cfg.Identifier, cfg.Identifier)
	}
}

// pluralSuffix returns "s" when count != 1, empty string otherwise.
func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
