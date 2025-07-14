package advertiser

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	ec "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"
	sdkWallet "github.com/bsv-blockchain/go-sdk/wallet"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/defs"
	"github.com/bsv-blockchain/go-wallet-toolbox/pkg/storage"
	toolboxWallet "github.com/bsv-blockchain/go-wallet-toolbox/pkg/wallet"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"
)

const AD_TOKEN_VALUE = 1

// WalletAdvertiser implements the Advertiser interface for managing SHIP and SLAP advertisements using a Wallet.
type WalletAdvertiser struct {
	chain               string
	privateKey          *ec.PrivateKey
	storageURL          string
	advertisableURI     string
	lookupResolverConfig *lookup.LookupResolver // Optional config for lookup resolver
	wallet              *toolboxWallet.Wallet  // Will be set during Init()
	identityKey         string
	initialized         bool
	logger              *slog.Logger
	storageCleanup      func()                 // Cleanup function for storage client
}

// NewWalletAdvertiser constructs a new WalletAdvertiser instance.
func NewWalletAdvertiser(chain, privateKeyHex, storageURL, advertisableURI string, lookupResolverConfig *lookup.LookupResolver, logger *slog.Logger) (*WalletAdvertiser, error) {
	if !utils.IsAdvertisableURI(advertisableURI) {
		return nil, fmt.Errorf("refusing to initialize with non-advertisable URI: %s", advertisableURI)
	}

	if logger == nil {
		logger = slog.Default()
	}

	privKey, err := ec.PrivateKeyFromHex(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Get identity key (public key hex)
	identityPubKey := privKey.PubKey()
	identityKeyHex := hex.EncodeToString(identityPubKey.Compressed())

	return &WalletAdvertiser{
		chain:                chain,
		privateKey:           privKey,
		storageURL:           storageURL,
		advertisableURI:      advertisableURI,
		lookupResolverConfig: lookupResolverConfig,
		identityKey:          identityKeyHex,
		initialized:          false,
		logger:               logger,
	}, nil
}

// Init initializes the wallet with remote storage, matching TypeScript pattern
func (wa *WalletAdvertiser) Init(ctx context.Context) error {
	// Convert chain string to BSVNetwork
	var network defs.BSVNetwork
	if wa.chain == "main" {
		network = defs.NetworkMainnet
	} else {
		network = defs.NetworkTestnet
	}

	// Create storage client from go-wallet-toolbox
	client, cleanup, err := storage.NewClient(wa.storageURL)
	if err != nil {
		return fmt.Errorf("failed to create storage client: %w", err)
	}
	wa.storageCleanup = cleanup
	
	// Create wallet using go-wallet-toolbox
	wallet, err := toolboxWallet.New(network, wa.privateKey, client)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}

	wa.wallet = wallet
	wa.initialized = true
	return nil
}

// CreateAdvertisements creates multiple advertisements in a single transaction.
func (wa *WalletAdvertiser) CreateAdvertisements(ctx context.Context, adsData []*types.AdvertisementData) (*overlay.TaggedBEEF, error) {
	if !wa.initialized {
		return nil, errors.New("initialize the Advertiser using Init() before use")
	}

	outputs := []sdkWallet.CreateActionOutput{}
	topics := make(map[string]bool)

	for _, ad := range adsData {
		if !utils.IsValidTopicOrServiceName(ad.TopicOrServiceName) {
			return nil, fmt.Errorf("refusing to create %s advertisement with invalid topic or service name: %s", 
				ad.Protocol, ad.TopicOrServiceName)
		}

		// Create PushDrop fields matching TypeScript
		fields := [][]byte{
			[]byte(string(ad.Protocol)),
			mustDecodeHex(wa.identityKey),
			[]byte(wa.advertisableURI),
			[]byte(ad.TopicOrServiceName),
		}

		// Create PushDrop locking script
		// Protocol: "2", KeyID: "1", UserID: "anyone"
		protocolName := "service host interconnect"
		if ad.Protocol == overlay.ProtocolSLAP {
			protocolName = "service lookup availability"
		}

		pd := &pushdrop.PushDrop{
			Wallet:     wa.wallet,
			Originator: "",
		}
		lockingScript, err := pd.Lock(
			ctx,
			fields,
			sdkWallet.Protocol{SecurityLevel: 2, Protocol: protocolName},
			"1",
			sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
			false, // forSelf
			true,  // includeSignature
			pushdrop.LockBefore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create locking script: %w", err)
		}

		wa.logger.Info("Creating advertisement", 
			"topic", ad.TopicOrServiceName,
			"uri", wa.advertisableURI)

		outputs = append(outputs, sdkWallet.CreateActionOutput{
			LockingScript:     lockingScript.Bytes(),
			Satoshis:          AD_TOKEN_VALUE,
			OutputDescription: fmt.Sprintf("%s advertisement of %s", ad.Protocol, ad.TopicOrServiceName),
		})

		// Track topics
		if ad.Protocol == overlay.ProtocolSHIP {
			topics["tm_ship"] = true
		} else {
			topics["tm_slap"] = true
		}
	}

	// Create the action
	actionArgs := &sdkWallet.CreateActionArgs{
		Description: "SHIP/SLAP Advertisement Issuance",
		Outputs:     outputs,
	}

	result, err := wa.wallet.CreateAction(ctx, *actionArgs, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create action: %w", err)
	}

	// Convert to BEEF
	tx, err := transaction.NewTransactionFromBEEF(result.Tx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transaction: %w", err)
	}

	beef, err := tx.BEEF()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to BEEF: %w", err)
	}

	// Collect unique topics
	topicList := make([]string, 0, len(topics))
	for topic := range topics {
		topicList = append(topicList, topic)
	}

	return &overlay.TaggedBEEF{
		Beef:   beef,
		Topics: topicList,
	}, nil
}

// FindAllAdvertisements finds all SHIP or SLAP advertisements for a given protocol created by this identity.
func (wa *WalletAdvertiser) FindAllAdvertisements(ctx context.Context, protocol overlay.Protocol) ([]*types.Advertisement, error) {
	if !wa.initialized {
		return nil, errors.New("initialize the Advertiser using Init() before use")
	}

	// Create lookup resolver
	var resolver *lookup.LookupResolver
	if wa.lookupResolverConfig != nil {
		resolver = lookup.NewLookupResolver(wa.lookupResolverConfig)
	} else {
		// Get network from chain
		var networkPreset overlay.Network
		if wa.chain == "main" {
			networkPreset = overlay.NetworkMainnet
		} else {
			networkPreset = overlay.NetworkTestnet
		}
		resolver = lookup.NewLookupResolver(&lookup.LookupResolver{
			NetworkPreset: networkPreset,
		})
	}

	advertisements := []*types.Advertisement{}

	// Determine service based on protocol
	service := "ls_ship"
	if protocol == overlay.ProtocolSLAP {
		service = "ls_slap"
	}

	// Create query - need to marshal to JSON
	queryData, err := json.Marshal(map[string]interface{}{
		"identityKey": wa.identityKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Query for advertisements
	lookupQuestion := &lookup.LookupQuestion{
		Service: service,
		Query:   queryData,
	}

	lookupAnswer, err := resolver.Query(ctx, lookupQuestion)
	if err != nil {
		wa.logger.Warn("Error finding advertisements", 
			"protocol", protocol,
			"error", err)
		return advertisements, nil
	}

	// Process results based on answer type
	if lookupAnswer.Type == lookup.AnswerTypeOutputList {
		for _, output := range lookupAnswer.Outputs {
			// Parse transaction from BEEF
			tx, err := transaction.NewTransactionFromBEEF(output.Beef)
			if err != nil {
				wa.logger.Error("Failed to parse BEEF", "error", err)
				continue
			}

			// Get the output
			if int(output.OutputIndex) >= len(tx.Outputs) {
				wa.logger.Error("Invalid output index", "index", output.OutputIndex)
				continue
			}

			lockingScript := tx.Outputs[output.OutputIndex].LockingScript
			advertisement, err := wa.ParseAdvertisement(lockingScript)
			if err != nil {
				wa.logger.Error("Failed to parse advertisement", "error", err)
				continue
			}

			if advertisement.Protocol == protocol {
				wa.logger.Info("Found advertisement",
					"topic", advertisement.TopicOrService,
					"domain", advertisement.Domain)
				advertisement.Beef = output.Beef
				outputIdx := uint32(output.OutputIndex)
				advertisement.OutputIndex = &outputIdx
				advertisements = append(advertisements, advertisement)
			}
		}
	}

	return advertisements, nil
}

// RevokeAdvertisements revokes existing advertisements.
func (wa *WalletAdvertiser) RevokeAdvertisements(ctx context.Context, advertisements []*types.Advertisement) (*overlay.TaggedBEEF, error) {
	if len(advertisements) == 0 {
		return nil, errors.New("must provide advertisements to revoke")
	}
	if !wa.initialized {
		return nil, errors.New("initialize the Advertiser using Init() before use")
	}

	// Create inputs from advertisements
	inputs := []sdkWallet.CreateActionInput{}
	topics := make(map[string]bool)
	
	// Collect all BEEFs to merge
	inputBeef := transaction.NewBeef()
	
	for _, ad := range advertisements {
		if ad.Beef == nil || ad.OutputIndex == nil {
			return nil, errors.New("advertisement to revoke must contain beef and output index")
		}

		// Parse transaction from BEEF
		tx, err := transaction.NewTransactionFromBEEF(ad.Beef)
		if err != nil {
			return nil, fmt.Errorf("failed to parse advertisement transaction: %w", err)
		}

		// Merge BEEF
		adBeef, err := transaction.NewBeefFromBytes(ad.Beef)
		if err != nil {
			return nil, fmt.Errorf("failed to parse BEEF: %w", err)
		}
		if err := inputBeef.MergeBeef(adBeef); err != nil {
			return nil, fmt.Errorf("failed to merge BEEF: %w", err)
		}

		txid := tx.TxID().String()
		wa.logger.Info("Revoking advertisement",
			"txid", txid,
			"outputIndex", *ad.OutputIndex,
			"topic", ad.TopicOrService,
			"domain", ad.Domain)

		// Create input
		outpoint := &transaction.Outpoint{
			Txid:  *tx.TxID(),
			Index: uint32(*ad.OutputIndex),
		}
		inputs = append(inputs, sdkWallet.CreateActionInput{
			Outpoint:          *outpoint,
			InputDescription:  fmt.Sprintf("Revoke a %s advertisement for %s", ad.Protocol, ad.TopicOrService),
			UnlockingScriptLength: 74, // Typical PushDrop signature length
		})

		// Track topics
		if ad.Protocol == overlay.ProtocolSHIP {
			topics["tm_ship"] = true
		} else {
			topics["tm_slap"] = true
		}
	}

	// Create partial transaction
	beefBytes, err := inputBeef.Bytes()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize input BEEF: %w", err)
	}

	partialArgs := &sdkWallet.CreateActionArgs{
		Description: "Revoke SHIP/SLAP advertisements",
		Inputs:      inputs,
		InputBEEF:   beefBytes,
	}

	partialResult, err := wa.wallet.CreateAction(ctx, *partialArgs, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create partial transaction: %w", err)
	}

	// Parse the signable transaction
	signableTx, err := transaction.NewTransactionFromBEEF(partialResult.SignableTransaction.Tx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse signable transaction: %w", err)
	}

	// Sign inputs with PushDrop unlock
	spends := make(map[uint32]sdkWallet.SignActionSpend)
	for i, ad := range advertisements {
		protocolName := "service host interconnect"
		if ad.Protocol == overlay.ProtocolSLAP {
			protocolName = "service lookup availability"
		}

		pd := &pushdrop.PushDrop{
			Wallet:     wa.wallet,
			Originator: "",
		}
		unlocker := pd.Unlock(
			ctx,
			sdkWallet.Protocol{SecurityLevel: 2, Protocol: protocolName},
			"1",
			sdkWallet.Counterparty{Type: sdkWallet.CounterpartyTypeAnyone},
			sdkWallet.SignOutputsAll,
			false, // anyoneCanPay
		)
		
		unlockingScript, err := unlocker.Sign(signableTx, i)
		if err != nil {
			return nil, fmt.Errorf("failed to create unlocking script: %w", err)
		}

		spends[uint32(i)] = sdkWallet.SignActionSpend{
			UnlockingScript: unlockingScript.Bytes(),
		}
	}

	// Sign the action
	signArgs := &sdkWallet.SignActionArgs{
		Spends:    spends,
		Reference: partialResult.SignableTransaction.Reference,
	}

	revokeResult, err := wa.wallet.SignAction(ctx, *signArgs, "")
	if err != nil {
		return nil, fmt.Errorf("failed to sign action: %w", err)
	}

	// Convert to BEEF
	revokeTx, err := transaction.NewTransactionFromBEEF(revokeResult.Tx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse revoke transaction: %w", err)
	}

	revokeBeef, err := revokeTx.BEEF()
	if err != nil {
		return nil, fmt.Errorf("failed to convert to BEEF: %w", err)
	}

	// Collect unique topics
	topicList := make([]string, 0, len(topics))
	for topic := range topics {
		topicList = append(topicList, topic)
	}

	return &overlay.TaggedBEEF{
		Beef:   revokeBeef,
		Topics: topicList,
	}, nil
}

// ParseAdvertisement parses an advertisement from the provided output script.
func (wa *WalletAdvertiser) ParseAdvertisement(outputScript *script.Script) (*types.Advertisement, error) {
	// Check for empty script
	if outputScript == nil || len(outputScript.Bytes()) == 0 {
		return nil, errors.New("empty script")
	}
	
	// Decode the PushDrop script
	pushDropData := pushdrop.Decode(outputScript)
	if pushDropData == nil {
		return nil, fmt.Errorf("failed to decode PushDrop")
	}
	fields := pushDropData.Fields

	if len(fields) < 4 {
		return nil, errors.New("invalid SHIP/SLAP advertisement")
	}

	protocolStr := string(fields[0])
	var protocol overlay.Protocol
	switch protocolStr {
	case "SHIP":
		protocol = overlay.ProtocolSHIP
	case "SLAP":
		protocol = overlay.ProtocolSLAP
	default:
		return nil, errors.New("invalid protocol type")
	}

	identityKey := hex.EncodeToString(fields[1])
	domain := string(fields[2])
	topicOrService := string(fields[3])

	return &types.Advertisement{
		Protocol:       protocol,
		IdentityKey:    identityKey,
		Domain:         domain,
		TopicOrService: topicOrService,
	}, nil
}

// Close cleans up resources used by the advertiser
func (wa *WalletAdvertiser) Close() {
	if wa.storageCleanup != nil {
		wa.storageCleanup()
	}
}

// mustDecodeHex decodes a hex string and panics on error
func mustDecodeHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}