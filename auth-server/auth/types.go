package auth

import (
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

type KeyPairs struct {
	Issuer  nkeys.KeyPair
	Curve   nkeys.KeyPair
	HasXKey bool
}

type User struct {
	Permissions jwt.Permissions
	Pass        string
	Account     string
}
