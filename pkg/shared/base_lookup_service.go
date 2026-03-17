package shared

import (
	"context"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
	"github.com/bsv-blockchain/go-overlay-services/pkg/core/engine"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/overlay/lookup"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// StoreRecordFunc is the function signature for storing a record parsed from PushDrop output.
type StoreRecordFunc func(ctx context.Context, txid string, outputIndex int, identityKey, domain, fourthField string) error

// FindAllFunc is the function signature for retrieving all records with pagination.
type FindAllFunc func(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error)

// BaseLookupConfig holds protocol-specific configuration for a BaseLookupService.
type BaseLookupConfig struct {
	// Topic is the expected topic name (e.g. "tm_ship" or "tm_slap")
	Topic string
	// ServiceID is the lookup service identifier (e.g. "ls_ship" or "ls_slap")
	ServiceID string
	// Identifier is the protocol identifier in PushDrop fields (e.g. "SHIP" or "SLAP")
	Identifier string
	// MetaDataName is the human-readable name for GetMetaData
	MetaDataName string
	// MetaDataDescription is the description for GetMetaData
	MetaDataDescription string
	// Documentation is the documentation string returned by GetDocumentation
	Documentation *string
	// StoreRecord stores a parsed PushDrop record
	StoreRecord StoreRecordFunc
	// DeleteRecord deletes a record by txid and output index
	DeleteRecord DeleteRecordFunc
	// FindAll returns all records with pagination
	FindAll FindAllFunc
}

// BaseLookupService provides shared implementations for the engine.LookupService interface
// methods that are structurally identical between SHIP and SLAP. Embed this in protocol-specific
// LookupService types to eliminate code duplication.
type BaseLookupService struct {
	DiscoveryNoOps

	Cfg BaseLookupConfig
}

// NewBaseLookupService creates a new BaseLookupService with the given configuration.
func NewBaseLookupService(cfg BaseLookupConfig) BaseLookupService {
	return BaseLookupService{Cfg: cfg}
}

// OutputAdmittedByTopic processes a PushDrop output and stores the record if it matches
// the expected topic and protocol identifier.
func (b *BaseLookupService) OutputAdmittedByTopic(ctx context.Context, payload *engine.OutputAdmittedByTopic) error {
	fields, err := ParsePushDropOutput(payload, b.Cfg.Topic, b.Cfg.Identifier)
	if err != nil {
		return err
	}
	if fields == nil {
		return nil
	}

	return b.Cfg.StoreRecord(ctx, fields.Txid, fields.OutputIndex, fields.IdentityKey, fields.Domain, fields.FourthField)
}

// OutputSpent removes the record when the UTXO is spent.
func (b *BaseLookupService) OutputSpent(ctx context.Context, payload *engine.OutputSpent) error {
	return HandleOutputSpent(ctx, payload, b.Cfg.Topic, b.Cfg.DeleteRecord)
}

// OutputEvicted removes the record when the UTXO is evicted from the mempool.
func (b *BaseLookupService) OutputEvicted(ctx context.Context, outpoint *transaction.Outpoint) error {
	return HandleOutputEvicted(ctx, outpoint, b.Cfg.DeleteRecord)
}

// ServiceName returns the protocol service identifier.
func (b *BaseLookupService) ServiceName() string {
	return b.Cfg.ServiceID
}

// FindAll delegates to the configured FindAll function (implements shared.QueryExecutor).
func (b *BaseLookupService) FindAll(ctx context.Context, limit, skip *int, sortOrder *types.SortOrder) ([]types.UTXOReference, error) {
	return b.Cfg.FindAll(ctx, limit, skip, sortOrder)
}

// GetDocumentation returns the protocol documentation string.
func (b *BaseLookupService) GetDocumentation() string {
	if b.Cfg.Documentation != nil {
		return *b.Cfg.Documentation
	}
	return ""
}

// GetMetaData returns the protocol metadata.
func (b *BaseLookupService) GetMetaData() *overlay.MetaData {
	return &overlay.MetaData{
		Name:        b.Cfg.MetaDataName,
		Description: b.Cfg.MetaDataDescription,
	}
}

// Lookup performs a lookup query using the shared ExecuteLookup framework.
// The executor parameter should be the outer LookupService that implements QueryExecutor.
func (b *BaseLookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion, executor QueryExecutor) (*lookup.LookupAnswer, error) {
	return ExecuteLookup(ctx, question, executor)
}
