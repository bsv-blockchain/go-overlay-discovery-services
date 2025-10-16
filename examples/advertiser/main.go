// Package main demonstrates how to use the WalletAdvertiser for creating and managing
// SHIP and SLAP overlay advertisements.
package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/advertiser"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	logger.Info("BSV Overlay Discovery Services - WalletAdvertiser Example")
	logger.Info("========================================================")

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
	logger.Info("Creating WalletAdvertiser", slog.String("step", "1"))
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
	logger.Info("WalletAdvertiser created successfully",
		slog.String("chain", advertiser.GetChain()),
		slog.String("storageURL", advertiser.GetStorageURL()),
		slog.String("advertisableURI", advertiser.GetAdvertisableURI()))

	// Set up mock dependencies (in a real scenario, these would be actual implementations)
	logger.Info("Setting up dependencies", slog.String("step", "2"))
	advertiser.SetSkipStorageValidation(true) // Skip storage validation for example
	logger.Info("Dependencies configured")

	// Initialize the advertiser
	logger.Info("Initializing WalletAdvertiser", slog.String("step", "3"))
	if err = advertiser.Init(); err != nil {
		log.Fatalf("Failed to initialize WalletAdvertiser: %v", err)
	}
	logger.Info("WalletAdvertiser initialized successfully",
		slog.Bool("initialized", advertiser.IsInitialized()))

	// Create some example advertisements
	logger.Info("Creating advertisements", slog.String("step", "4"))
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

	// This will fail in the current implementation since BSV SDK integration is not complete
	_, err = advertiser.CreateAdvertisements(adsData)
	if err != nil {
		logger.Warn("CreateAdvertisements failed (expected)", slog.String("error", err.Error()))
		logger.Info("This is expected as BSV SDK integration is not yet implemented")
	}

	// Parse an example advertisement
	logger.Info("Parsing an advertisement", slog.String("step", "5"))
	outputScriptBytes := []byte{0x01, 0x02, 0x03, 0x04, 0x05} // Mock script
	outputScript := script.NewFromBytes(outputScriptBytes)
	advertisement, err := advertiser.ParseAdvertisement(outputScript)
	if err != nil {
		logger.Error("Failed to parse advertisement", slog.String("error", err.Error()))
	} else {
		logger.Info("Advertisement parsed successfully",
			slog.Any("protocol", advertisement.Protocol),
			slog.String("identityKey", advertisement.IdentityKey),
			slog.String("domain", advertisement.Domain),
			slog.String("topicService", advertisement.TopicOrService))
	}

	// Find all advertisements for a protocol
	logger.Info("Finding advertisements", slog.String("step", "6"))
	_, err = advertiser.FindAllAdvertisements(overlay.ProtocolSHIP)
	if err != nil {
		logger.Warn("FindAllAdvertisements failed (expected)", slog.String("error", err.Error()))
		logger.Info("This is expected as storage integration is not yet implemented")
	}

	logger.Info("Example completed successfully")
	logger.Info("Note: Some operations failed as expected because they require:")
	logger.Info("- BSV SDK integration for transaction creation and signing")
	logger.Info("- Storage backend integration for persistence")
	logger.Info("- Real PushDrop decoder implementation")
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
