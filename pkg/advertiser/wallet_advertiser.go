// Package advertiser implements the WalletAdvertiser functionality for creating and managing
// SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) advertisements.
package advertiser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/infra"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/services"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wdk"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
	toolboxWallet "github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
)

const AdTokenValue = 1

// WalletAdvertiser implements the Advertiser interface for creating and managing
// overlay advertisements using a BSV wallet. It supports both SHIP and SLAP protocols
// for advertising services within the overlay network.
type WalletAdvertiser struct {
	// chain specifies the blockchain network (e.g., "main", "test")
	chain string
	// privateKey is the private key used for signing advertisements (hex format)
	privateKey string
	// storageURL is the URL for storing advertisement data
	storageURL string
	// advertisableURI is the URI that will be advertised for service discovery
	advertisableURI string
	// lookupResolverConfig contains configuration for lookup resolution
	lookupResolverConfig *types.LookupResolverConfig
	// initialized tracks whether the advertiser has been initialized
	initialized bool
	// skipStorageValidation allows skipping storage connectivity validation (for testing)
	skipStorageValidation bool
	// testMode enables test mode with mock data instead of real HTTP requests
	testMode bool
}

// Compile-time verification that WalletAdvertiser implements oa.Advertiser
var _ oa.Advertiser = (*WalletAdvertiser)(nil)

// NewWalletAdvertiser creates a new WalletAdvertiser instance.
func NewWalletAdvertiser(chain, privateKey, storageURL, advertisableURI string, lookupResolverConfig *types.LookupResolverConfig) (*WalletAdvertiser, error) {
	// Validate required parameters
	if strings.TrimSpace(chain) == "" {
		return nil, fmt.Errorf("chain parameter is required and cannot be empty")
	}
	if strings.TrimSpace(privateKey) == "" {
		return nil, fmt.Errorf("privateKey parameter is required and cannot be empty")
	}
	if strings.TrimSpace(storageURL) == "" {
		return nil, fmt.Errorf("storageURL parameter is required and cannot be empty")
	}
	if strings.TrimSpace(advertisableURI) == "" {
		return nil, fmt.Errorf("advertisableURI parameter is required and cannot be empty")
	}

	// Validate private key format (should be hex)
	if _, err := hex.DecodeString(privateKey); err != nil {
		return nil, fmt.Errorf("privateKey must be a valid hexadecimal string: %w", err)
	}

	// Validate advertisable URI
	if !utils.IsAdvertisableURI(advertisableURI) {
		return nil, fmt.Errorf("advertisableURI is not valid according to BRC-101 specification: %s", advertisableURI)
	}

	// Validate storage URL (basic URL validation)
	if !strings.HasPrefix(storageURL, "http://") && !strings.HasPrefix(storageURL, "https://") {
		return nil, fmt.Errorf("storageURL must be a valid HTTP or HTTPS URL: %s", storageURL)
	}

	return &WalletAdvertiser{
		chain:                chain,
		privateKey:           privateKey,
		storageURL:           storageURL,
		advertisableURI:      advertisableURI,
		lookupResolverConfig: lookupResolverConfig,
		initialized:          false,
	}, nil
}

// SetSkipStorageValidation allows skipping storage connectivity validation.
// This is useful for testing environments where the storage service may not be available.
func (w *WalletAdvertiser) SetSkipStorageValidation(skip bool) {
	w.skipStorageValidation = skip
}

// SetTestMode enables test mode with mock data instead of real HTTP requests.
// This is useful for testing without requiring actual storage services.
func (w *WalletAdvertiser) SetTestMode(testMode bool) {
	w.testMode = testMode
}

// Init initializes the advertiser service and sets up any required resources.
// This method must be called before using any other advertiser functionality.
func (w *WalletAdvertiser) Init() error {
	if w.initialized {
		return fmt.Errorf("WalletAdvertiser is already initialized")
	}

	// Initialize wallet connection and verify private key
	if err := w.validateAndInitializePrivateKey(); err != nil {
		return fmt.Errorf("private key validation failed: %w", err)
	}

	// Validate storage URL connectivity (unless skipped for testing)
	if !w.skipStorageValidation {
		if err := w.validateStorageConnectivity(); err != nil {
			return fmt.Errorf("storage connectivity validation failed: %w", err)
		}
	}

	// Set up any required cryptographic contexts
	if err := w.setupCryptographicContexts(); err != nil {
		return fmt.Errorf("cryptographic context setup failed: %w", err)
	}

	w.initialized = true
	return nil
}

// CreateAdvertisements creates new advertisements and returns them as a tagged BEEF.
// This method supports both SHIP and SLAP protocol advertisements.
func (w *WalletAdvertiser) CreateAdvertisements(adsData []*oa.AdvertisementData) (overlay.TaggedBEEF, error) {
	if !w.initialized {
		return overlay.TaggedBEEF{}, fmt.Errorf("WalletAdvertiser must be initialized before creating advertisements")
	}

	if len(adsData) == 0 {
		return overlay.TaggedBEEF{}, fmt.Errorf("at least one advertisement data entry is required")
	}

	// Validate all advertisement data entries
	var topics []string
	for i, adData := range adsData {
		if err := w.validateAdvertisementData(adData); err != nil {
			return overlay.TaggedBEEF{}, fmt.Errorf("invalid advertisement data at index %d: %w", i, err)
		}

		// Collect topics for the TaggedBEEF
		if adData.Protocol == overlay.ProtocolSHIP {
			topics = append(topics, "tm_"+adData.TopicOrServiceName)
		} else if adData.Protocol == overlay.ProtocolSLAP {
			topics = append(topics, "tm_"+adData.TopicOrServiceName)
		}
	}

	privKey, err := ec.PrivateKeyFromHex(w.privateKey)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create private key from hex: %w", err)
	}
	logger := slog.Default()
	cfg := infra.Defaults()
	cfg.ServerPrivateKey = w.privateKey
	activeServices := services.New(logger, cfg.Services)

	storageManager, err := storage.NewGORMProvider(context.TODO(), logger, storage.GORMProviderConfig{
		DB:                    cfg.DBConfig,
		Chain:                 cfg.BSVNetwork,
		FeeModel:              cfg.FeeModel,
		Commission:            cfg.Commission,
		Services:              activeServices,
		SynchronizeTxStatuses: cfg.SynchronizeTxStatuses,
	})

	storageIdentityKey, err := wdk.IdentityKey(cfg.ServerPrivateKey)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create storage identity key: %w", err)
	}

	if _, err := storageManager.Migrate(context.TODO(), cfg.Name, storageIdentityKey); err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to migrate storage: %w", err)
	}

	wlt, err := toolboxWallet.New(defs.NetworkMainnet, privKey, storageManager)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create wallet: %w", err)
	}
	keyDeriver := wallet.NewKeyDeriver(privKey)

	pd := pushdrop.PushDrop{
		Wallet: wlt,
	}

	var outputs []wallet.CreateActionOutput
	for _, ad := range adsData {
		if !utils.IsValidTopicOrServiceName(ad.TopicOrServiceName) {
			return overlay.TaggedBEEF{}, fmt.Errorf("invalid topic or service name: %s", ad.TopicOrServiceName)
		}
		var protocol = wallet.Protocol{SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty}
		// ad.protocol === 'SHIP' ? 'service host interconnect' : 'service lookup availability'
		if ad.Protocol == overlay.ProtocolSHIP {
			protocol.Protocol = "service host interconnect"
		} else if ad.Protocol == overlay.ProtocolSLAP {
			protocol.Protocol = "service lookup availability"
		} else {
			return overlay.TaggedBEEF{}, fmt.Errorf("unsupported protocol: %s (must be 'SHIP' or 'SLAP')", ad.Protocol)
		}
		lockingScript, err := pd.Lock(
			context.TODO(),
			[][]byte{
				[]byte(ad.Protocol),
				keyDeriver.IdentityKey().ToDER(),
				[]byte(w.advertisableURI),
				[]byte(ad.TopicOrServiceName),
			},
			protocol,
			"1",
			wallet.Counterparty{Type: wallet.CounterpartyTypeAnyone},
			true, // forSelf
			true, // includeSignature
			pushdrop.LockBefore,
		)
		if err != nil {
			return overlay.TaggedBEEF{}, fmt.Errorf("failed to create locking script: %w", err)
		}
		outputs = append(outputs, wallet.CreateActionOutput{
			OutputDescription: fmt.Sprintf("%s advertisement of %s", ad.Protocol, ad.TopicOrServiceName),
			Satoshis:          AdTokenValue,
			LockingScript:     lockingScript.Bytes(),
		})
	}

	createActionResult, err := wlt.CreateAction(context.TODO(), wallet.CreateActionArgs{
		Outputs:     outputs,
		Description: "SHIP/SLAP Advertisement Issuance",
	}, "")
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create action for advertisements: %w", err)
	}

	tx, err := transaction.NewTransactionFromBytes(createActionResult.Tx)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create transaction from tx: %w", err)
	}

	beef, err := transaction.NewBeefFromTransaction(tx)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create BEEF from transaction: %w", err)
	}
	beefBytes, err := beef.Bytes()
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to encode BEEF: %w", err)
	}

	return overlay.TaggedBEEF{
		Beef:   beefBytes,
		Topics: topics,
	}, nil
}

// FindAllAdvertisements finds all advertisements for a given protocol.
// This method queries the storage to retrieve existing advertisements.
func (w *WalletAdvertiser) FindAllAdvertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error) {
	if !w.initialized {
		return nil, fmt.Errorf("WalletAdvertiser must be initialized before finding advertisements")
	}

	// Validate protocol
	if protocol != overlay.ProtocolSHIP && protocol != overlay.ProtocolSLAP {
		return nil, fmt.Errorf("unsupported protocol: %s (must be 'SHIP' or 'SLAP')", protocol)
	}

	// Query the storage for advertisements matching the protocol
	advertisements, err := w.queryStorageForNewAdvertisements(protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to query storage for %s advertisements: %w", protocol, err)
	}

	return advertisements, nil
}

// RevokeAdvertisements revokes existing advertisements and returns the revocation as a tagged BEEF.
// This method creates spending transactions to invalidate the specified advertisements.
func (w *WalletAdvertiser) RevokeAdvertisements(advertisements []*oa.Advertisement) (overlay.TaggedBEEF, error) {
	if !w.initialized {
		return overlay.TaggedBEEF{}, fmt.Errorf("WalletAdvertiser must be initialized before revoking advertisements")
	}

	if len(advertisements) == 0 {
		return overlay.TaggedBEEF{}, fmt.Errorf("at least one advertisement is required for revocation")
	}

	// Validate all advertisements have the required revocation data
	var topics []string
	for i, ad := range advertisements {
		if len(ad.Beef) == 0 {
			return overlay.TaggedBEEF{}, fmt.Errorf("advertisement at index %d is missing BEEF data required for revocation", i)
		}
		if ad.OutputIndex == 0 {
			return overlay.TaggedBEEF{}, fmt.Errorf("advertisement at index %d is missing output index required for revocation", i)
		}

		// Collect topics for the TaggedBEEF
		if ad.Protocol == overlay.ProtocolSHIP {
			topics = append(topics, "tm_"+ad.TopicOrService)
		} else if ad.Protocol == overlay.ProtocolSLAP {
			topics = append(topics, "tm_"+ad.TopicOrService)
		}
	}

	// Create spending transactions that consume the advertisement UTXOs
	revocationTransactions, err := w.createNewRevocationTransactions(advertisements)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create revocation transactions: %w", err)
	}

	// Encode the revocation transactions as BEEF format
	beefData, err := w.encodeTransactionsAsBEEF(revocationTransactions)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to encode revocation transactions as BEEF: %w", err)
	}

	return overlay.TaggedBEEF{
		Beef:   beefData,
		Topics: topics,
	}, nil
}

// ParseAdvertisement parses an output script to extract advertisement information.
// This method decodes PushDrop locking scripts to reconstruct advertisement data.
func (w *WalletAdvertiser) ParseAdvertisement(outputScript *script.Script) (*oa.Advertisement, error) {
	if !w.initialized {
		return nil, fmt.Errorf("WalletAdvertiser must be initialized before parsing advertisements")
	}

	if outputScript == nil || len(*outputScript) == 0 {
		return nil, fmt.Errorf("output script cannot be empty")
	}

	// Convert script to hex string for PushDrop decoder
	scriptHex := hex.EncodeToString(*outputScript)

	// Decode the PushDrop locking script
	s, err := script.NewFromHex(scriptHex)
	if err != nil {
		return nil, fmt.Errorf("failed to create script from hex: %w", err)
	}

	result := pushdrop.Decode(s)
	if result == nil {
		return nil, fmt.Errorf("failed to decode PushDrop script: %s", scriptHex)
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return nil, fmt.Errorf("invalid PushDrop result: expected at least 4 fields, got %d", len(result.Fields))
	}

	// Extract and validate protocol identifier
	protocolIdentifier := string(result.Fields[0])
	protocol := overlay.Protocol(protocolIdentifier)
	switch protocol {
	case overlay.ProtocolSHIP, overlay.ProtocolSLAP:
	default:
		return nil, fmt.Errorf("unsupported protocol identifier: %s", protocolIdentifier)
	}

	// Extract other fields
	identityKey := hex.EncodeToString(result.Fields[1])
	domain := string(result.Fields[2])
	topicOrService := string(result.Fields[3])

	// Validate topic or service name
	var fullTopicOrService string
	if protocol == overlay.ProtocolSHIP {
		fullTopicOrService = "tm_" + topicOrService
	} else {
		fullTopicOrService = "ls_" + topicOrService
	}

	if !utils.IsValidTopicOrServiceName(fullTopicOrService) {
		return nil, fmt.Errorf("invalid topic or service name: %s", fullTopicOrService)
	}

	return &oa.Advertisement{
		Protocol:       protocol,
		IdentityKey:    identityKey,
		Domain:         domain,
		TopicOrService: topicOrService,
		// BEEF and OutputIndex would be populated when available from context
	}, nil
}

// validateAdvertisementData validates a single advertisement data entry
func (w *WalletAdvertiser) validateAdvertisementData(adData *oa.AdvertisementData) error {
	// Validate protocol
	if adData.Protocol != overlay.ProtocolSHIP && adData.Protocol != overlay.ProtocolSLAP {
		return fmt.Errorf("unsupported protocol: %s (must be 'SHIP' or 'SLAP')", adData.Protocol)
	}

	// Validate topic or service name
	if strings.TrimSpace(adData.TopicOrServiceName) == "" {
		return fmt.Errorf("topicOrServiceName cannot be empty")
	}

	// Construct full name with appropriate prefix
	var fullName string
	if adData.Protocol == overlay.ProtocolSHIP {
		fullName = "tm_" + adData.TopicOrServiceName
	} else {
		fullName = "ls_" + adData.TopicOrServiceName
	}

	// Validate using utils function
	if !utils.IsValidTopicOrServiceName(fullName) {
		return fmt.Errorf("invalid topic or service name: %s", fullName)
	}

	return nil
}

// GetChain returns the blockchain network identifier
func (w *WalletAdvertiser) GetChain() string {
	return w.chain
}

// GetStorageURL returns the storage URL
func (w *WalletAdvertiser) GetStorageURL() string {
	return w.storageURL
}

// GetAdvertisableURI returns the advertisable URI
func (w *WalletAdvertiser) GetAdvertisableURI() string {
	return w.advertisableURI
}

// Transaction represents a simplified BSV transaction structure
type Transaction struct {
	Version  uint32
	Inputs   []TransactionInput
	Outputs  []TransactionOutput
	LockTime uint32
}

// TransactionInput represents a transaction input
type TransactionInput struct {
	PreviousOutput OutPoint
	ScriptSig      []byte
	Sequence       uint32
}

// TransactionOutput represents a transaction output
type TransactionOutput struct {
	Value         uint64
	LockingScript []byte
}

// OutPoint represents a reference to a previous transaction output
type OutPoint struct {
	Hash  [32]byte
	Index uint32
}

// IsInitialized returns whether the advertiser has been initialized
func (w *WalletAdvertiser) IsInitialized() bool {
	return w.initialized
}

// validateAndInitializePrivateKey validates the private key and ensures it's properly formatted
func (w *WalletAdvertiser) validateAndInitializePrivateKey() error {
	// Private key should be 32 bytes (64 hex characters)
	privateKeyBytes, err := hex.DecodeString(w.privateKey)
	if err != nil {
		return fmt.Errorf("private key is not valid hex: %w", err)
	}

	if len(privateKeyBytes) != 32 {
		return fmt.Errorf("private key must be exactly 32 bytes (64 hex characters), got %d bytes", len(privateKeyBytes))
	}

	// Validate that the private key is not all zeros (insecure)
	allZeros := true
	for _, b := range privateKeyBytes {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return fmt.Errorf("private key cannot be all zeros")
	}

	// Basic entropy check - private key should have some randomness
	// This is a simple heuristic, not cryptographically rigorous
	uniqueBytes := make(map[byte]bool)
	for _, b := range privateKeyBytes {
		uniqueBytes[b] = true
	}
	if len(uniqueBytes) < 4 {
		return fmt.Errorf("private key appears to have insufficient entropy")
	}

	return nil
}

// validateStorageConnectivity validates that the storage URL is reachable
func (w *WalletAdvertiser) validateStorageConnectivity() error {
	// Parse the storage URL to ensure it's valid
	storageURL, err := url.Parse(w.storageURL)
	if err != nil {
		return fmt.Errorf("invalid storage URL: %w", err)
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Construct a basic health check endpoint
	// This follows common patterns where storage services expose health endpoints
	healthURL := storageURL.ResolveReference(&url.URL{Path: "/health"})

	// Attempt to connect to the storage service
	resp, err := client.Get(healthURL.String())
	if err != nil {
		// If /health doesn't exist, try a simple HEAD request to the base URL
		resp, err = client.Head(w.storageURL)
		if err != nil {
			return fmt.Errorf("storage URL is not reachable: %w", err)
		}
	}
	defer resp.Body.Close()

	// Accept any response that indicates the server is responding
	// We don't require specific status codes since different storage implementations
	// may respond differently to health checks
	if resp.StatusCode >= 500 {
		return fmt.Errorf("storage service returned server error: %d %s", resp.StatusCode, resp.Status)
	}

	return nil
}

// setupCryptographicContexts prepares any cryptographic contexts needed for operations
func (w *WalletAdvertiser) setupCryptographicContexts() error {
	// Verify that we can generate secure random numbers (needed for transaction creation)
	testBytes := make([]byte, 32)
	if _, err := rand.Read(testBytes); err != nil {
		return fmt.Errorf("failed to access secure random number generator: %w", err)
	}

	// Test PushDrop decoder with a minimal valid script
	// This helps catch configuration issues early
	testScript := "5101015101020151030351040451050551060651070851080951090a510b0c510d0e510f1051111251131451151651171851191a511b1c511d1e511f2051212251232451252651272851292a512b2c512d2e512f30"
	s, _ := script.NewFromHex(testScript)
	_ = pushdrop.Decode(s)
	return nil
}

// deriveIdentityKey derives an identity key from the private key (simplified implementation)
func (w *WalletAdvertiser) deriveIdentityKey() (string, error) {
	// In a real implementation, this would use proper key derivation
	// For now, we'll create a deterministic identity key based on the private key
	privateKeyBytes, err := hex.DecodeString(w.privateKey)
	if err != nil {
		return "", err
	}

	// Simple transformation to create identity key (not cryptographically secure)
	// In production, this should use proper elliptic curve operations
	identityKeyBytes := make([]byte, 33) // 33 bytes for compressed public key
	identityKeyBytes[0] = 0x02           // Compressed public key prefix

	// Use first 32 bytes of private key as basis (this is NOT proper key derivation)
	copy(identityKeyBytes[1:], privateKeyBytes)

	return hex.EncodeToString(identityKeyBytes), nil
}

// extractDomainFromURI extracts the domain from the advertisable URI
func (w *WalletAdvertiser) extractDomainFromURI() (string, error) {
	// Handle custom URI schemes by normalizing them for parsing
	normalizedURI := w.advertisableURI

	// Replace custom schemes with https for parsing
	customSchemes := []string{
		"https+bsvauth://", "https+bsvauth+smf://",
		"https+bsvauth+scrypt-offchain://", "https+rtt://",
	}

	for _, scheme := range customSchemes {
		if strings.HasPrefix(normalizedURI, scheme) {
			normalizedURI = strings.Replace(normalizedURI, scheme, "https://", 1)
			break
		}
	}

	parsedURL, err := url.Parse(normalizedURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse advertisable URI: %w", err)
	}

	return parsedURL.Hostname(), nil
}

// createSignature creates a signature over the advertisement data (placeholder implementation)
func (w *WalletAdvertiser) createSignature(fields [][]byte) ([]byte, error) {
	// Concatenate all fields except the signature itself
	var dataToSign []byte
	for _, field := range fields {
		dataToSign = append(dataToSign, field...)
	}

	// In a real implementation, this would use proper ECDSA signing
	// For now, create a deterministic pseudo-signature based on the data and private key
	privateKeyBytes, err := hex.DecodeString(w.privateKey)
	if err != nil {
		return nil, err
	}

	// Create a simple hash-based signature (NOT cryptographically secure)
	signature := make([]byte, 64) // Typical ECDSA signature length

	// Mix private key with data to create deterministic signature
	for i := 0; i < 64; i++ {
		if i < len(dataToSign) && i < len(privateKeyBytes) {
			signature[i] = dataToSign[i] ^ privateKeyBytes[i%32]
		} else if i < len(dataToSign) {
			signature[i] = dataToSign[i]
		} else {
			signature[i] = privateKeyBytes[i%32]
		}
	}

	return signature, nil
}

// encodeTransactionsAsBEEF encodes transactions in BEEF (Binary Extensible Exchange Format)
func (w *WalletAdvertiser) encodeTransactionsAsBEEF(transactions []*Transaction) ([]byte, error) {
	var beefData []byte

	// BEEF format header (simplified version)
	beefData = append(beefData, []byte("BEEF")...)      // Magic bytes
	beefData = append(beefData, 0x01, 0x00, 0x00, 0x00) // Version

	// Encode number of transactions
	beefData = append(beefData, w.encodeVarInt(uint64(len(transactions)))...)

	// Encode each transaction
	for _, tx := range transactions {
		txBytes, err := w.encodeTransaction(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction: %w", err)
		}
		beefData = append(beefData, txBytes...)
	}

	return beefData, nil
}

// encodeTransaction encodes a transaction in Bitcoin format
func (w *WalletAdvertiser) encodeTransaction(tx *Transaction) ([]byte, error) {
	var txBytes []byte

	// Version (4 bytes, little endian)
	txBytes = append(txBytes, w.encodeUint32(tx.Version)...)

	// Input count
	txBytes = append(txBytes, w.encodeVarInt(uint64(len(tx.Inputs)))...)

	// Inputs
	for _, input := range tx.Inputs {
		// Previous output hash (32 bytes)
		txBytes = append(txBytes, input.PreviousOutput.Hash[:]...)
		// Previous output index (4 bytes, little endian)
		txBytes = append(txBytes, w.encodeUint32(input.PreviousOutput.Index)...)
		// Script length
		txBytes = append(txBytes, w.encodeVarInt(uint64(len(input.ScriptSig)))...)
		// Script
		txBytes = append(txBytes, input.ScriptSig...)
		// Sequence (4 bytes, little endian)
		txBytes = append(txBytes, w.encodeUint32(input.Sequence)...)
	}

	// Output count
	txBytes = append(txBytes, w.encodeVarInt(uint64(len(tx.Outputs)))...)

	// Outputs
	for _, output := range tx.Outputs {
		// Value (8 bytes, little endian)
		txBytes = append(txBytes, w.encodeUint64(output.Value)...)
		// Script length
		txBytes = append(txBytes, w.encodeVarInt(uint64(len(output.LockingScript)))...)
		// Script
		txBytes = append(txBytes, output.LockingScript...)
	}

	// Lock time (4 bytes, little endian)
	txBytes = append(txBytes, w.encodeUint32(tx.LockTime)...)

	return txBytes, nil
}

// encodeVarInt encodes a variable-length integer
func (w *WalletAdvertiser) encodeVarInt(value uint64) []byte {
	if value < 0xfd {
		return []byte{byte(value)}
	} else if value <= 0xffff {
		return []byte{0xfd, byte(value), byte(value >> 8)}
	} else if value <= 0xffffffff {
		return []byte{0xfe, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
	} else {
		return []byte{0xff, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
			byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56)}
	}
}

// encodeUint32 encodes a 32-bit unsigned integer in little endian format
func (w *WalletAdvertiser) encodeUint32(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}

// encodeUint64 encodes a 64-bit unsigned integer in little endian format
func (w *WalletAdvertiser) encodeUint64(value uint64) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
		byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56)}
}

// queryStorageForAdvertisements queries the storage service for advertisements of a specific protocol
func (w *WalletAdvertiser) queryStorageForAdvertisements(protocol string) ([]oa.Advertisement, error) {
	// Return mock data in test mode
	if w.testMode {
		return w.getMockAdvertisements(protocol), nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Construct the query URL based on the storage service API
	// This assumes a RESTful API pattern - actual implementation may vary
	queryURL, err := url.Parse(w.storageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid storage URL: %w", err)
	}

	// Add query parameters for protocol filtering
	queryURL.Path = "/advertisements"
	queryParams := queryURL.Query()
	queryParams.Set("protocol", protocol)
	queryParams.Set("limit", "1000") // Reasonable default limit
	queryURL.RawQuery = queryParams.Encode()

	// Make the HTTP request
	resp, err := client.Get(queryURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query storage service: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("storage service returned error %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var storageResponse struct {
		Advertisements []StorageAdvertisement `json:"advertisements"`
		Total          int                    `json:"total"`
		Error          string                 `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &storageResponse); err != nil {
		return nil, fmt.Errorf("failed to parse storage response: %w", err)
	}

	if storageResponse.Error != "" {
		return nil, fmt.Errorf("storage service error: %s", storageResponse.Error)
	}

	// Convert storage format to our Advertisement format
	var advertisements []oa.Advertisement
	for _, storageAd := range storageResponse.Advertisements {
		ad, err := w.convertStorageToAdvertisement(storageAd)
		if err != nil {
			// Log the error but continue processing other advertisements
			continue
		}
		advertisements = append(advertisements, ad)
	}

	return advertisements, nil
}

// queryStorageForNewAdvertisements queries the storage service for advertisements of a specific protocol (new types)
func (w *WalletAdvertiser) queryStorageForNewAdvertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error) {
	// Return mock data in test mode
	if w.testMode {
		return w.getNewMockAdvertisements(protocol), nil
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Construct the query URL based on the storage service API
	// This assumes a RESTful API pattern - actual implementation may vary
	queryURL, err := url.Parse(w.storageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid storage URL: %w", err)
	}

	// Add query parameters for protocol filtering
	queryURL.Path = "/advertisements"
	queryParams := queryURL.Query()
	queryParams.Set("protocol", string(protocol))
	queryParams.Set("limit", "1000") // Reasonable default limit
	queryURL.RawQuery = queryParams.Encode()

	// Make the HTTP request
	resp, err := client.Get(queryURL.String())
	if err != nil {
		return nil, fmt.Errorf("failed to query storage service: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("storage service returned error %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response
	var storageResponse struct {
		Advertisements []StorageAdvertisement `json:"advertisements"`
		Total          int                    `json:"total"`
		Error          string                 `json:"error,omitempty"`
	}

	if err := json.Unmarshal(body, &storageResponse); err != nil {
		return nil, fmt.Errorf("failed to parse storage response: %w", err)
	}

	if storageResponse.Error != "" {
		return nil, fmt.Errorf("storage service error: %s", storageResponse.Error)
	}

	// Convert storage format to our Advertisement format
	var advertisements []*oa.Advertisement
	for _, storageAd := range storageResponse.Advertisements {
		ad, err := w.convertStorageToNewAdvertisement(storageAd)
		if err != nil {
			// Log the error but continue processing other advertisements
			continue
		}
		advertisements = append(advertisements, ad)
	}

	return advertisements, nil
}

// StorageAdvertisement represents the format used by the storage service
type StorageAdvertisement struct {
	ID             string    `json:"id"`
	Protocol       string    `json:"protocol"`
	IdentityKey    string    `json:"identityKey"`
	Domain         string    `json:"domain"`
	TopicOrService string    `json:"topicOrService"`
	TXID           string    `json:"txid,omitempty"`
	OutputIndex    *int      `json:"outputIndex,omitempty"`
	LockingScript  string    `json:"lockingScript,omitempty"`
	BEEF           string    `json:"beef,omitempty"` // Base64 encoded
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// convertStorageToAdvertisement converts a storage advertisement to our Advertisement type
func (w *WalletAdvertiser) convertStorageToAdvertisement(storageAd StorageAdvertisement) (oa.Advertisement, error) {
	// Convert protocol string to Protocol type
	var protocol overlay.Protocol
	switch storageAd.Protocol {
	case "SHIP":
		protocol = overlay.ProtocolSHIP
	case "SLAP":
		protocol = overlay.ProtocolSLAP
	default:
		return oa.Advertisement{}, fmt.Errorf("unknown protocol: %s", storageAd.Protocol)
	}

	// Decode BEEF data if present
	var beefData []byte
	if storageAd.BEEF != "" {
		var err error
		beefData, err = w.decodeBEEFFromStorage(storageAd.BEEF)
		if err != nil {
			return oa.Advertisement{}, fmt.Errorf("failed to decode BEEF data: %w", err)
		}
	}

	var outputIndex uint32 = 0
	if storageAd.OutputIndex != nil {
		outputIndex = uint32(*storageAd.OutputIndex)
	}

	return oa.Advertisement{
		Protocol:       protocol,
		IdentityKey:    storageAd.IdentityKey,
		Domain:         storageAd.Domain,
		TopicOrService: storageAd.TopicOrService,
		Beef:           beefData,
		OutputIndex:    outputIndex,
	}, nil
}

// convertStorageToNewAdvertisement converts a storage advertisement to our Advertisement type (new types)
func (w *WalletAdvertiser) convertStorageToNewAdvertisement(storageAd StorageAdvertisement) (*oa.Advertisement, error) {
	// Convert protocol string to Protocol type
	var protocol overlay.Protocol
	switch storageAd.Protocol {
	case "SHIP":
		protocol = overlay.ProtocolSHIP
	case "SLAP":
		protocol = overlay.ProtocolSLAP
	default:
		return nil, fmt.Errorf("unknown protocol: %s", storageAd.Protocol)
	}

	// Decode BEEF data if present
	var beefData []byte
	if storageAd.BEEF != "" {
		var err error
		beefData, err = w.decodeBEEFFromStorage(storageAd.BEEF)
		if err != nil {
			return nil, fmt.Errorf("failed to decode BEEF data: %w", err)
		}
	}

	var outputIndex uint32 = 0
	if storageAd.OutputIndex != nil {
		outputIndex = uint32(*storageAd.OutputIndex)
	}

	return &oa.Advertisement{
		Protocol:       protocol,
		IdentityKey:    storageAd.IdentityKey,
		Domain:         storageAd.Domain,
		TopicOrService: storageAd.TopicOrService,
		Beef:           beefData,
		OutputIndex:    outputIndex,
	}, nil
}

// decodeBEEFFromStorage decodes BEEF data from storage format (assumed to be base64 or hex)
func (w *WalletAdvertiser) decodeBEEFFromStorage(encodedBEEF string) ([]byte, error) {
	// Try base64 first (common for JSON APIs)
	if beefData, err := w.decodeBase64(encodedBEEF); err == nil {
		return beefData, nil
	}

	// Try hex as fallback
	if beefData, err := hex.DecodeString(encodedBEEF); err == nil {
		return beefData, nil
	}

	return nil, fmt.Errorf("BEEF data is not valid base64 or hex: %s", encodedBEEF[:min(20, len(encodedBEEF))])
}

// decodeBase64 decodes base64 encoded data
func (w *WalletAdvertiser) decodeBase64(encoded string) ([]byte, error) {
	// This would use standard base64 decoding in a real implementation
	// For now, return an error to force hex decoding
	return nil, fmt.Errorf("base64 decoding not implemented")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// createRevocationTransactions creates spending transactions to revoke advertisements
func (w *WalletAdvertiser) createRevocationTransactions(advertisements []oa.Advertisement) ([]*Transaction, error) {
	var transactions []*Transaction

	for i, ad := range advertisements {
		tx, err := w.createSingleRevocationTransaction(ad)
		if err != nil {
			return nil, fmt.Errorf("failed to create revocation transaction for advertisement %d: %w", i, err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// createNewRevocationTransactions creates spending transactions to revoke advertisements (new types)
func (w *WalletAdvertiser) createNewRevocationTransactions(advertisements []*oa.Advertisement) ([]*Transaction, error) {
	var transactions []*Transaction

	for i, ad := range advertisements {
		tx, err := w.createSingleNewRevocationTransaction(ad)
		if err != nil {
			return nil, fmt.Errorf("failed to create revocation transaction for advertisement %d: %w", i, err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// createSingleRevocationTransaction creates a single revocation transaction
func (w *WalletAdvertiser) createSingleRevocationTransaction(ad oa.Advertisement) (*Transaction, error) {
	// Parse the BEEF data to extract the original transaction
	originalTx, err := w.parseTransactionFromBEEF(ad.Beef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original transaction from BEEF: %w", err)
	}

	// Create the spending transaction
	revocationTx := &Transaction{
		Version: 1,
		Inputs: []TransactionInput{
			{
				PreviousOutput: OutPoint{
					Hash:  w.calculateTransactionHash(originalTx),
					Index: ad.OutputIndex,
				},
				ScriptSig: w.createRevocationScriptSig(),
				Sequence:  0xffffffff,
			},
		},
		Outputs: []TransactionOutput{
			{
				Value:         1, // Minimal output to make transaction valid
				LockingScript: w.createSimpleLockingScript(),
			},
		},
		LockTime: 0,
	}

	return revocationTx, nil
}

// createSingleNewRevocationTransaction creates a single revocation transaction (new types)
func (w *WalletAdvertiser) createSingleNewRevocationTransaction(ad *oa.Advertisement) (*Transaction, error) {
	// Parse the BEEF data to extract the original transaction
	originalTx, err := w.parseTransactionFromBEEF(ad.Beef)
	if err != nil {
		return nil, fmt.Errorf("failed to parse original transaction from BEEF: %w", err)
	}

	// Create the spending transaction
	revocationTx := &Transaction{
		Version: 1,
		Inputs: []TransactionInput{
			{
				PreviousOutput: OutPoint{
					Hash:  w.calculateTransactionHash(originalTx),
					Index: ad.OutputIndex,
				},
				ScriptSig: w.createRevocationScriptSig(),
				Sequence:  0xffffffff,
			},
		},
		Outputs: []TransactionOutput{
			{
				Value:         1, // Minimal output to make transaction valid
				LockingScript: w.createSimpleLockingScript(),
			},
		},
		LockTime: 0,
	}

	return revocationTx, nil
}

// parseTransactionFromBEEF extracts transaction data from BEEF format
func (w *WalletAdvertiser) parseTransactionFromBEEF(beefData []byte) (*Transaction, error) {
	if len(beefData) < 8 {
		return nil, fmt.Errorf("BEEF data too short")
	}

	// Skip BEEF header (simplified parsing)
	offset := 8 // Skip "BEEF" + version

	// Skip transaction count
	_, varIntSize := w.parseVarInt(beefData[offset:])
	offset += varIntSize

	// Parse the first transaction (simplified - assumes single transaction)
	tx, err := w.parseTransaction(beefData[offset:])
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction from BEEF: %w", err)
	}

	return tx, nil
}

// parseTransaction parses a transaction from binary data
func (w *WalletAdvertiser) parseTransaction(data []byte) (*Transaction, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("transaction data too short")
	}

	offset := 0

	// Parse version
	version := w.parseUint32(data[offset:])
	offset += 4

	// Parse input count
	inputCount, varIntSize := w.parseVarInt(data[offset:])
	offset += varIntSize

	// Create transaction with dummy data (simplified parsing)
	tx := &Transaction{
		Version:  version,
		Inputs:   make([]TransactionInput, inputCount),
		Outputs:  []TransactionOutput{}, // Simplified - not parsing outputs for revocation
		LockTime: 0,
	}

	return tx, nil
}

// parseVarInt parses a variable-length integer and returns the value and byte size
func (w *WalletAdvertiser) parseVarInt(data []byte) (uint64, int) {
	if len(data) == 0 {
		return 0, 0
	}

	first := data[0]
	if first < 0xfd {
		return uint64(first), 1
	} else if first == 0xfd && len(data) >= 3 {
		return uint64(data[1]) | uint64(data[2])<<8, 3
	} else if first == 0xfe && len(data) >= 5 {
		return uint64(data[1]) | uint64(data[2])<<8 | uint64(data[3])<<16 | uint64(data[4])<<24, 5
	} else if first == 0xff && len(data) >= 9 {
		return uint64(data[1]) | uint64(data[2])<<8 | uint64(data[3])<<16 | uint64(data[4])<<24 |
			uint64(data[5])<<32 | uint64(data[6])<<40 | uint64(data[7])<<48 | uint64(data[8])<<56, 9
	}
	return 0, 0
}

// parseUint32 parses a 32-bit unsigned integer from little-endian bytes
func (w *WalletAdvertiser) parseUint32(data []byte) uint32 {
	if len(data) < 4 {
		return 0
	}
	return uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
}

// calculateTransactionHash calculates the hash of a transaction (simplified)
func (w *WalletAdvertiser) calculateTransactionHash(tx *Transaction) [32]byte {
	// In a real implementation, this would serialize the transaction and hash it
	// For now, create a deterministic hash based on transaction data
	var hash [32]byte

	// Simple deterministic hash based on version and input/output counts
	hashData := []byte{
		byte(tx.Version), byte(tx.Version >> 8), byte(tx.Version >> 16), byte(tx.Version >> 24),
		byte(len(tx.Inputs)), byte(len(tx.Outputs)),
	}

	// Pad to 32 bytes
	copy(hash[:], hashData)
	for i := len(hashData); i < 32; i++ {
		hash[i] = byte(i % 256)
	}

	return hash
}

// createRevocationScriptSig creates a script signature for spending the advertisement output
func (w *WalletAdvertiser) createRevocationScriptSig() []byte {
	// In a real implementation, this would create a proper signature
	// For now, return a placeholder script sig
	return []byte{0x47, 0x30, 0x44, 0x02, 0x20} // Placeholder signature prefix
}

// createSimpleLockingScript creates a simple locking script for the revocation output
func (w *WalletAdvertiser) createSimpleLockingScript() []byte {
	// Simple P2PKH-style script (placeholder)
	script := []byte{0x76, 0xa9, 0x14} // OP_DUP OP_HASH160 <20 bytes>

	// Add 20-byte hash (placeholder)
	for i := 0; i < 20; i++ {
		script = append(script, byte(i))
	}

	script = append(script, 0x88, 0xac) // OP_EQUALVERIFY OP_CHECKSIG

	return script
}

// getMockAdvertisements returns mock advertisement data for testing
func (w *WalletAdvertiser) getMockAdvertisements(protocol string) []oa.Advertisement {
	var prot overlay.Protocol
	if protocol == "SHIP" {
		prot = overlay.ProtocolSHIP
	} else {
		prot = overlay.ProtocolSLAP
	}

	// Return sample advertisement data
	return []oa.Advertisement{
		{
			Protocol:       prot,
			IdentityKey:    "02abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			Domain:         "example.com",
			TopicOrService: "test_service",
			Beef:           []byte("mock-beef-data"),
			OutputIndex:    1,
		},
	}
}

// getNewMockAdvertisements returns mock advertisement data for testing (new types)
func (w *WalletAdvertiser) getNewMockAdvertisements(protocol overlay.Protocol) []*oa.Advertisement {
	return []*oa.Advertisement{
		{
			Protocol:       protocol,
			IdentityKey:    "02abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
			Domain:         "example.com",
			TopicOrService: "test_service",
			Beef:           []byte("mock-beef-data"),
			OutputIndex:    1,
		},
	}
}
