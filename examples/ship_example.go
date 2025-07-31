// Package main demonstrates usage of the SHIP lookup service
package main

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/ship"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

func main() {
	fmt.Println("=== SHIP Lookup Service Examples ===")
	fmt.Println()

	// Run the various example functions
	fmt.Println("1. Running SHIP Usage Example:")
	ExampleUsage()

	fmt.Println("\n2. Running SHIP Storage Interface Example:")
	ExampleSHIPStorageInterface()

	fmt.Println("\n3. Running Lookup Service Interface Example:")
	ExampleLookupServiceInterface()

	fmt.Println("\n=== Examples Complete ===")
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

	// 3. Create mock PushDrop decoder and utils for this example
	// In production, you would use the real BSV SDK implementations
	pushDropDecoder := ship.NewMockPushDropDecoder()
	utils := ship.NewMockUtils()

	// 4. Create the SHIP lookup service
	lookupService := ship.NewSHIPLookupService(storage, pushDropDecoder, utils)

	// 5. Example: Handle an output admitted by topic
	admissionPayload := types.OutputAdmittedByTopic{
		Mode:          types.AdmissionModeLockingScript,
		Topic:         "tm_ship",
		LockingScript: "deadbeef", // This would be the actual locking script in hex
		Txid:          "abc123def456",
		OutputIndex:   0,
	}

	if err := lookupService.OutputAdmittedByTopic(admissionPayload); err != nil {
		log.Printf("Failed to handle admitted output: %v", err)
	} else {
		fmt.Println("Successfully processed SHIP advertisement")
	}

	// 6. Example: Perform lookup queries

	// Legacy findAll query
	legacyQuestion := types.LookupQuestion{
		Service: "ls_ship",
		Query:   "findAll",
	}

	results, err := lookupService.Lookup(legacyQuestion)
	if err != nil {
		log.Printf("Legacy lookup failed: %v", err)
	} else {
		fmt.Printf("Found %d SHIP records\n", len(results))
	}

	// Modern object-based query
	domain := "https://example.com"
	modernQuery := map[string]interface{}{
		"domain": domain,
		"topics": []string{"tm_bridge", "tm_sync"},
		"limit":  10,
	}

	modernQuestion := types.LookupQuestion{
		Service: "ls_ship",
		Query:   modernQuery,
	}

	results, err = lookupService.Lookup(modernQuestion)
	if err != nil {
		log.Printf("Modern lookup failed: %v", err)
	} else {
		fmt.Printf("Found %d SHIP records for domain %s\n", len(results), domain)
		for _, result := range results {
			fmt.Printf("  - UTXO: %s:%d\n", result.Txid, result.OutputIndex)
		}
	}

	// 7. Example: Get service metadata and documentation
	metadata, err := lookupService.GetMetaData()
	if err != nil {
		log.Printf("Failed to get metadata: %v", err)
	} else {
		fmt.Printf("Service: %s - %s\n", metadata.Name, metadata.ShortDescription)
	}

	documentation, err := lookupService.GetDocumentation()
	if err != nil {
		log.Printf("Failed to get documentation: %v", err)
	} else {
		fmt.Printf("Documentation length: %d characters\n", len(documentation))
	}

	// 8. Example: Handle spent output
	spentPayload := types.OutputSpent{
		Mode:        types.SpendNotificationModeNone,
		Topic:       "tm_ship",
		Txid:        "abc123def456",
		OutputIndex: 0,
	}

	if err := lookupService.OutputSpent(spentPayload); err != nil {
		log.Printf("Failed to handle spent output: %v", err)
	} else {
		fmt.Println("Successfully processed spent SHIP output")
	}
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
	var _ types.LookupService = &ship.SHIPLookupService{}

	fmt.Println("SHIPLookupService successfully implements types.LookupService")
}
