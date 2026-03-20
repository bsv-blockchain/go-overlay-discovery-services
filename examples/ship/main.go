// Package main demonstrates usage of the SHIP lookup service
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/ship"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

//nolint:gochecknoglobals // logger is used across multiple example functions
var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func main() {
	logger.Info("SHIP Lookup Service Examples")

	// Run the various example functions
	logger.Info("Running OutputAdmittedByTopic API Demo", slog.String("step", "1"))
	ExampleOutputAdmittedByTopicDemo()

	logger.Info("Running SHIP Storage Interface Example", slog.String("step", "2"))
	ExampleSHIPStorageInterface()

	logger.Info("Running Lookup Service Interface Example", slog.String("step", "3"))
	ExampleLookupServiceInterface()

	logger.Info("Running SHIP Usage Example (requires MongoDB)", slog.String("step", "4"))
	ExampleUsage()

	logger.Info("Examples Complete")
}

// ExampleOutputAdmittedByTopic demonstrates how to call OutputAdmittedByTopic
// with a properly constructed engine.OutputAdmittedByTopic payload.
// This shows the expected API structure for SHIP advertisement processing.
func ExampleOutputAdmittedByTopic(ctx context.Context, lookupService *ship.LookupService) error {
	logger.Info("Demonstrating OutputAdmittedByTopic API usage")

	// Construct the OutputAdmittedByTopic payload
	// In production, this structure is created by the overlay engine with real BEEF data.
	payload := &engine.OutputAdmittedByTopic{
		Topic:       ship.Topic, // "tm_ship"
		OutputIndex: 0,
		AtomicBEEF:  []byte("sample"), // Sample atomic BEEF data (invalid; for demo only)
	}

	// Call OutputAdmittedByTopic (this would normally be called by the engine)
	err := lookupService.OutputAdmittedByTopic(ctx, payload)
	if err != nil {
		return fmt.Errorf("OutputAdmittedByTopic failed: %w", err)
	}

	logger.Info("Successfully processed SHIP advertisement",
		slog.Int("outputIndex", int(payload.OutputIndex)),
		slog.String("identityKey", "deadbeef01020304"),
		slog.String("domain", "https://example.com"),
		slog.String("topic", "tm_bridge"))

	return nil
}

// ExampleOutputAdmittedByTopicDemo demonstrates the API structure for OutputAdmittedByTopic
// without requiring actual storage. This shows developers the expected data structures.
func ExampleOutputAdmittedByTopicDemo() {
	logger.Info("OutputAdmittedByTopic API Structure Demo:")

	sampleTxidHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	// Show the structure that would be passed to OutputAdmittedByTopic
	// In production, AtomicBEEF is provided by the overlay engine.
	payload := &engine.OutputAdmittedByTopic{
		Topic:       ship.Topic, // "tm_ship"
		OutputIndex: 0,
		AtomicBEEF:  []byte("sample"),
	}

	logger.Info("OutputAdmittedByTopic API Structure Demo",
		slog.String("topic", payload.Topic),
		slog.String("outpoint", sampleTxidHex),
		slog.Int("outputIndex", int(payload.OutputIndex)))
	logger.Info("Expected SHIP fields in script",
		slog.String("protocol", "SHIP"),
		slog.String("identityKey", "deadbeef01020304"),
		slog.String("domain", "https://example.com"),
		slog.String("topic", "tm_bridge"))
	logger.Info("This payload would be created by the overlay engine automatically")
}

// ExampleUsage demonstrates how to use the SHIP lookup service
func ExampleUsage() {
	// 1. Set up MongoDB connection
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if errDisconnect := client.Disconnect(ctx); errDisconnect != nil {
			log.Printf("error disconnecting from MongoDB: %v", errDisconnect)
		}
	}()

	// 2. Create SHIP storage
	db := client.Database("overlay_services")
	storage := ship.NewStorage(db)

	// Ensure indexes are created
	if errIndex := storage.EnsureIndexes(ctx); errIndex != nil {
		log.Fatal("Failed to ensure indexes:", errIndex)
	}

	// 3. Create the SHIP lookup service
	lookupService := ship.NewLookupService(storage)

	// 4. Example: Handle an output admitted by topic
	// This demonstrates the API structure. In production, the overlay engine
	// would call this method automatically when SHIP-related outputs are detected.
	if errOutput := ExampleOutputAdmittedByTopic(ctx, lookupService); errOutput != nil {
		log.Printf("OutputAdmittedByTopic example failed: %v", errOutput)
	}

	// 6. Example: Perform lookup queries

	// Legacy findAll query
	legacyQuestion := &lookup.LookupQuestion{
		Service: "ls_ship",
		Query:   json.RawMessage(`"findAll"`),
	}

	results, err := lookupService.Lookup(ctx, legacyQuestion)
	if err != nil {
		log.Printf("Legacy lookup failed: %v", err)
	} else {
		if utxos, ok := results.Result.([]types.UTXOReference); ok {
			logger.Info("Found SHIP records", "count", len(utxos))
		} else {
			logger.Info("Found SHIP records (unknown format)")
		}
	}

	// Modern object-based query
	domain := "https://example.com"
	modernQuery := map[string]interface{}{
		"domain": domain,
		"topics": []string{"tm_bridge", "tm_sync"},
		"limit":  10,
	}

	modernQueryJSON, err := json.Marshal(modernQuery)
	if err != nil {
		log.Printf("Failed to marshal modernQuery: %v", err)
		return
	}
	modernQuestion := &lookup.LookupQuestion{
		Service: "ls_ship",
		Query:   modernQueryJSON,
	}

	results, err = lookupService.Lookup(ctx, modernQuestion)
	if err != nil {
		log.Printf("Modern lookup failed: %v", err)
	} else {
		if utxos, ok := results.Result.([]types.UTXOReference); ok {
			logger.Info("Found SHIP records for domain", "count", len(utxos), "domain", domain)
			for _, result := range utxos {
				logger.Info("UTXO", "txid", result.Txid, "index", result.OutputIndex)
			}
		} else {
			logger.Info("Found SHIP records for domain (unknown format)", "domain", domain)
		}
	}

	// 7. Example: Get service metadata and documentation
	metadata := lookupService.GetMetaData()
	logger.Info("Service metadata", "name", metadata.Name, "description", metadata.Description)

	documentation := lookupService.GetDocumentation()
	logger.Info("Documentation", "length", len(documentation))

	// 8. Example: Handle spent output
	// This demonstrates the API structure. In production, the overlay engine
	// would call this method automatically when SHIP-related outputs are spent.
	if errSpent := ExampleOutputSpent(ctx, lookupService); errSpent != nil {
		log.Printf("OutputSpent example failed: %v", errSpent)
	}
}

// ExampleOutputSpent demonstrates how to call OutputSpent
// with a properly constructed engine.OutputSpent payload.
// This shows the expected API structure for SHIP output spending.
func ExampleOutputSpent(ctx context.Context, lookupService *ship.LookupService) error {
	logger.Info("Demonstrating OutputSpent API usage:")

	// Create a sample transaction ID (32 bytes) - same as the admitted output
	sampleTxidHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	txidBytes, err := hex.DecodeString(sampleTxidHex)
	if err != nil {
		return fmt.Errorf("failed to decode sample txid: %w", err)
	}

	// Convert to [32]byte array required by transaction.Outpoint
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	// Create the outpoint (transaction output reference) for the spent output
	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0, // Same output that was previously admitted
	}

	// Create a sample spending transaction ID
	spendingTxidHex := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	spendingTxidBytes, err := hex.DecodeString(spendingTxidHex)
	if err != nil {
		return fmt.Errorf("failed to decode spending txid: %w", err)
	}

	var spendingTxidArray [32]byte
	copy(spendingTxidArray[:], spendingTxidBytes)

	// Convert to chainhash.Hash for spending transaction reference
	spendingTxidHash := (*chainhash.Hash)(&spendingTxidArray)

	// Create a sample unlocking script
	unlockingScript := &script.Script{}
	if errAppend := unlockingScript.AppendPushData([]byte{0x30, 0x44}); errAppend != nil {
		log.Fatal("failed to append signature to unlocking script:", errAppend)
	}
	if errAppend := unlockingScript.AppendPushData([]byte{0x21, 0x02}); errAppend != nil {
		log.Fatal("failed to append pubkey to unlocking script:", errAppend)
	}

	// Construct the OutputSpent payload
	// This structure would normally be created by the overlay engine
	payload := &engine.OutputSpent{
		Outpoint:        outpoint,
		Topic:           ship.Topic, // "tm_ship"
		SpendingTxid:    spendingTxidHash,
		InputIndex:      0,
		UnlockingScript: unlockingScript,
	}

	// Call OutputSpent (this would normally be called by the engine)
	err = lookupService.OutputSpent(ctx, payload)
	if err != nil {
		return fmt.Errorf("OutputSpent failed: %w", err)
	}

	logger.Info("Successfully processed spent SHIP output", "outpoint", sampleTxidHex, "index", outpoint.Index)
	logger.Info("Spent by transaction", "txid", spendingTxidHex, "input", payload.InputIndex)
	logger.Info("SHIP record removed from storage")
	logger.Info("Discovery service no longer advertises this host/topic combination")

	return nil
}

// ExampleSHIPStorageInterface demonstrates how SHIPStorage implements the interface
func ExampleSHIPStorageInterface() {
	// This example shows that SHIPStorage implements SHIPStorageInterface
	var _ ship.StorageInterface = &ship.Storage{}

	logger.Info("SHIPStorage successfully implements SHIPStorageInterface")
}

// ExampleLookupServiceInterface demonstrates how SHIPLookupService implements the BSV overlay interface
func ExampleLookupServiceInterface() {
	// This example shows that LookupService implements types.LookupService
	var _ engine.LookupService = &ship.LookupService{}

	logger.Info("LookupService successfully implements types.LookupService")
}
