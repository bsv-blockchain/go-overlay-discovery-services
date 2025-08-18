// Package main demonstrates usage of the SHIP lookup service
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/ship"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

func main() {
	fmt.Println("=== SHIP Lookup Service Examples ===")
	fmt.Println()

	// Run the various example functions
	fmt.Println("1. Running OutputAdmittedByTopic API Demo:")
	ExampleOutputAdmittedByTopicDemo()

	fmt.Println("\n2. Running SHIP Storage Interface Example:")
	ExampleSHIPStorageInterface()

	fmt.Println("\n3. Running Lookup Service Interface Example:")
	ExampleLookupServiceInterface()

	fmt.Println("\n4. Running SHIP Usage Example (requires MongoDB):")
	ExampleUsage()

	fmt.Println("\n=== Examples Complete ===")
}

// ExampleOutputAdmittedByTopic demonstrates how to call OutputAdmittedByTopic
// with a properly constructed engine.OutputAdmittedByTopic payload.
// This shows the expected API structure for SHIP advertisement processing.
func ExampleOutputAdmittedByTopic(ctx context.Context, lookupService *ship.SHIPLookupService) error {
	fmt.Println("Demonstrating OutputAdmittedByTopic API usage:")

	// Create a sample transaction ID (32 bytes)
	sampleTxidHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	txidBytes, err := hex.DecodeString(sampleTxidHex)
	if err != nil {
		return fmt.Errorf("failed to decode sample txid: %w", err)
	}

	// Convert to [32]byte array required by transaction.Outpoint
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	// Create the outpoint (transaction output reference)
	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0, // First output
	}

	// Create a valid PushDrop locking script for SHIP advertisement
	lockingScript, err := createSampleSHIPScript()
	if err != nil {
		return fmt.Errorf("failed to create sample SHIP script: %w", err)
	}

	// Construct the OutputAdmittedByTopic payload
	// This structure would normally be created by the overlay engine
	payload := &engine.OutputAdmittedByTopic{
		Topic:         ship.SHIPTopic, // "tm_ship"
		Outpoint:      outpoint,
		Satoshis:      1000, // Sample satoshi value
		LockingScript: lockingScript,
		AtomicBEEF:    []byte("sample"), // Sample atomic BEEF data
	}

	// Call OutputAdmittedByTopic (this would normally be called by the engine)
	err = lookupService.OutputAdmittedByTopic(ctx, payload)
	if err != nil {
		return fmt.Errorf("OutputAdmittedByTopic failed: %w", err)
	}

	fmt.Printf("  ✓ Successfully processed SHIP advertisement for outpoint %s:%d\n",
		sampleTxidHex, outpoint.Index)
	fmt.Println("  ✓ SHIP record stored with:")
	fmt.Println("    - Identity Key: deadbeef01020304")
	fmt.Println("    - Domain: https://example.com")
	fmt.Println("    - Supported Topic: tm_bridge")

	return nil
}

// createSampleSHIPScript creates a valid PushDrop script containing SHIP advertisement data.
// This demonstrates the expected format for SHIP locking scripts.
func createSampleSHIPScript() (*script.Script, error) {
	// Create a valid public key (33 bytes) for the base script
	pubKeyHex := "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	pubKeyBytes, err := hex.DecodeString(pubKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Create the script
	s := &script.Script{}

	// Add public key and OP_CHECKSIG (standard P2PK pattern)
	s.AppendPushData(pubKeyBytes)
	s.AppendOpcodes(script.OpCHECKSIG)

	// Add SHIP advertisement fields using PushDrop format
	fields := [][]byte{
		[]byte("SHIP"), // Protocol identifier
		[]byte{0xde, 0xad, 0xbe, 0xef, 0x01, 0x02, 0x03, 0x04}, // Identity key
		[]byte("https://example.com"),                          // Domain where service is hosted
		[]byte("tm_bridge"),                                    // Topic/service supported
	}

	// Add fields to script
	for _, field := range fields {
		s.AppendPushData(field)
	}

	// Add DROP operations to clean up stack (PushDrop pattern)
	notYetDropped := len(fields)
	for notYetDropped > 1 {
		s.AppendOpcodes(script.Op2DROP)
		notYetDropped -= 2
	}
	if notYetDropped != 0 {
		s.AppendOpcodes(script.OpDROP)
	}

	return s, nil
}

// ExampleOutputAdmittedByTopicDemo demonstrates the API structure for OutputAdmittedByTopic
// without requiring actual storage. This shows developers the expected data structures.
func ExampleOutputAdmittedByTopicDemo() {
	fmt.Println("OutputAdmittedByTopic API Structure Demo:")

	// Sample transaction ID
	sampleTxidHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	txidBytes, _ := hex.DecodeString(sampleTxidHex)
	var txidArray [32]byte
	copy(txidArray[:], txidBytes)

	// Create sample outpoint
	outpoint := &transaction.Outpoint{
		Txid:  txidArray,
		Index: 0,
	}

	// Create sample locking script
	lockingScript, err := createSampleSHIPScript()
	if err != nil {
		log.Printf("Failed to create sample script: %v", err)
		return
	}

	// Show the structure that would be passed to OutputAdmittedByTopic
	payload := &engine.OutputAdmittedByTopic{
		Topic:         ship.SHIPTopic, // "tm_ship"
		Outpoint:      outpoint,
		Satoshis:      1000,
		LockingScript: lockingScript,
		AtomicBEEF:    []byte("sample"),
	}

	fmt.Printf("  ✓ Payload Topic: %s\n", payload.Topic)
	fmt.Printf("  ✓ Outpoint: %s:%d\n", sampleTxidHex, payload.Outpoint.Index)
	fmt.Printf("  ✓ Satoshis: %d\n", payload.Satoshis)
	fmt.Printf("  ✓ LockingScript: %s\n", lockingScript.String())
	fmt.Println("  ✓ Expected SHIP fields in script:")
	fmt.Println("    - Protocol: SHIP")
	fmt.Println("    - Identity Key: deadbeef01020304")
	fmt.Println("    - Domain: https://example.com")
	fmt.Println("    - Topic: tm_bridge")
	fmt.Println("  ✓ This payload would be created by the overlay engine automatically")
}

// ExampleUsage demonstrates how to use the SHIP lookup service
func ExampleUsage() {
	// 1. Set up MongoDB connection
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	// 2. Create SHIP storage
	db := client.Database("overlay_services")
	storage := ship.NewSHIPStorage(db)

	// Ensure indexes are created
	if err := storage.EnsureIndexes(ctx); err != nil {
		log.Fatal("Failed to ensure indexes:", err)
	}

	// 3. Create the SHIP lookup service
	lookupService := ship.NewSHIPLookupService(storage)

	// 4. Example: Handle an output admitted by topic
	// Note: This demonstrates the API structure. In production, the overlay engine
	// would call this method automatically when SHIP-related outputs are detected.
	if err := ExampleOutputAdmittedByTopic(ctx, lookupService); err != nil {
		log.Printf("OutputAdmittedByTopic example failed: %v", err)
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
			fmt.Printf("Found %d SHIP records\n", len(utxos))
		} else {
			fmt.Printf("Found SHIP records (unknown format)\n")
		}
	}

	// Modern object-based query
	domain := "https://example.com"
	modernQuery := map[string]interface{}{
		"domain": domain,
		"topics": []string{"tm_bridge", "tm_sync"},
		"limit":  10,
	}

	modernQueryJSON, _ := json.Marshal(modernQuery)
	modernQuestion := &lookup.LookupQuestion{
		Service: "ls_ship",
		Query:   modernQueryJSON,
	}

	results, err = lookupService.Lookup(ctx, modernQuestion)
	if err != nil {
		log.Printf("Modern lookup failed: %v", err)
	} else {
		if utxos, ok := results.Result.([]types.UTXOReference); ok {
			fmt.Printf("Found %d SHIP records for domain %s\n", len(utxos), domain)
			for _, result := range utxos {
				fmt.Printf("  - UTXO: %s:%d\n", result.Txid, result.OutputIndex)
			}
		} else {
			fmt.Printf("Found SHIP records for domain %s (unknown format)\n", domain)
		}
	}

	// 7. Example: Get service metadata and documentation
	metadata := lookupService.GetMetaData()
	fmt.Printf("Service: %s - %s\n", metadata.Name, metadata.Description)

	documentation := lookupService.GetDocumentation()
	fmt.Printf("Documentation length: %d characters\n", len(documentation))

	// 8. Example: Handle spent output
	// Note: This demonstrates the API structure. In production, the overlay engine
	// would call this method automatically when SHIP-related outputs are spent.
	if err := ExampleOutputSpent(ctx, lookupService); err != nil {
		log.Printf("OutputSpent example failed: %v", err)
	}
}

// ExampleOutputSpent demonstrates how to call OutputSpent
// with a properly constructed engine.OutputSpent payload.
// This shows the expected API structure for SHIP output spending.
func ExampleOutputSpent(ctx context.Context, lookupService *ship.SHIPLookupService) error {
	fmt.Println("Demonstrating OutputSpent API usage:")

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
	unlockingScript.AppendPushData([]byte{0x30, 0x44}) // Sample signature
	unlockingScript.AppendPushData([]byte{0x21, 0x02}) // Sample pubkey

	// Construct the OutputSpent payload
	// This structure would normally be created by the overlay engine
	payload := &engine.OutputSpent{
		Outpoint:        outpoint,
		Topic:           ship.SHIPTopic, // "tm_ship"
		SpendingTxid:    spendingTxidHash,
		InputIndex:      0,
		UnlockingScript: unlockingScript,
	}

	// Call OutputSpent (this would normally be called by the engine)
	err = lookupService.OutputSpent(ctx, payload)
	if err != nil {
		return fmt.Errorf("OutputSpent failed: %w", err)
	}

	fmt.Printf("  ✓ Successfully processed spent SHIP output for outpoint %s:%d\n",
		sampleTxidHex, outpoint.Index)
	fmt.Printf("  ✓ Spent by transaction: %s (input %d)\n",
		spendingTxidHex, payload.InputIndex)
	fmt.Println("  ✓ SHIP record removed from storage")
	fmt.Println("  ✓ Discovery service no longer advertises this host/topic combination")

	return nil
}

// ExampleSHIPStorageInterface demonstrates how SHIPStorage implements the interface
func ExampleSHIPStorageInterface() {
	// This example shows that SHIPStorage implements SHIPStorageInterface
	var _ ship.SHIPStorageInterface = &ship.SHIPStorage{}

	fmt.Println("SHIPStorage successfully implements SHIPStorageInterface")
}

// ExampleLookupServiceInterface demonstrates how SHIPLookupService implements the BSV overlay interface
func ExampleLookupServiceInterface() {
	// This example shows that SHIPLookupService implements types.LookupService
	var _ engine.LookupService = &ship.SHIPLookupService{}

	fmt.Println("SHIPLookupService successfully implements types.LookupService")
}
