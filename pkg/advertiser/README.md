# WalletAdvertiser Package

The `advertiser` package provides the `WalletAdvertiser` implementation for creating and managing SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) advertisements within the BSV overlay network.

## Overview

The `WalletAdvertiser` implements the `Advertiser` interface and provides functionality to:

- Create new overlay advertisements for SHIP and SLAP protocols
- Parse existing advertisements from blockchain output scripts
- Find all advertisements for a specific protocol
- Revoke existing advertisements
- Validate advertisement data and URIs

## Core Components

### WalletAdvertiser Struct

The main struct that implements advertisement functionality:

```go
type WalletAdvertiser struct {
    chain                string
    privateKey           string
    storageURL           string
    advertisableURI      string
    lookupResolverConfig *types.LookupResolverConfig
    // ... other fields
}
```

### Key Methods

#### Constructor
```go
func NewWalletAdvertiser(chain, privateKey, storageURL, advertisableURI string, lookupResolverConfig *LookupResolverConfig) (*WalletAdvertiser, error)
```

Creates a new WalletAdvertiser with validation of all parameters.

#### Initialization
```go
func (w *WalletAdvertiser) Init() error
```

Initializes the advertiser and validates dependencies are set.

#### Advertisement Creation
```go
func (w *WalletAdvertiser) CreateAdvertisements(adsData []*oa.AdvertisementData) (overlay.TaggedBEEF, error)
```

Creates new advertisements as BSV transactions (requires BSV SDK integration).

#### Advertisement Parsing
```go
func (w *WalletAdvertiser) ParseAdvertisement(outputScript Script) (Advertisement, error)
```

Parses PushDrop output scripts to extract advertisement information.

#### Advertisement Discovery
```go
func (w *WalletAdvertiser) FindAllAdvertisements(protocol string) ([]Advertisement, error)
```

Finds all advertisements for a specific protocol (requires storage integration).

#### Advertisement Revocation
```go
func (w *WalletAdvertiser) RevokeAdvertisements(advertisements []Advertisement) (TaggedBEEF, error)
```

Revokes existing advertisements by spending their UTXOs (requires BSV SDK integration).

## Usage Example

```go
package main

import (
    "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/advertiser"
    oa "github.com/bsv-blockchain/go-overlay-services/pkg/core/advertiser"
    "github.com/bsv-blockchain/go-sdk/overlay"
)

func main() {
    // Create advertiser
    advertiser, err := advertiser.NewWalletAdvertiser(
        "main",                                    // chain
        "your-private-key-hex",                   // private key
        "https://storage.example.com",            // storage URL
        "https://your-service.com/",              // advertisable URI
        nil,                                      // lookup config (optional)
    )
    if err != nil {
        panic(err)
    }

    // Set dependencies (mock implementations shown)
    advertiser.SetPushDropDecoder(&MockPushDropDecoder{})
    advertiser.SetUtils(&MockUtils{})

    // Initialize
    if err := advertiser.Init(); err != nil {
        panic(err)
    }

    // Create advertisements
    adsData := []*oa.AdvertisementData{
        {
            Protocol:           overlay.ProtocolSHIP,
            TopicOrServiceName: "payments",
        },
    }

    taggedBEEF, err := advertiser.CreateAdvertisements(adsData)
    // Handle error (implementation pending BSV SDK integration)
}
```

## Dependencies

The WalletAdvertiser requires the following dependencies to be set before initialization:

### PushDropDecoder
```go
type PushDropDecoder interface {
    Decode(lockingScript string) (*PushDropResult, error)
}
```

Used for decoding PushDrop locking scripts that contain advertisement data.

### Utils
```go
type Utils interface {
    ToUTF8(data []byte) string
    ToHex(data []byte) string
}
```

Provides utility functions for data conversion between binary, hex, and UTF-8 formats.

## Validation

The WalletAdvertiser performs comprehensive validation:

### Constructor Validation
- All required parameters must be non-empty
- Private key must be valid hexadecimal
- Storage URL must be HTTP/HTTPS
- Advertisable URI must comply with BRC-101 specification

### Advertisement Data Validation
- Protocol must be "SHIP" or "SLAP"
- Topic/service names must follow BRC-87 guidelines
- Names must use appropriate prefixes ("tm_" for SHIP, "ls_" for SLAP)

### URI Validation
Uses the `utils.IsAdvertisableURI()` function to validate URIs according to BRC-101:
- HTTPS-based schemes (https://, https+bsvauth://, etc.)
- WebSocket Secure (wss://)
- JS8 Call-based URIs (js8c+bsvauth+smf:)

## Error Handling

The WalletAdvertiser provides detailed error messages for:
- Invalid parameters during construction
- Missing dependencies during initialization
- Invalid advertisement data
- Parsing failures
- Validation errors

All errors are wrapped with context to help with debugging.

## Current Implementation Status

### ✅ Completed
- Complete struct definition and constructor
- Parameter validation and error handling
- Advertisement data validation
- PushDrop parsing logic
- Comprehensive unit tests
- Interface compliance verification

### ⏳ Pending (requires external dependencies)
- BSV SDK integration for transaction creation and signing
- Storage backend integration for advertisement persistence
- Actual BEEF creation and revocation implementation

## Testing

The package includes comprehensive unit tests covering:
- Constructor validation with various invalid inputs
- Initialization requirements and error cases
- Method validation and error handling
- Advertisement parsing with mock data
- Proper interface implementation

Run tests with:
```bash
go test ./pkg/advertiser/ -v
```

## Integration Notes

When integrating with the full BSV ecosystem, you'll need to:

1. **Implement BSV SDK integration** for transaction creation, signing, and broadcasting
2. **Add storage backend** for advertisement persistence and querying
3. **Provide real PushDropDecoder** implementation
4. **Add wallet management** for key handling and UTXO management

The current implementation provides a solid foundation with proper validation, error handling, and interface compliance, ready for these integrations.
