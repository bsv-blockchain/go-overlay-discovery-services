# Changelog

## [0.1.0] - 2025-07-09

### Added
- Initial Go port of overlay-discovery-services from TypeScript
- Complete implementation of SHIP (Service Host Interconnect Protocol) service
- Complete implementation of SLAP (Service Lookup Availability Protocol) service
- MongoDB storage implementations for both SHIP and SLAP protocols
- Comprehensive validation utilities:
  - `IsValidTopicOrServiceName` for validating topic/service names
  - `IsAdvertisableURI` for validating advertisement URIs with support for multiple schemes
- Token signature verification using BRC-48 protocol
- WalletAdvertiser implementations:
  - Basic placeholder implementation for simple testing
  - Full implementation using go-wallet-toolbox with complete functionality
- Full test coverage for all components
- API documentation and usage examples

### Key Features
- Compatible with go-overlay-services framework
- Returns formulas in lookup responses (matching TypeScript implementation)
- Supports all BRC-101 URI schemes:
  - https:// (standard HTTPS)
  - https+bsvauth:// (with BSV authentication)
  - https+bsvauth+smf:// (with auth and payment)
  - https+bsvauth+scrypt-offchain:// (with sCrypt support)
  - https+rtt:// (real-time transactions)
  - wss:// (WebSocket Secure)
  - js8c+bsvauth+smf: (JS8 Call protocol)

### Dependencies
- github.com/bsv-blockchain/go-sdk v1.2.4
- github.com/bsv-blockchain/go-wallet-toolbox v0.1.0
- github.com/4chain-ag/go-overlay-services v0.1.0
- go.mongodb.org/mongo-driver v1.17.4
- github.com/stretchr/testify v1.10.0

### Notes
- Uses local go-overlay-services and go-wallet-toolbox via go.mod replace directives
- Full WalletAdvertiser implementation requires running wallet storage server
- PushDrop implementation uses wallet-based template from go-sdk
- Advertisement revocation not yet fully implemented but architecture is in place