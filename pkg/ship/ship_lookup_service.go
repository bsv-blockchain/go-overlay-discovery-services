package ship

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/transaction"
	"github.com/bsv-blockchain/go-sdk/transaction/template/pushdrop"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// SHIPLookupService provides a concrete implementation of the SHIP lookup service
type SHIPLookupService struct {
	storage               Storage
	AdmissionMode         types.AdmissionMode
	SpendNotificationMode types.SpendNotificationMode
}

// NewSHIPLookupService creates a new SHIP lookup service
func NewSHIPLookupService(storage Storage) *SHIPLookupService {
	return &SHIPLookupService{
		storage:               storage,
		AdmissionMode:         types.AdmissionModeLockingScript,
		SpendNotificationMode: types.SpendNotificationModeNone,
	}
}

// OutputAdmittedByTopic handles outputs admitted by topic
func (s *SHIPLookupService) OutputAdmittedByTopic(ctx context.Context, payload *engine.OutputAdmittedByTopic) error {
	// Validate admission mode matches what we expect
	if s.AdmissionMode != types.AdmissionModeLockingScript {
		return errors.New("invalid admission mode")
	}
	
	if payload.Topic != "tm_ship" {
		return nil
	}

	// Decode the PushDrop data from the locking script
	result := pushdrop.Decode(payload.LockingScript)
	if result == nil {
		return errors.New("failed to decode pushdrop data")
	}

	if len(result.Fields) < 4 {
		return errors.New("invalid SHIP advertisement: insufficient fields")
	}

	shipIdentifier := string(result.Fields[0])
	if shipIdentifier != "SHIP" {
		return nil
	}

	// Identity key needs to be hex encoded
	identityKey := hex.EncodeToString(result.Fields[1])
	domain := string(result.Fields[2])
	topicSupported := string(result.Fields[3])

	return s.storage.StoreSHIPRecord(ctx, payload.Outpoint, identityKey, domain, topicSupported)
}

// OutputSpent handles outputs that have been spent
func (s *SHIPLookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	// Validate spend notification mode
	if s.SpendNotificationMode != types.SpendNotificationModeNone {
		return errors.New("invalid spend notification mode")
	}
	
	if payload.Topic != "tm_ship" {
		return nil
	}
	
	return s.storage.DeleteSHIPRecord(ctx, payload.Outpoint)
}

// OutputNoLongerRetainedInHistory handles outputs no longer retained in history
func (s *SHIPLookupService) OutputNoLongerRetainedInHistory(ctx context.Context, outpoint *transaction.Outpoint, topic string) error {
	// For SHIP, we don't need to do anything special for historical retention
	return nil
}

// OutputEvicted handles outputs that have been evicted
func (s *SHIPLookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	return s.storage.DeleteSHIPRecord(ctx, outpoint)
}

// OutputBlockHeightUpdated handles block height updates
func (s *SHIPLookupService) OutputBlockHeightUpdated(ctx context.Context, txid *chainhash.Hash, blockHeight uint32, blockIndex uint64) error {
	// For SHIP, we don't need to track block heights
	return nil
}

// Lookup performs a SHIP lookup query
func (s *SHIPLookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	if question.Query == nil {
		return nil, errors.New("a valid query must be provided")
	}
	
	if question.Service != "ls_ship" {
		return nil, errors.New("lookup service not supported")
	}

	// Check for findAll query
	if string(question.Query) == `"findAll"` {
		outpoints, err := s.storage.FindAll(ctx)
		if err != nil {
			return nil, err
		}
		return s.buildFormulaAnswer(outpoints), nil
	}

	// Parse the SHIP query
	var shipQuery types.SHIPQuery
	if err := json.Unmarshal(question.Query, &shipQuery); err != nil {
		return nil, err
	}

	// Validate query parameters - empty strings are not valid
	if shipQuery.Domain != nil && *shipQuery.Domain == "" {
		return nil, errors.New("query.domain must be a non-empty string if provided")
	}
	if shipQuery.IdentityKey != nil && *shipQuery.IdentityKey == "" {
		return nil, errors.New("query.identityKey must be a non-empty string if provided")
	}
	// Topics array can be empty, that's valid

	outpoints, err := s.storage.FindRecord(ctx, &shipQuery)
	if err != nil {
		return nil, err
	}

	return s.buildFormulaAnswer(outpoints), nil
}

// GetDocumentation returns documentation for the SHIP lookup service
func (s *SHIPLookupService) GetDocumentation() string {
	return `SHIP Lookup Service

The SHIP lookup service allows querying for overlay services hosting specific topics within the overlay network.`
}

// GetMetaData returns metadata for the SHIP lookup service
func (s *SHIPLookupService) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SHIP Lookup Service",
		Description: "Provides lookup capabilities for SHIP tokens.",
	}
}

// buildFormulaAnswer builds a formula answer from outpoints
func (s *SHIPLookupService) buildFormulaAnswer(outpoints []*transaction.Outpoint) *lookup.LookupAnswer {
	formulas := make([]lookup.LookupFormula, len(outpoints))
	for i, outpoint := range outpoints {
		formulas[i] = lookup.LookupFormula{
			Outpoint: outpoint,
			// No history function needed for basic lookups
		}
	}

	return &lookup.LookupAnswer{
		Type:     lookup.AnswerTypeFormula,
		Formulas: formulas,
	}
}

// Verify that SHIPLookupService implements engine.LookupService
var _ engine.LookupService = (*SHIPLookupService)(nil)
