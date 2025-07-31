// Package main demonstrates usage of the utility functions for overlay discovery services
package main

import (
	"fmt"
	"log"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

func main() {
	fmt.Println("=== Overlay Discovery Services Utility Examples ===")
	fmt.Println()

	// Example 1: URI validation
	fmt.Println("1. URI Validation Examples:")
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
		status := "✓ Valid"
		if !isValid {
			status = "✗ Invalid"
		}
		fmt.Printf("  %s: %s\n", status, uri)
	}

	// Example 2: Topic/Service name validation
	fmt.Println("\n2. Topic/Service Name Validation Examples:")
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
		status := "✓ Valid"
		if !isValid {
			status = "✗ Invalid"
		}
		fmt.Printf("  %s: %s\n", status, name)
	}

	// Example 3: Helper functions
	fmt.Println("\n3. Helper Function Examples:")

	// Hex conversion examples
	testBytes := []byte{0x01, 0x23, 0xab, 0xcd}
	hexString := utils.BytesToHex(testBytes)
	fmt.Printf("  Bytes to Hex: %v -> %s\n", testBytes, hexString)

	backToBytes, err := utils.HexToBytes(hexString)
	if err != nil {
		log.Printf("Error converting hex to bytes: %v", err)
	} else {
		fmt.Printf("  Hex to Bytes: %s -> %v\n", hexString, backToBytes)
	}

	// UTF-8 conversion
	utf8Bytes := []byte("Hello, 世界!")
	utf8String := utils.UTFBytesToString(utf8Bytes)
	fmt.Printf("  UTF-8 Bytes to String: %v -> %s\n", utf8Bytes, utf8String)

	// Example 4: Token signature validation (with mock wallet)
	fmt.Println("\n4. Token Signature Validation Example (Mock):")

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
	mockWallet := &utils.MockWallet{}

	isValid, err := utils.IsTokenSignatureCorrectlyLinked(lockingPubKey, tokenFields, mockWallet)
	if err != nil {
		fmt.Printf("  Token validation error (expected with mock wallet): %v\n", err)
	} else {
		status := "✗ Invalid"
		if isValid {
			status = "✓ Valid"
		}
		fmt.Printf("  Token signature validation: %s\n", status)
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Note: Token signature validation requires a real BSV SDK wallet implementation.")
	fmt.Println("The MockWallet is provided for testing and will always return errors.")
}
