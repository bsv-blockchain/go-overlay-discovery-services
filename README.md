# Go Overlay Discovery Services

Go implementation of the BSV Overlay Discovery Services, providing SHIP (Service Host Interconnect Protocol) and SLAP (Service Lookup Availability Protocol) functionality.

## Overview

This package implements the discovery services for BSV overlay networks, allowing services to advertise their availability and enabling clients to discover services hosting specific topics.

### Key Components

- **SHIP (Service Host Interconnect Protocol)**: Allows overlay services to advertise which topics they host
- **SLAP (Service Lookup Availability Protocol)**: Enables discovery of services that can perform lookups for specific topics

## Installation

```bash
go get github.com/bsv-blockchain/go-overlay-discovery-services
```

## Project Structure

```
├── pkg/
│   ├── advertiser/         # Advertisement creation and management
│   ├── services/           # Core service implementations
│   │   ├── ship/          # SHIP protocol service
│   │   └── slap/          # SLAP protocol service
│   ├── storage/           # Storage interfaces and implementations
│   ├── types/             # Common types and structures
│   └── utils/             # Utility functions (validation, verification)
├── go.mod
└── go.sum
```

## Usage

### Creating a SHIP Service

```go
import (
    "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/services/ship"
    "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/storage"
)

// Create MongoDB storage
db := // ... initialize MongoDB connection
shipStorage := storage.NewSHIPMongoStorage(db)

// Create SHIP service
shipService := ship.NewSHIPLookupService(shipStorage)

// The service implements the overlay TopicManager interface
// and can be registered with an overlay engine
```

### Using the Wallet Advertiser

Two implementations are available:

1. **Basic Implementation** (`WalletAdvertiser`) - Placeholder with basic structure
2. **Full Implementation** (`FullWalletAdvertiser`) - Complete implementation using go-wallet-toolbox

```go
import (
    "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/advertiser"
    "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/types"
)

// Create full advertiser (requires go-wallet-toolbox)
wa, err := advertiser.NewFullWalletAdvertiser(
    "main",  // or "test"
    privateKeyHex,
    "http://localhost:8080",  // Storage server URL
    "https://myservice.example.com/",
    "https://lookup.example.com",
)

// Initialize (connects to storage server)
err = wa.Init()

// Create advertisements
adsData := []*types.AdvertisementData{
    {
        Protocol:           overlay.ProtocolSHIP,
        TopicOrServiceName: "my-topic",
    },
}
taggedBeef, err := wa.CreateAdvertisements(adsData)
```

See [Wallet Advertiser Documentation](docs/WALLET_ADVERTISER.md) for details on both implementations.

### Validation Utilities

```go
import "github.com/bsv-blockchain/go-overlay-discovery-services/pkg/utils"

// Validate topic names
if utils.IsValidTopicOrServiceName("my-topic") {
    // Valid topic name
}

// Validate advertisable URIs
if utils.IsAdvertisableURI("https://example.com/") {
    // Valid URI for advertisement
}
```

## Storage

The package includes MongoDB storage implementations for both SHIP and SLAP protocols. The storage interfaces are defined in `pkg/storage/interfaces.go` and can be implemented for other storage backends.

### MongoDB Indexes

The MongoDB implementations automatically create the following indexes:
- SHIP: compound index on `identityKey + domain + topic`
- SLAP: compound index on `identityKey + domain + service`

## Protocol Details

### SHIP Advertisement Format

SHIP advertisements use PushDrop scripts with the following fields:
1. Protocol identifier ("SHIP")
2. Identity key (public key hex)
3. Service domain/URI
4. Topic name

### SLAP Advertisement Format

SLAP advertisements use a similar format:
1. Protocol identifier ("SLAP")
2. Identity key (public key hex) 
3. Service domain/URI
4. Service name

### Supported URI Schemes

The following URI schemes are supported for advertisements:
- `https://` - Standard HTTPS
- `https+bsvauth://` - HTTPS with BSV authentication
- `https+bsvauth+smf://` - HTTPS with BSV auth and payment
- `https+bsvauth+scrypt-offchain://` - HTTPS with sCrypt support
- `https+rtt://` - Real-time transaction support
- `wss://` - WebSocket Secure
- `js8c+bsvauth+smf:` - JS8 Call protocol (experimental)

## Dependencies

- [go-sdk](https://github.com/bsv-blockchain/go-sdk) - BSV SDK for Go
- [go-overlay-services](https://github.com/4chain-ag/go-overlay-services) - Overlay services framework
- MongoDB driver for storage implementation

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run with short flag to skip integration tests
go test -short ./...

# Run specific package tests
go test ./pkg/utils -v
go test ./pkg/services/ship -v
```

## License

[License information to be added]

## Contributing

[Contributing guidelines to be added]