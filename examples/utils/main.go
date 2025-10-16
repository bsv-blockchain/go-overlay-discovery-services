// Package main demonstrates usage of the utility functions for overlay discovery services
package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

//nolint:gochecknoglobals // logger is used across multiple example functions
var logger = slog.New(slog.NewTextHandler(os.Stdout, nil))

func main() {
	logger.Info("=== Overlay Discovery Services Utility Examples ===")

	// Example 1: URI validation
	logger.Info("1. URI Validation Examples:")
	testURIs := []string{
		"https://example.com/",
		"https://localhost/",
		"wss://overlay-service.com",
		"https+bsvauth+smf://api.example.com/",
		"js8c+bsvauth+smf:?lat=40.7128&long=-74.0060&freq=7.078&radius=100",
		"http://example.com", // Should be invalid
	}

	for _, uri := range testURIs {
		isValid := utils.IsAdvertisableURI(uri)
		status := "Valid"
		if !isValid {
			status = "Invalid"
		}
		logger.Info("URI validation result", "status", status, "uri", uri)
	}

	// Example 2: Topic/Service name validation
	logger.Info("2. Topic/Service Name Validation Examples:")
	testNames := []string{
		"tm_payments",
		"ls_identity_verification",
		"tm_chat_messages_system",
		"payments",       // Invalid - no prefix
		"TM_payments",    // Invalid - uppercase
		"tm_payments123", // Invalid - contains numbers
		"tm_",            // Invalid - empty after prefix
	}

	for _, name := range testNames {
		isValid := utils.IsValidTopicOrServiceName(name)
		status := "Valid"
		if !isValid {
			status = "Invalid"
		}
		logger.Info("Topic/Service name validation result", "status", status, "name", name)
	}

	// Example 3: Helper functions
	logger.Info("3. Helper Function Examples:")

	// Hex conversion examples
	testBytes := []byte{0x01, 0x23, 0xab, 0xcd}
	hexString := utils.BytesToHex(testBytes)
	logger.Info("Bytes to Hex conversion", "bytes", testBytes, "hex", hexString)

	backToBytes, err := utils.HexToBytes(hexString)
	if err != nil {
		log.Printf("Error converting hex to bytes: %v", err)
	} else {
		logger.Info("Hex to Bytes conversion", "hex", hexString, "bytes", backToBytes)
	}

	// Example 4: Token signature validation (with mock wallet)
	logger.Info("4. Token Signature Validation Example (Mock):")

	// Create mock token fields for demonstration
	protocol := []byte("SHIP")
	identityKey := []byte{0x01, 0x02, 0x03, 0x04}
	extraData := []byte("example data")
	signature := []byte{0xff, 0xee, 0xdd}

	tokenFields := utils.TokenFields{
		protocol,
		identityKey,
		extraData,
		signature,
	}

	lockingPubKey := "03abc123def456"

	isValid, err := utils.IsTokenSignatureCorrectlyLinked(context.TODO(), lockingPubKey, tokenFields)
	if err != nil {
		logger.Info("Token validation error (expected with mock wallet)", "error", err)
	} else {
		status := "Invalid"
		if isValid {
			status = "Valid"
		}
		logger.Info("Token signature validation result", "status", status)
	}

	logger.Info("=== Example Complete ===")
	logger.Info("Note: Token signature validation requires a real BSV SDK wallet implementation.")
	logger.Info("The MockWallet is provided for testing and will always return errors.")
}
