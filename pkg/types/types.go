package types

import (
	"context"
	"time"

	"github.com/bsv-blockchain/go-sdk/chainhash"
	"github.com/bsv-blockchain/go-sdk/overlay"
	"github.com/bsv-blockchain/go-sdk/script"
	"github.com/bsv-blockchain/go-sdk/transaction"
)

// SHIPRecord represents a SHIP protocol record stored in the database
type SHIPRecord struct {
	Txid        *chainhash.Hash `json:"txid" bson:"txid"`
	OutputIndex uint32          `json:"outputIndex" bson:"outputIndex"`
	IdentityKey string          `json:"identityKey" bson:"identityKey"`
	Domain      string          `json:"domain" bson:"domain"`
	Topic       string          `json:"topic" bson:"topic"`
	CreatedAt   time.Time       `json:"createdAt" bson:"createdAt"`
}

// SLAPRecord represents a SLAP protocol record stored in the database
type SLAPRecord struct {
	Txid        *chainhash.Hash `json:"txid" bson:"txid"`
	OutputIndex uint32          `json:"outputIndex" bson:"outputIndex"`
	IdentityKey string          `json:"identityKey" bson:"identityKey"`
	Domain      string          `json:"domain" bson:"domain"`
	Service     string          `json:"service" bson:"service"`
	CreatedAt   time.Time       `json:"createdAt" bson:"createdAt"`
}

// Outpoint returns a transaction.Outpoint for the SHIPRecord
func (s *SHIPRecord) Outpoint() *transaction.Outpoint {
	return &transaction.Outpoint{
		Txid:  *s.Txid,
		Index: s.OutputIndex,
	}
}

// Outpoint returns a transaction.Outpoint for the SLAPRecord
func (s *SLAPRecord) Outpoint() *transaction.Outpoint {
	return &transaction.Outpoint{
		Txid:  *s.Txid,
		Index: s.OutputIndex,
	}
}

// SHIPQuery represents a query for SHIP records
type SHIPQuery struct {
	Domain      *string  `json:"domain,omitempty"`
	Topics      []string `json:"topics,omitempty"`
	IdentityKey *string  `json:"identityKey,omitempty"`
}

// SLAPQuery represents a query for SLAP records
type SLAPQuery struct {
	Domain      *string `json:"domain,omitempty"`
	Service     *string `json:"service,omitempty"`
	IdentityKey *string `json:"identityKey,omitempty"`
}

// Advertisement represents an overlay advertisement
type Advertisement struct {
	Protocol       overlay.Protocol `json:"protocol"`
	IdentityKey    string           `json:"identityKey"`
	Domain         string           `json:"domain"`
	TopicOrService string           `json:"topicOrService"`
	Beef           []byte           `json:"beef,omitempty"`
	OutputIndex    *uint32          `json:"outputIndex,omitempty"`
}

// AdvertisementData represents data needed to create an advertisement
type AdvertisementData struct {
	Protocol           overlay.Protocol `json:"protocol"`
	TopicOrServiceName string           `json:"topicOrServiceName"`
}

// Advertiser interface defines the methods for managing advertisements
type Advertiser interface {
	// Init initializes the advertiser
	Init(ctx context.Context) error

	// CreateAdvertisements creates multiple advertisements in a single transaction
	CreateAdvertisements(ctx context.Context, adsData []*AdvertisementData) (*overlay.TaggedBEEF, error)

	// FindAllAdvertisements finds all advertisements for a given protocol created by this identity
	FindAllAdvertisements(ctx context.Context, protocol overlay.Protocol) ([]*Advertisement, error)

	// RevokeAdvertisements revokes existing advertisements
	RevokeAdvertisements(ctx context.Context, advertisements []*Advertisement) (*overlay.TaggedBEEF, error)

	// ParseAdvertisement parses an advertisement from the provided output script
	ParseAdvertisement(outputScript *script.Script) (*Advertisement, error)
}
