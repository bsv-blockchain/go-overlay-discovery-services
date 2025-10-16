// Package types defines the core data structures and interfaces for the BSV Overlay Discovery Services.
// enabling interaction with SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) systems.
//
//nolint:revive // Package name is descriptive in context
package types

import (
	"time"
)

// UTXOReference represents a reference to a specific UTXO (Unspent Transaction Output).
// It contains the transaction ID and the output index within that transaction.
type UTXOReference struct {
	// Txid is the transaction ID in hexadecimal format
	Txid string `json:"txid" bson:"txid"`
	// OutputIndex is the index of the output within the transaction
	OutputIndex int `json:"outputIndex" bson:"outputIndex"`
}

// SHIPRecord represents a SHIP (Service Host Interconnect Protocol) record.
// SHIP records are used to advertise services and their availability on specific domains and topics.
type SHIPRecord struct {
	// Txid is the transaction ID where this record is stored
	Txid string `json:"txid" bson:"txid"`
	// OutputIndex is the index of the output within the transaction
	OutputIndex int `json:"outputIndex" bson:"outputIndex"`
	// IdentityKey is the public key that identifies the service provider
	IdentityKey string `json:"identityKey" bson:"identityKey"`
	// Domain is the domain where the service is hosted
	Domain string `json:"domain" bson:"domain"`
	// Topic is the specific topic or service type being advertised
	Topic string `json:"topic" bson:"topic"`
	// CreatedAt is the timestamp when the record was created
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

// SLAPRecord represents a SLAP (Service Lookup Availability Protocol) record.
// SLAP records are used to advertise service availability and lookup capabilities.
type SLAPRecord struct {
	// Txid is the transaction ID where this record is stored
	Txid string `json:"txid" bson:"txid"`
	// OutputIndex is the index of the output within the transaction
	OutputIndex int `json:"outputIndex" bson:"outputIndex"`
	// IdentityKey is the public key that identifies the service provider
	IdentityKey string `json:"identityKey" bson:"identityKey"`
	// Domain is the domain where the service is hosted
	Domain string `json:"domain" bson:"domain"`
	// Service is the specific service being advertised
	Service string `json:"service" bson:"service"`
	// CreatedAt is the timestamp when the record was created
	CreatedAt time.Time `json:"createdAt" bson:"createdAt"`
}

// SortOrder represents the sort order for query results
type SortOrder string

const (
	// SortOrderAsc represents ascending sort order
	SortOrderAsc SortOrder = "asc"
	// SortOrderDesc represents descending sort order
	SortOrderDesc SortOrder = "desc"
)

// SHIPQuery represents query parameters for searching SHIP records.
// All fields are optional and can be used to filter and paginate results.
type SHIPQuery struct {
	// FindAll indicates whether to return all records (ignores other filters when true)
	FindAll *bool `json:"findAll,omitempty" bson:"findAll,omitempty"`
	// Domain filters records by domain
	Domain *string `json:"domain,omitempty" bson:"domain,omitempty"`
	// Topics filters records by topic names
	Topics []string `json:"topics,omitempty" bson:"topics,omitempty"`
	// IdentityKey filters records by identity key
	IdentityKey *string `json:"identityKey,omitempty" bson:"identityKey,omitempty"`
	// Limit specifies the maximum number of records to return
	Limit *int `json:"limit,omitempty" bson:"limit,omitempty"`
	// Skip specifies the number of records to skip (for pagination)
	Skip *int `json:"skip,omitempty" bson:"skip,omitempty"`
	// SortOrder specifies the sort order for results
	SortOrder *SortOrder `json:"sortOrder,omitempty" bson:"sortOrder,omitempty"`
}

// SLAPQuery represents query parameters for searching SLAP records.
// All fields are optional and can be used to filter and paginate results.
type SLAPQuery struct {
	// FindAll indicates whether to return all records (ignores other filters when true)
	FindAll *bool `json:"findAll,omitempty" bson:"findAll,omitempty"`
	// Domain filters records by domain
	Domain *string `json:"domain,omitempty" bson:"domain,omitempty"`
	// Service filters records by service name
	Service *string `json:"service,omitempty" bson:"service,omitempty"`
	// IdentityKey filters records by identity key
	IdentityKey *string `json:"identityKey,omitempty" bson:"identityKey,omitempty"`
	// Limit specifies the maximum number of records to return
	Limit *int `json:"limit,omitempty" bson:"limit,omitempty"`
	// Skip specifies the number of records to skip (for pagination)
	Skip *int `json:"skip,omitempty" bson:"skip,omitempty"`
	// SortOrder specifies the sort order for results
	SortOrder *SortOrder `json:"sortOrder,omitempty" bson:"sortOrder,omitempty"`
}

// Script represents a locking script that can be decoded
type Script []byte

// PushDropResult represents the result of decoding a PushDrop locking script
type PushDropResult struct {
	// Fields contains the decoded fields from the PushDrop script
	Fields [][]byte `json:"fields"`
}

// PushDropDecoder interface defines the methods for decoding PushDrop locking scripts
type PushDropDecoder interface {
	// Decode decodes a PushDrop locking script and returns the fields
	Decode(lockingScript string) (*PushDropResult, error)
}

// LookupResolverConfig represents configuration for lookup resolver functionality
type LookupResolverConfig struct {
	// HTTPSEndpoint is the HTTPS endpoint for lookup resolution
	HTTPSEndpoint *string `json:"httpsEndpoint,omitempty"`
	// MaxRetries is the maximum number of retries for failed lookups
	MaxRetries *int `json:"maxRetries,omitempty"`
	// TimeoutMS is the timeout in milliseconds for lookup operations
	TimeoutMS *int `json:"timeoutMS,omitempty"`
}
