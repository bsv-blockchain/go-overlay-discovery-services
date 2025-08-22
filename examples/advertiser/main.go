// Package main demonstrates how to use the WalletAdvertiser for creating and managing
// SHIP and SLAP overlay advertisements.
package main

import (
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/advertiser"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
)

func main() {
	fmt.Println("BSV Overlay Discovery Services - WalletAdvertiser Example")
	fmt.Println("========================================================")

	// Example configuration
	chain := "main"
	privateKey := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	storageURL := "https://storage.example.com"
	advertisableURI := "https://service.example.com/"

	// Optional lookup resolver configuration
	lookupConfig := &types.LookupResolverConfig{
		HTTPSEndpoint: stringPtr("https://resolver.example.com"),
		MaxRetries:    intPtr(3),
		TimeoutMS:     intPtr(5000),
	}

	// Create a new WalletAdvertiser
	fmt.Println("\n1. Creating WalletAdvertiser...")
	advertiser, err := advertiser.NewWalletAdvertiser(
		chain,
		privateKey,
		storageURL,
		advertisableURI,
		lookupConfig,
	)
	if err != nil {
		log.Fatalf("Failed to create WalletAdvertiser: %v", err)
	}
	fmt.Printf("✓ WalletAdvertiser created successfully\n")
	fmt.Printf("  Chain: %s\n", advertiser.GetChain())
	fmt.Printf("  Storage URL: %s\n", advertiser.GetStorageURL())
	fmt.Printf("  Advertisable URI: %s\n", advertiser.GetAdvertisableURI())

	// Set up mock dependencies (in a real scenario, these would be actual implementations)
	fmt.Println("\n2. Setting up dependencies...")
	advertiser.SetSkipStorageValidation(true) // Skip storage validation for example
	advertiser.SetTestMode(true)              // Enable test mode for example
	fmt.Println("✓ Dependencies configured")

	// Initialize the advertiser
	fmt.Println("\n3. Initializing WalletAdvertiser...")
	if err := advertiser.Init(); err != nil {
		log.Fatalf("Failed to initialize WalletAdvertiser: %v", err)
	}
	fmt.Printf("✓ WalletAdvertiser initialized successfully\n")
	fmt.Printf("  Initialized: %v\n", advertiser.IsInitialized())

	// Create some example advertisements
	fmt.Println("\n4. Creating advertisements...")
	adsData := []*oa.AdvertisementData{
		{
			Protocol:           overlay.ProtocolSHIP,
			TopicOrServiceName: "payments",
		},
		{
			Protocol:           overlay.ProtocolSLAP,
			TopicOrServiceName: "identity_verification",
		},
	}

	// Note: This will fail in the current implementation since BSV SDK integration is not complete
	_, err = advertiser.CreateAdvertisements(adsData)
	if err != nil {
		fmt.Printf("⚠ CreateAdvertisements failed (expected): %v\n", err)
		fmt.Println("   This is expected as BSV SDK integration is not yet implemented")
	}

	// Parse an example advertisement
	fmt.Println("\n5. Parsing an advertisement...")
	outputScriptBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // Mock script
	outputScript := script.NewFromBytes(outputScriptBytes)
	advertisement, err := advertiser.ParseAdvertisement(outputScript)
	if err != nil {
		fmt.Printf("Failed to parse advertisement: %v\n", err)
	} else {
		fmt.Printf("✓ Advertisement parsed successfully:\n")
		fmt.Printf("  Protocol: %s\n", advertisement.Protocol)
		fmt.Printf("  Identity Key: %s\n", advertisement.IdentityKey)
		fmt.Printf("  Domain: %s\n", advertisement.Domain)
		fmt.Printf("  Topic/Service: %s\n", advertisement.TopicOrService)
	}

	// Find all advertisements for a protocol
	fmt.Println("\n6. Finding advertisements...")
	_, err = advertiser.FindAllAdvertisements(overlay.ProtocolSHIP)
	if err != nil {
		fmt.Printf("⚠ FindAllAdvertisements failed (expected): %v\n", err)
		fmt.Println("   This is expected as storage integration is not yet implemented")
	}

	fmt.Println("\n✓ Example completed successfully!")
	fmt.Println("\nNote: Some operations failed as expected because they require:")
	fmt.Println("- BSV SDK integration for transaction creation and signing")
	fmt.Println("- Storage backend integration for persistence")
	fmt.Println("- Real PushDrop decoder implementation")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
