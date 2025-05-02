// authkeys/server_keys.go

// Package authkeys for working with keys for server connect
package authkeys

import (
	"fmt"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/nkeys"
)

// Parse return KeyPairs from config
// Parse the issuer account signing key
// Parse the xkey seed if present
func Parse(issuerSeed, xkeySeed string) (*auth.KeyPairs, error) {
	kp := &auth.KeyPairs{}

	var err error
	kp.Issuer, err = nkeys.FromSeed([]byte(issuerSeed))
	if err != nil {
		return nil, fmt.Errorf("error parsing issuer seed: %w", err)
	}

	if xkeySeed != "" {
		kp.Curve, err = nkeys.FromSeed([]byte(xkeySeed))
		if err != nil {
			return nil, fmt.Errorf("error parsing xkey seed: %w", err)
		}
		kp.HasXKey = true
	}

	return kp, nil
}
