module sergey-arkhipov/nats-auth-callout-server

go 1.24.2

require (
	github.com/nats-io/jwt/v2 v2.7.4 // Latest is v2.10.7 (from nats-server releases) :cite[1]
	github.com/nats-io/nats.go v1.41.2 // Latest as of 2025-05-02 :cite[5]
	github.com/nats-io/nkeys v0.4.11 // Used in nats.go v1.41.2 :cite[5]
)

require gopkg.in/yaml.v3 v3.0.1

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
)
