package slap

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

// SLAPLookupService provides a concrete implementation of the SLAP lookup service
type SLAPLookupService struct {
	storage               Storage
	AdmissionMode         types.AdmissionMode
	SpendNotificationMode types.SpendNotificationMode
}

// NewSLAPLookupService creates a new SLAP lookup service
func NewSLAPLookupService(storage Storage) *SLAPLookupService {
	return &SLAPLookupService{
		storage:               storage,
		AdmissionMode:         types.AdmissionModeLockingScript,
		SpendNotificationMode: types.SpendNotificationModeNone,
	}
}

// OutputAdmittedByTopic handles outputs admitted by topic
func (s *SLAPLookupService) OutputAdmittedByTopic(ctx context.Context, payload *engine.OutputAdmittedByTopic) error {
	// Validate admission mode matches what we expect
	if s.AdmissionMode != types.AdmissionModeLockingScript {
		return errors.New("invalid admission mode")
	}
	
	if payload.Topic != "tm_slap" {
		return nil
	}

	// Decode the PushDrop data from the locking script
	result := pushdrop.Decode(payload.LockingScript)
	if result == nil {
		return errors.New("failed to decode pushdrop data")
	}

	if len(result.Fields) < 4 {
		return errors.New("invalid SLAP advertisement: insufficient fields")
	}

	protocol := string(result.Fields[0])
	if protocol != "SLAP" {
		return nil
	}

	// Identity key needs to be hex encoded
	identityKey := hex.EncodeToString(result.Fields[1])
	domain := string(result.Fields[2])
	service := string(result.Fields[3])

	return s.storage.StoreSLAPRecord(ctx, payload.Outpoint, identityKey, domain, service)
}

// OutputSpent handles outputs that have been spent
func (s *SLAPLookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	// Validate spend notification mode
	if s.SpendNotificationMode != types.SpendNotificationModeNone {
		return errors.New("invalid spend notification mode")
	}
	
	if payload.Topic != "tm_slap" {
		return nil
	}
	
	return s.storage.DeleteSLAPRecord(ctx, payload.Outpoint)
}

// OutputNoLongerRetainedInHistory handles outputs no longer retained in history
func (s *SLAPLookupService) OutputNoLongerRetainedInHistory(ctx context.Context, outpoint *transaction.Outpoint, topic string) error {
	// For SLAP, we don't need to do anything special for historical retention
	return nil
}

// OutputEvicted handles outputs that have been evicted
func (s *SLAPLookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	return s.storage.DeleteSLAPRecord(ctx, outpoint)
}

// OutputBlockHeightUpdated handles block height updates
func (s *SLAPLookupService) OutputBlockHeightUpdated(ctx context.Context, txid *chainhash.Hash, blockHeight uint32, blockIndex uint64) error {
	// For SLAP, we don't need to track block heights
	return nil
}

// Lookup performs a SLAP lookup query
func (s *SLAPLookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error) {
	if question.Query == nil {
		return nil, errors.New("a valid query must be provided")
	}
	
	if question.Service != "ls_slap" {
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

	// Parse the SLAP query
	var slapQuery types.SLAPQuery
	if err := json.Unmarshal(question.Query, &slapQuery); err != nil {
		return nil, err
	}

	// Validate query parameters - empty strings are not valid
	if slapQuery.Domain != nil && *slapQuery.Domain == "" {
		return nil, errors.New("query.domain must be a non-empty string if provided")
	}
	if slapQuery.Service != nil && *slapQuery.Service == "" {
		return nil, errors.New("query.service must be a non-empty string if provided")
	}
	if slapQuery.IdentityKey != nil && *slapQuery.IdentityKey == "" {
		return nil, errors.New("query.identityKey must be a non-empty string if provided")
	}

	outpoints, err := s.storage.FindRecord(ctx, &slapQuery)
	if err != nil {
		return nil, err
	}

	return s.buildFormulaAnswer(outpoints), nil
}

// GetDocumentation returns documentation for the SLAP lookup service
func (s *SLAPLookupService) GetDocumentation() string {
	return `SLAP Lookup Service

The SLAP lookup service allows querying for service availability within the overlay network.`
}

// GetMetaData returns metadata for the SLAP lookup service
func (s *SLAPLookupService) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        "SLAP Lookup Service",
		Description: "Provides lookup capabilities for SLAP tokens.",
	}
}

// buildFormulaAnswer builds a formula answer from outpoints
func (s *SLAPLookupService) buildFormulaAnswer(outpoints []*transaction.Outpoint) *lookup.LookupAnswer {
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

// Verify that SLAPLookupService implements engine.LookupService
var _ engine.LookupService = (*SLAPLookupService)(nil)
