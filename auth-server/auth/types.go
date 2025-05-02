// Package auth defines core authentication types and structures used throughout
// the NATS authentication system. These types are used for:
// - Key pair management (Issuer and Curve keys)
// - User credential and permission storage
// - JWT claim generation and validation
package auth

import (
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

// KeyPairs holds the cryptographic key pairs used for NATS authentication.
// Contains both the issuer key pair (for signing tokens) and optional curve key
// pair (for encryption). The HasXKey flag indicates if curve keys are available.
//
// Usage:
//
//	kp := KeyPairs{
//	    Issuer:  issuerKey,
//	    Curve:   curveKey,
//	    HasXKey: true,
//	}
type KeyPairs struct {
	Issuer  nkeys.KeyPair // Key pair for signing JWTs
	Curve   nkeys.KeyPair // Optional key pair for encryption (XKey)
	HasXKey bool          // True if Curve keys are available
}

// User represents an authenticated NATS user with their permissions and credentials.
// This is typically loaded from persistent storage and used to generate JWT tokens.
//
// Example:
//
//	user := User{
//	    Account: "DEMO",
//	    Pass:    "securepassword",
//	    Permissions: jwt.Permissions{
//	        Pub: &jwt.Permission{Allow: []string{"public.>"}},
//	    },
//	}
type User struct {
	Permissions jwt.Permissions // NATS permissions (pub/sub)
	Pass        string          // User password (hashed in production)
	Account     string          // NATS account name
}
