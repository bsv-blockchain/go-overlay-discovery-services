// Package types defines the core data structures and interfaces for the BSV Overlay Discovery Services.
// This package provides Go equivalents for the TypeScript interfaces used in the overlay-discovery-services library,
// enabling interaction with SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) systems.
package types

import (
	"context"
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

// Protocol represents the advertisement protocol type
type Protocol string

const (
	// ProtocolSHIP represents the SHIP protocol
	ProtocolSHIP Protocol = "SHIP"
	// ProtocolSLAP represents the SLAP protocol
	ProtocolSLAP Protocol = "SLAP"
)

// Advertisement represents a unified advertisement structure that can be either SHIP or SLAP.
// This corresponds to the Advertisement interface from the TypeScript implementation.
type Advertisement struct {
	// Protocol specifies whether this is a SHIP or SLAP advertisement
	Protocol Protocol `json:"protocol" bson:"protocol"`
	// IdentityKey is the public key that identifies the advertiser
	IdentityKey string `json:"identityKey" bson:"identityKey"`
	// Domain is the domain where the service is hosted
	Domain string `json:"domain" bson:"domain"`
	// TopicOrService is the topic (for SHIP) or service name (for SLAP)
	TopicOrService string `json:"topicOrService" bson:"topicOrService"`
	// Beef is the Binary Extensible Exchange Format data (optional, used for revocation)
	Beef []byte `json:"beef,omitempty" bson:"beef,omitempty"`
	// OutputIndex is the index of the output within the transaction (optional, used for revocation)
	OutputIndex *int `json:"outputIndex,omitempty" bson:"outputIndex,omitempty"`
}

// AdvertisementData represents the data needed to create a new advertisement.
// This corresponds to the AdvertisementData interface from the TypeScript implementation.
type AdvertisementData struct {
	// Protocol specifies whether this is a SHIP or SLAP advertisement
	Protocol Protocol `json:"protocol" bson:"protocol"`
	// TopicOrServiceName is the topic (for SHIP) or service name (for SLAP) to advertise
	TopicOrServiceName string `json:"topicOrServiceName" bson:"topicOrServiceName"`
}

// TaggedBEEF represents a Tagged Binary Extensible Exchange Format structure.
// This is used for transaction data that includes metadata tags.
type TaggedBEEF struct {
	// BEEF is the Binary Extensible Exchange Format data
	BEEF []byte `json:"beef" bson:"beef"`
	// Topics are the metadata topics associated with this BEEF
	Topics []string `json:"topics,omitempty" bson:"topics,omitempty"`
}

// BSV Overlay Interface Types
// These types define the interfaces used by BSV overlay services for lookup operations.

// AdmissionMode represents the mode for output admission
type AdmissionMode string

const (
	// AdmissionModeLockingScript indicates admission by locking script
	AdmissionModeLockingScript AdmissionMode = "locking-script"
)

// SpendNotificationMode represents the mode for spend notifications
type SpendNotificationMode string

const (
	// SpendNotificationModeNone indicates no spend notifications
	SpendNotificationModeNone SpendNotificationMode = "none"
)

// OutputAdmittedByTopic represents the payload for an output admitted by topic
type OutputAdmittedByTopic struct {
	// Mode is the admission mode (should be "locking-script" for SHIP)
	Mode AdmissionMode `json:"mode"`
	// Topic is the topic that admitted this output (should be "tm_ship" for SHIP)
	Topic string `json:"topic"`
	// LockingScript is the locking script of the output in hex format
	LockingScript string `json:"lockingScript"`
	// Txid is the transaction ID of the output
	Txid string `json:"txid"`
	// OutputIndex is the index of the output within the transaction
	OutputIndex int `json:"outputIndex"`
}

// OutputSpent represents the payload for a spent output
type OutputSpent struct {
	// Mode is the spend notification mode (should be "none" for SHIP)
	Mode SpendNotificationMode `json:"mode"`
	// Topic is the topic associated with this output
	Topic string `json:"topic"`
	// Txid is the transaction ID of the spent output
	Txid string `json:"txid"`
	// OutputIndex is the index of the spent output within the transaction
	OutputIndex int `json:"outputIndex"`
}

// LookupQuestion represents a lookup query request
type LookupQuestion struct {
	// Service is the lookup service identifier (e.g., "ls_ship")
	Service string `json:"service"`
	// Query can be a string (e.g., "findAll") or an object with query parameters
	Query interface{} `json:"query"`
}

// LookupFormula represents the result of a lookup operation
// It's an alias for a slice of UTXO references
type LookupFormula []UTXOReference

// MetaData represents metadata information for a lookup service
type MetaData struct {
	// Name is the service name
	Name string `json:"name"`
	// ShortDescription is a brief description of the service
	ShortDescription string `json:"shortDescription"`
	// IconURL is an optional URL to the service icon
	IconURL *string `json:"iconURL,omitempty"`
	// Version is an optional version string
	Version *string `json:"version,omitempty"`
	// InformationURL is an optional URL for more information
	InformationURL *string `json:"informationURL,omitempty"`
}

// LookupService interface defines the methods that must be implemented by lookup services
type LookupService interface {
	// OutputAdmittedByTopic handles an output being admitted by topic
	OutputAdmittedByTopic(ctx context.Context, payload OutputAdmittedByTopic) error
	// OutputSpent handles an output being spent
	OutputSpent(ctx context.Context, payload OutputSpent) error
	// OutputEvicted handles an output being evicted
	OutputEvicted(ctx context.Context, txid string, outputIndex int) error
	// Lookup performs a lookup query and returns matching results
	Lookup(ctx context.Context, question LookupQuestion) (LookupFormula, error)
	// GetDocumentation returns the service documentation
	GetDocumentation() (string, error)
	// GetMetaData returns the service metadata
	GetMetaData() (MetaData, error)
}

// PushDrop Types and Interfaces
// These types define the interface for PushDrop decoding functionality.

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

// Utils interface defines utility methods for working with decoded data
type Utils interface {
	// ToUTF8 converts a byte array to a UTF-8 string
	ToUTF8(data []byte) string
	// ToHex converts a byte array to a hexadecimal string
	ToHex(data []byte) string
}

// Advertiser interface defines the methods that must be implemented by advertisement services.
// This interface provides functionality for creating, finding, and revoking overlay advertisements.
type Advertiser interface {
	// Init initializes the advertiser service and sets up any required resources
	Init() error
	// CreateAdvertisements creates new advertisements and returns them as a tagged BEEF
	CreateAdvertisements(adsData []AdvertisementData) (TaggedBEEF, error)
	// FindAllAdvertisements finds all advertisements for a given protocol
	FindAllAdvertisements(protocol string) ([]Advertisement, error)
	// RevokeAdvertisements revokes existing advertisements and returns the revocation as a tagged BEEF
	RevokeAdvertisements(advertisements []Advertisement) (TaggedBEEF, error)
	// ParseAdvertisement parses an output script to extract advertisement information
	ParseAdvertisement(outputScript Script) (Advertisement, error)
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
