module github.com/bsv-blockchain/go-overlay-discovery-services

go 1.24.3

require (
	github.com/bsv-blockchain/go-overlay-services v0.1.1
	github.com/bsv-blockchain/go-sdk v1.2.8
	github.com/stretchr/testify v1.10.0
	go.mongodb.org/mongo-driver v1.12.1
)

replace github.com/bsv-blockchain/go-overlay-services => ../go-overlay-services

replace github.com/bsv-blockchain/go-sdk => ../go-sdk

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/snappy v0.0.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/montanaflynn/stats v0.0.0-20171201202039-1bf9dbcd8cbe // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
