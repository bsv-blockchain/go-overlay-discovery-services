package shared

import (
	"errors"

	"github.com/bsv-blockchain/go-sdk/overlay/lookup"

	"github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Common error variables shared between SHIP and SLAP lookup validation.
var (
	ErrQueryLimitInvalid     = errors.New("query.limit must be a positive number if provided")
	ErrQuerySkipInvalid      = errors.New("query.skip must be a non-negative number if provided")
	ErrQuerySortOrderInvalid = errors.New("query.sortOrder must be 'asc' or 'desc' if provided")
)

// ValidatePagination validates limit, skip, and sortOrder query parameters.
func ValidatePagination(limit, skip *int, sortOrder *types.SortOrder) error {
	if limit != nil {
		if *limit < 0 {
			return ErrQueryLimitInvalid
		}
	}

	if skip != nil {
		if *skip < 0 {
			return ErrQuerySkipInvalid
		}
	}

	if sortOrder != nil {
		if *sortOrder != types.SortOrderAsc && *sortOrder != types.SortOrderDesc {
			return ErrQuerySortOrderInvalid
		}
	}

	return nil
}

// ConvertUTXOsToLookupAnswer converts a slice of UTXO references to a LookupAnswer.
func ConvertUTXOsToLookupAnswer(utxos []types.UTXOReference) *lookup.LookupAnswer {
	return &lookup.LookupAnswer{
		Type:   lookup.AnswerTypeFreeform,
		Result: utxos,
	}
}
