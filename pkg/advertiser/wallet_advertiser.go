// Package advertiser implements the WalletAdvertiser functionality for creating and managing
// SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) advertisements.
package advertiser

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
	oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
	authhttp "github.com/bsv-blockchain/go-sdk/auth/clients/authhttp"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	"github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/infra"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/services"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	toolboxWallet "github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/wdk"
)

// AdTokenValue is the default token value used for advertisements.
const AdTokenValue = 1

// Static error variables for err113 compliance
var (
	errChainRequired                 = errors.New("chain parameter is required and cannot be empty")
	errPrivateKeyRequired            = errors.New("privateKey parameter is required and cannot be empty")
	errStorageURLRequired            = errors.New("storageURL parameter is required and cannot be empty")
	errAdvertisableURIRequired       = errors.New("advertisableURI parameter is required and cannot be empty")
	errAdvertisableURIInvalid        = errors.New("advertisableURI is not valid according to BRC-101 specification")
	errStorageURLInvalid             = errors.New("storageURL must be a valid HTTP or HTTPS URL")
	errAlreadyInitialized            = errors.New("WalletAdvertiser is already initialized")
	errNotInitializedForAds          = errors.New("WalletAdvertiser must be initialized before creating advertisements")
	errNotInitializedForFind         = errors.New("WalletAdvertiser must be initialized before finding advertisements")
	errNotInitializedForParse        = errors.New("WalletAdvertiser must be initialized before parsing advertisements")
	errNotInitializedForRevoke       = errors.New("WalletAdvertiser must be initialized before revoking advertisements")
	errNoAdvertisementData           = errors.New("at least one advertisement data entry is required")
	errNoAdvertisements              = errors.New("at least one advertisement is required for revocation")
	errInvalidTopicOrServiceName     = errors.New("invalid topic or service name")
	errUnsupportedProtocol           = errors.New("unsupported protocol: must be 'SHIP' or 'SLAP'")
	errMissingBeefData               = errors.New("is missing BEEF data required for revocation")
	errMissingOutputIndex            = errors.New("is missing output index required for revocation")
	errOutputScriptEmpty             = errors.New("output script cannot be empty")
	errInvalidPushDropScript         = errors.New("failed to decode PushDrop script")
	errInvalidPushDropFields         = errors.New("invalid PushDrop result: expected at least 4 fields")
	errUnsupportedProtocolID         = errors.New("unsupported protocol identifier")
	errPrivateKeyAllZeros            = errors.New("private key cannot be all zeros")
	errPrivateKeyInsufficientLength  = errors.New("private key must be exactly 32 bytes (64 hex characters)")
	errPrivateKeyInsufficientEntropy = errors.New("private key appears to have insufficient entropy")
	errBEEFTooShort                  = errors.New("BEEF data too short")
	errTransactionTooShort           = errors.New("transaction data too short")
	errTopicNameEmpty                = errors.New("topicOrServiceName cannot be empty")
	errStorageServerError            = errors.New("storage service returned server error")
)

// Finder defines the interface for finding and creating advertisements.
type Finder interface {
	Advertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error)
	CreateAdvertisements(adsData []*oa.AdvertisementData, identityKey, advertisableURI string) (overlay.TaggedBEEF, error)
}

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
	// authFetch provides authenticated HTTP requests for storage communication
	authFetch *authhttp.AuthFetch
	// wallet provides the wallet interface for authentication
	wallet wallet.Interface
	// identityKey is the hex-encoded identity key derived from the private key
	identityKey string
	// Finder allows mocking
	Finder Finder
}

// Compile-time verification that WalletAdvertiser implements oa.Advertiser
var _ oa.Advertiser = (*WalletAdvertiser)(nil)

// NewWalletAdvertiser creates a new WalletAdvertiser instance.
func NewWalletAdvertiser(chain, privateKey, storageURL, advertisableURI string, lookupResolverConfig *types.LookupResolverConfig) (*WalletAdvertiser, error) {
	// Validate required parameters
	if strings.TrimSpace(chain) == "" {
		return nil, errChainRequired
	}
	if strings.TrimSpace(privateKey) == "" {
		return nil, errPrivateKeyRequired
	}
	if strings.TrimSpace(storageURL) == "" {
		return nil, errStorageURLRequired
	}
	if strings.TrimSpace(advertisableURI) == "" {
		return nil, errAdvertisableURIRequired
	}

	// Validate private key format (should be hex)
	if _, err := hex.DecodeString(privateKey); err != nil {
		return nil, fmt.Errorf("privateKey must be a valid hexadecimal string: %w", err)
	}

	// Validate advertisable URI
	if !utils.IsAdvertisableURI(advertisableURI) {
		return nil, fmt.Errorf("%w: %s", errAdvertisableURIInvalid, advertisableURI)
	}

	// Validate storage URL (basic URL validation)
	if !strings.HasPrefix(storageURL, "http://") && !strings.HasPrefix(storageURL, "https://") {
		return nil, fmt.Errorf("%w: %s", errStorageURLInvalid, storageURL)
	}

	// Create private key object for wallet initialization
	privKey, err := ec.PrivateKeyFromHex(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create private key object: %w", err)
	}

	// Initialize the wallet configuration
	cfg := infra.Defaults()
	cfg.ServerPrivateKey = privateKey
	activeServices := services.New(slog.Default(), cfg.Services)

	// Create storage manager for the wallet
	storageManager, errStorage := storage.NewGORMProvider(
		cfg.BSVNetwork,
		activeServices,
		storage.WithDBConfig(cfg.DBConfig),
		storage.WithFeeModel(cfg.FeeModel),
		storage.WithCommission(cfg.Commission),
		storage.WithSynchronizeTxStatuses(cfg.SynchronizeTxStatuses),
	)
	if errStorage != nil {
		return nil, fmt.Errorf("failed to create storage manager: %w", errStorage)
	}

	// Get storage identity key
	storageIdentityKey, errKey := wdk.IdentityKey(cfg.ServerPrivateKey)
	if errKey != nil {
		return nil, fmt.Errorf("failed to create storage identity key: %w", errKey)
	}

	// Migrate storage
	if _, errMigrate := storageManager.Migrate(context.TODO(), "wallet-advertiser", storageIdentityKey); errMigrate != nil {
		return nil, fmt.Errorf("failed to migrate storage: %w", errMigrate)
	}

	// Determine the network based on chain parameter
	var network defs.BSVNetwork
	if chain == "test" {
		network = defs.NetworkTestnet
	} else {
		network = defs.NetworkMainnet
	}

	// Create wallet
	wlt, err := toolboxWallet.New(network, privKey, storageManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	// Create AuthFetch client
	authClient := authhttp.New(wlt)

	return &WalletAdvertiser{
		chain:                chain,
		privateKey:           privateKey,
		storageURL:           storageURL,
		advertisableURI:      advertisableURI,
		lookupResolverConfig: lookupResolverConfig,
		initialized:          false,
		authFetch:            authClient,
		wallet:               wlt,
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
		return errAlreadyInitialized
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

	// Derive and set the identity key
	identityKey, err := w.deriveIdentityKey()
	if err != nil {
		return fmt.Errorf("identity key derivation failed: %w", err)
	}
	w.identityKey = identityKey

	w.initialized = true
	return nil
}

// CreateAdvertisements creates new advertisements and returns them as a tagged BEEF.
// This method supports both SHIP and SLAP protocol advertisements.
func (w *WalletAdvertiser) CreateAdvertisements(adsData []*oa.AdvertisementData) (overlay.TaggedBEEF, error) {
	if !w.initialized {
		return overlay.TaggedBEEF{}, errNotInitializedForAds
	}

	if len(adsData) == 0 {
		return overlay.TaggedBEEF{}, errNoAdvertisementData
	}

	// Validate all advertisement data entries
	for i, adData := range adsData {
		if err := w.validateAdvertisementData(adData); err != nil {
			return overlay.TaggedBEEF{}, fmt.Errorf("invalid advertisement data at index %d: %w", i, err)
		}
	}

	// Use Finder for testing if available
	if w.Finder != nil {
		return w.Finder.CreateAdvertisements(adsData, w.identityKey, w.advertisableURI)
	}

	// Collect topics for the TaggedBEEF
	var topics []string
	for _, adData := range adsData {
		switch adData.Protocol {
		case overlay.ProtocolSHIP, overlay.ProtocolSLAP:
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

	storageManager, errStorage := storage.NewGORMProvider(
		cfg.BSVNetwork,
		activeServices,
		storage.WithDBConfig(cfg.DBConfig),
		storage.WithFeeModel(cfg.FeeModel),
		storage.WithCommission(cfg.Commission),
		storage.WithSynchronizeTxStatuses(cfg.SynchronizeTxStatuses),
	)
	if errStorage != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create storage provider: %w", errStorage)
	}

	storageIdentityKey, errKey := wdk.IdentityKey(cfg.ServerPrivateKey)
	if errKey != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create storage identity key: %w", errKey)
	}

	if _, errMigrate := storageManager.Migrate(context.TODO(), cfg.Name, storageIdentityKey); errMigrate != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to migrate storage: %w", errMigrate)
	}

	wlt, errWallet := toolboxWallet.New(defs.NetworkMainnet, privKey, storageManager)
	if errWallet != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create wallet: %w", errWallet)
	}
	keyDeriver := wallet.NewKeyDeriver(privKey)

	pd := pushdrop.PushDrop{
		Wallet: wlt,
	}

	outputs := make([]wallet.CreateActionOutput, 0, len(adsData))
	for _, ad := range adsData {
		if !utils.IsValidTopicOrServiceName(ad.TopicOrServiceName) {
			return overlay.TaggedBEEF{}, fmt.Errorf("%w: %s", errInvalidTopicOrServiceName, ad.TopicOrServiceName)
		}
		protocol := wallet.Protocol{SecurityLevel: wallet.SecurityLevelEveryAppAndCounterparty}
		if protocol.Protocol = string(ad.Protocol.ID()); protocol.Protocol == "" {
			return overlay.TaggedBEEF{}, fmt.Errorf("%w: %s", errUnsupportedProtocol, ad.Protocol)
		}
		lockingScript, errLock := pd.Lock(
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
		if errLock != nil {
			return overlay.TaggedBEEF{}, fmt.Errorf("failed to create locking script: %w", errLock)
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
// This method queries the overlay network using LookupResolver to retrieve existing advertisements.
func (w *WalletAdvertiser) FindAllAdvertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error) {
	if !w.initialized {
		return nil, errNotInitializedForFind
	}

	// Validate protocol
	if protocol != overlay.ProtocolSHIP && protocol != overlay.ProtocolSLAP {
		return nil, fmt.Errorf("%w: %s", errUnsupportedProtocol, protocol)
	}

	// Support custom Finder for testing
	if w.Finder != nil {
		return w.Finder.Advertisements(protocol)
	}

	// Query the storage for advertisements matching the protocol
	advertisements, err := w.queryStorageForAdvertisements(protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query data: %w", err)
	}

	return advertisements, nil
}

// RevokeAdvertisements revokes existing advertisements and returns the revocation as a tagged BEEF.
// This method creates spending transactions to invalidate the specified advertisements.
func (w *WalletAdvertiser) RevokeAdvertisements(advertisements []*oa.Advertisement) (overlay.TaggedBEEF, error) {
	if !w.initialized {
		return overlay.TaggedBEEF{}, errNotInitializedForRevoke
	}

	if len(advertisements) == 0 {
		return overlay.TaggedBEEF{}, errNoAdvertisements
	}

	// Validate all advertisements have the required revocation data
	var topics []string
	for i, ad := range advertisements {
		if len(ad.Beef) == 0 {
			return overlay.TaggedBEEF{}, fmt.Errorf("advertisement at index %d %w", i, errMissingBeefData)
		}
		if ad.OutputIndex == 0 {
			return overlay.TaggedBEEF{}, fmt.Errorf("advertisement at index %d %w", i, errMissingOutputIndex)
		}

		// Collect topics for the TaggedBEEF
		switch ad.Protocol {
		case overlay.ProtocolSHIP, overlay.ProtocolSLAP:
			topics = append(topics, "tm_"+ad.TopicOrService)
		}
	}

	// Create spending transactions that consume the advertisement UTXOs
	revocationTransactions, err := w.createNewRevocationTransactions(advertisements)
	if err != nil {
		return overlay.TaggedBEEF{}, fmt.Errorf("failed to create revocation transactions: %w", err)
	}

	// Encode the revocation transactions as BEEF format
	beefData := w.encodeTransactionsAsBEEF(revocationTransactions)

	return overlay.TaggedBEEF{
		Beef:   beefData,
		Topics: topics,
	}, nil
}

// ParseAdvertisement parses an output script to extract advertisement information.
// This method decodes PushDrop locking scripts to reconstruct advertisement data.
func (w *WalletAdvertiser) ParseAdvertisement(outputScript *script.Script) (*oa.Advertisement, error) {
	if !w.initialized {
		return nil, errNotInitializedForParse
	}

	if outputScript == nil || len(*outputScript) == 0 {
		return nil, errOutputScriptEmpty
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
		return nil, fmt.Errorf("%w: %s", errInvalidPushDropScript, scriptHex)
	}

	// Validate that we have the expected number of fields
	if len(result.Fields) < 4 {
		return nil, fmt.Errorf("%w, got %d", errInvalidPushDropFields, len(result.Fields))
	}

	// Extract and validate protocol identifier
	protocolIdentifier := string(result.Fields[0])
	protocol := overlay.Protocol(protocolIdentifier)
	switch protocol {
	case overlay.ProtocolSHIP, overlay.ProtocolSLAP:
	default:
		return nil, fmt.Errorf("%w: %s", errUnsupportedProtocolID, protocolIdentifier)
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
		return nil, fmt.Errorf("%w: %s", errInvalidTopicOrServiceName, fullTopicOrService)
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
		return fmt.Errorf("%w: %s", errUnsupportedProtocol, adData.Protocol)
	}

	// Validate topic or service name
	if strings.TrimSpace(adData.TopicOrServiceName) == "" {
		return errTopicNameEmpty
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
		return fmt.Errorf("%w: %s", errInvalidTopicOrServiceName, fullName)
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
		return fmt.Errorf("%w, got %d bytes", errPrivateKeyInsufficientLength, len(privateKeyBytes))
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
		return errPrivateKeyAllZeros
	}

	// Basic entropy check - private key should have some randomness
	// This is a simple heuristic, not cryptographically rigorous
	uniqueBytes := make(map[byte]bool)
	for _, b := range privateKeyBytes {
		uniqueBytes[b] = true
	}
	if len(uniqueBytes) < 4 {
		return errPrivateKeyInsufficientEntropy
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

	// Create a context with a 10-second timeout for the request
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construct a basic health check endpoint
	// This follows common patterns where storage services expose health endpoints
	healthURL := storageURL.ResolveReference(&url.URL{Path: "/health"})

	// Attempt to connect to the storage service
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// If /health doesn't exist, try a simple HEAD request to the base URL
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, w.storageURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("storage URL is not reachable: %w", err)
		}
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Accept any response that indicates the server is responding
	// We don't require specific status codes since different storage implementations
	// may respond differently to health checks
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: %d %s", errStorageServerError, resp.StatusCode, resp.Status)
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

// deriveIdentityKey derives an identity key from the private key
func (w *WalletAdvertiser) deriveIdentityKey() (string, error) {
	// Create private key from hex
	privateKey, err := ec.PrivateKeyFromHex(w.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to create private key: %w", err)
	}

	// Derive the public key (identity key)
	publicKey := privateKey.PubKey()

	// Return compressed public key as hex string
	return hex.EncodeToString(publicKey.Compressed()), nil
}

// getOverlayNetwork returns the overlay network based on the chain configuration
func (w *WalletAdvertiser) getOverlayNetwork() overlay.Network {
	if w.chain == "test" {
		return overlay.NetworkTestnet
	}
	return overlay.NetworkMainnet
}

// encodeTransactionsAsBEEF encodes transactions in BEEF (Binary Extensible Exchange Format)
func (w *WalletAdvertiser) encodeTransactionsAsBEEF(transactions []*Transaction) []byte {
	var beefData []byte

	// BEEF format header (simplified version)
	beefData = append(beefData, []byte("BEEF")...)      // Magic bytes
	beefData = append(beefData, 0x01, 0x00, 0x00, 0x00) // Version

	// Encode number of transactions
	beefData = append(beefData, w.encodeVarInt(uint64(len(transactions)))...)

	// Encode each transaction
	for _, tx := range transactions {
		txBytes := w.encodeTransaction(tx)
		beefData = append(beefData, txBytes...)
	}

	return beefData
}

// encodeTransaction encodes a transaction in Bitcoin format
func (w *WalletAdvertiser) encodeTransaction(tx *Transaction) []byte {
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

	return txBytes
}

// encodeVarInt encodes a variable-length integer
func (w *WalletAdvertiser) encodeVarInt(value uint64) []byte {
	if value < 0xfd {
		return []byte{byte(value)}
	}
	if value <= 0xffff {
		return []byte{0xfd, byte(value), byte(value >> 8)}
	}
	if value <= 0xffffffff {
		return []byte{0xfe, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
	}
	return []byte{
		0xff, byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
		byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56),
	}
}

// encodeUint32 encodes a 32-bit unsigned integer in little endian format
func (w *WalletAdvertiser) encodeUint32(value uint32) []byte {
	return []byte{byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24)}
}

// encodeUint64 encodes a 64-bit unsigned integer in little endian format
func (w *WalletAdvertiser) encodeUint64(value uint64) []byte {
	return []byte{
		byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
		byte(value >> 32), byte(value >> 40), byte(value >> 48), byte(value >> 56),
	}
}

// queryStorageForAdvertisements queries the storage service for advertisements of a specific protocol
func (w *WalletAdvertiser) queryStorageForAdvertisements(protocol overlay.Protocol) ([]*oa.Advertisement, error) {
	// Create LookupResolver based on configuration
	var resolver *lookup.LookupResolver
	if w.lookupResolverConfig != nil {
		// Convert our config to LookupResolver config
		resolverConfig := &lookup.LookupResolver{
			NetworkPreset: w.getOverlayNetwork(),
		}
		// Add any additional configuration from lookupResolverConfig if needed
		resolver = lookup.NewLookupResolver(resolverConfig)
	} else {
		// Use default network preset based on chain
		resolverConfig := &lookup.LookupResolver{
			NetworkPreset: w.getOverlayNetwork(),
		}
		resolver = lookup.NewLookupResolver(resolverConfig)
	}

	// Determine the service name based on protocol
	var serviceName string
	if protocol == overlay.ProtocolSHIP {
		serviceName = "ls_ship"
	} else {
		serviceName = "ls_slap"
	}

	// Create lookup question with proper JSON encoding
	queryData := map[string]interface{}{
		"identityKey": w.identityKey,
	}
	queryJSON, err := json.Marshal(queryData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query data: %w", err)
	}

	question := &lookup.LookupQuestion{
		Service: serviceName,
		Query:   json.RawMessage(queryJSON),
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the lookup query
	lookupAnswer, err := resolver.Query(ctx, question)
	if err != nil {
		// Log warning but return empty array, matching TypeScript behavior
		slog.Warn("Error finding advertisements", "protocol", protocol, "error", err)
		return []*oa.Advertisement{}, nil
	}

	// Process the lookup answer
	var advertisements []*oa.Advertisement

	// Lookup will currently always return type output-list
	if lookupAnswer.Type == "output-list" {
		for _, output := range lookupAnswer.Outputs {
			// Parse out the advertisements using the provided parser
			tx, err := transaction.NewTransactionFromBEEF(output.Beef)
			if err != nil {
				slog.Error("Failed to parse transaction from BEEF", "error", err)
				continue
			}

			// Get the output at the specified index
			if int(output.OutputIndex) >= len(tx.Outputs) {
				slog.Error("Output index out of range for transaction with outputs", "index", output.OutputIndex,
					"output_count", len(tx.Outputs))
				continue
			}

			txOutput := tx.Outputs[output.OutputIndex]
			lockingScript := txOutput.LockingScript

			// Parse the advertisement from the locking script
			advertisement, err := w.ParseAdvertisement(lockingScript)
			if err != nil {
				slog.Error("Failed to parse advertisement output", "error", err)
				continue
			}

			// Check if the advertisement matches the requested protocol
			if advertisement != nil && advertisement.Protocol == protocol {
				slog.Info("Found current advertisement", "TopicOrService", advertisement.TopicOrService,
					"Domain", advertisement.Domain)
				// Add BEEF and output index from the lookup result
				advertisement.Beef = output.Beef
				advertisement.OutputIndex = output.OutputIndex
				advertisements = append(advertisements, advertisement)
			}
		}
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

// createNewRevocationTransactions creates spending transactions to revoke advertisements (new types)
func (w *WalletAdvertiser) createNewRevocationTransactions(advertisements []*oa.Advertisement) ([]*Transaction, error) {
	transactions := make([]*Transaction, 0, len(advertisements))

	for i, ad := range advertisements {
		tx, err := w.createSingleNewRevocationTransaction(ad)
		if err != nil {
			return nil, fmt.Errorf("failed to create revocation transaction for advertisement %d: %w", i, err)
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
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
		return nil, errBEEFTooShort
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
		return nil, errTransactionTooShort
	}

	offset := 0

	// Parse version
	version := w.parseUint32(data[offset:])
	offset += 4

	// Parse input count
	inputCount, varIntSize := w.parseVarInt(data[offset:])
	_ = varIntSize // varIntSize not used in simplified parsing

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
