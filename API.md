# API Documentation

## Service Interfaces

### SHIP Service

The SHIP service implements the `engine.TopicManager` interface for handling SHIP protocol operations.

#### Methods

##### `OutputAdmittedByTopic`
Processes newly admitted SHIP outputs.

```go
func (s *SHIPTopicManager) OutputAdmittedByTopic(ctx context.Context, args *engine.OutputAdmittedByTopicArgs) error
```

**Parameters:**
- `ctx`: Context for cancellation
- `args`: Contains the admitted output details including topic, BEEF, and tx details

**Returns:**
- Error if processing fails

##### `Lookup`
Performs lookups for SHIP advertisements.

```go
func (s *SHIPLookupService) Lookup(ctx context.Context, question *lookup.LookupQuestion) (*lookup.LookupAnswer, error)
```

**Parameters:**
- `ctx`: Context for cancellation
- `question`: Contains service name and query

**Query Format:**
- `"findAll"`: Returns all SHIP records
- JSON object with optional fields:
  - `identityKey`: Filter by identity key
  - `domain`: Filter by domain
  - `topics`: Array of topics to filter by

**Returns:**
- `LookupAnswer` with formula type containing matching outpoints
- Error if lookup fails

### SLAP Service

The SLAP service has an identical interface to SHIP but handles SLAP protocol advertisements.

## Storage Interfaces

### SHIP Storage

The SHIP storage interface is located in `pkg/ship/storage_interface.go`:

```go
type Storage interface {
    EnsureIndexes(ctx context.Context) error
    StoreSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey, domain, topic string) error
    DeleteSHIPRecord(ctx context.Context, outpoint *transaction.Outpoint) error
    FindRecord(ctx context.Context, query *types.SHIPQuery) ([]*transaction.Outpoint, error)
    FindAll(ctx context.Context) ([]*transaction.Outpoint, error)
}
```

### SLAP Storage

The SLAP storage interface is located in `pkg/slap/storage_interface.go`:

```go
type Storage interface {
    EnsureIndexes(ctx context.Context) error
    StoreSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint, identityKey, domain, service string) error
    DeleteSLAPRecord(ctx context.Context, outpoint *transaction.Outpoint) error
    FindRecord(ctx context.Context, query *types.SLAPQuery) ([]*transaction.Outpoint, error)
    FindAll(ctx context.Context) ([]*transaction.Outpoint, error)
}
```

## Advertiser Interface

```go
type Advertiser interface {
    Init(ctx context.Context) error
    CreateAdvertisements(ctx context.Context, adsData []*types.AdvertisementData) (*overlay.TaggedBEEF, error)
    FindAllAdvertisements(ctx context.Context, protocol overlay.Protocol) ([]*types.Advertisement, error)
    RevokeAdvertisements(ctx context.Context, advertisements []*types.Advertisement) (*overlay.TaggedBEEF, error)
    ParseAdvertisement(outputScript *script.Script) (*types.Advertisement, error)
}
```

### Methods

#### `Init`
Initializes the advertiser (required before other operations).

**Parameters:**
- `ctx`: Context for cancellation

**Returns:**
- Error if initialization fails

#### `CreateAdvertisements`
Creates one or more advertisements in a single transaction.

**Parameters:**
- `ctx`: Context for cancellation
- `adsData`: Array of advertisement data specifying protocol and topic/service names

**Returns:**
- `TaggedBEEF`: Transaction in BEEF format with relevant topics
- Error if creation fails

#### `FindAllAdvertisements`
Finds all advertisements created by this identity for a given protocol.

**Parameters:**
- `ctx`: Context for cancellation
- `protocol`: Either `overlay.ProtocolSHIP` or `overlay.ProtocolSLAP`

**Returns:**
- Array of found advertisements
- Error if search fails

#### `RevokeAdvertisements`
Revokes existing advertisements by spending them.

**Parameters:**
- `ctx`: Context for cancellation
- `advertisements`: Array of advertisements to revoke (must contain Beef and OutputIndex)

**Returns:**
- `TaggedBEEF`: Revocation transaction
- Error if revocation fails

#### `ParseAdvertisement`
Parses an advertisement from the provided output script.

**Parameters:**
- `outputScript`: The output script to parse

**Returns:**
- `Advertisement`: Parsed advertisement data
- Error if parsing fails

## Package Structure

The codebase is organized following the TypeScript structure:

- `pkg/ship/` - SHIP protocol implementation
  - `ship_lookup_service.go` - Lookup service implementation
  - `ship_topic_manager.go` - Topic manager for output admittance
  - `ship_storage.go` - MongoDB storage implementation
  - `storage_interface.go` - Storage interface definition
- `pkg/slap/` - SLAP protocol implementation
  - `slap_lookup_service.go` - Lookup service implementation
  - `slap_topic_manager.go` - Topic manager for output admittance
  - `slap_storage.go` - MongoDB storage implementation
  - `storage_interface.go` - Storage interface definition
- `pkg/advertiser/` - Advertisement creation and management
  - `wallet_advertiser.go` - Main advertiser implementation
- `pkg/types/` - Shared type definitions
  - `types.go` - Advertisement and query types
  - `admission_types.go` - Admission and notification mode types
- `pkg/utils/` - Utility functions
  - `validation.go` - Topic/service name and URI validation
  - `token_verification.go` - Token signature verification

## Type Definitions

### AdvertisementData
```go
type AdvertisementData struct {
    Protocol           overlay.Protocol
    TopicOrServiceName string
}
```

### Advertisement
```go
type Advertisement struct {
    Protocol       overlay.Protocol
    IdentityKey    string
    Domain         string
    TopicOrService string
    Beef           []byte              // Optional: BEEF containing the advertisement
    OutputIndex    *uint32             // Optional: Output index within the transaction
}
```

### SHIPQuery
```go
type SHIPQuery struct {
    IdentityKey *string
    Domain      *string
    Topics      []string
}
```

### SLAPQuery
```go
type SLAPQuery struct {
    Domain      *string
    Service     *string
    IdentityKey *string
}
```

### SHIPStorageRecord (MongoDB storage)
```go
type SHIPStorageRecord struct {
    Outpoint    string    `bson:"outpoint"`     // Format: "txid.outputIndex"
    IdentityKey string    `bson:"identityKey"`
    Domain      string    `bson:"domain"`
    Topic       string    `bson:"topic"`
    CreatedAt   time.Time `bson:"createdAt"`
}
```

### SLAPStorageRecord (MongoDB storage)
```go
type SLAPStorageRecord struct {
    Outpoint    string    `bson:"outpoint"`     // Format: "txid.outputIndex"
    IdentityKey string    `bson:"identityKey"`
    Domain      string    `bson:"domain"`
    Service     string    `bson:"service"`
    CreatedAt   time.Time `bson:"createdAt"`
}
```

**Note:** The storage uses `transaction.Outpoint` which serializes to the format "txid.outputIndex" for storage.

## Error Handling

All methods return descriptive errors that can be checked for specific conditions:

- Invalid input validation errors
- Storage operation errors
- Protocol parsing errors
- Network communication errors

Example error checking:
```go
answer, err := shipService.Lookup(ctx, question)
if err != nil {
    if strings.Contains(err.Error(), "not supported") {
        // Handle unsupported service
    } else if strings.Contains(err.Error(), "invalid query") {
        // Handle invalid query format
    }
    // Handle other errors
}
```

## Utility Functions

### isValidTopicOrServiceName
```go
func IsValidTopicOrServiceName(name string) bool
```
Validates topic or service names according to the pattern: `^(?=.{1,50}$)(?:tm_|ls_)[a-z]+(?:_[a-z]+)*$`

### isAdvertisableURI
```go
func IsAdvertisableURI(uri string) bool
```
Validates URIs for advertising. Supports:
- `https://` (excluding localhost)
- `https+bsvauth://`
- `https+bsvauth+smf://`
- `https+bsvauth+scrypt-offchain://`
- `https+rtt://`
- `wss://` (excluding localhost)
- `js8c+bsvauth+smf:` (with required lat/long/freq/radius parameters)

### isTokenSignatureCorrectlyLinked
```go
func IsTokenSignatureCorrectlyLinked(lockingPublicKey *ec.PublicKey, fields [][]byte) bool
```
Verifies that a token signature is correctly linked to the locking public key.

**Parameters:**
- `lockingPublicKey`: The public key that should match the derived key
- `fields`: PushDrop fields including the signature

**Returns:**
- `true` if the signature is valid and the derived key matches the locking key