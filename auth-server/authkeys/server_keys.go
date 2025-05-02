// Package authkeys for working with keys for server connect
package authkeys

import (
	"fmt"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"strings"

	"github.com/nats-io/nkeys"
)

// Package authkeys provides utilities for parsing NATS key pairs used in server
// authentication. It handles issuer account signing keys and optional xkey seeds,
// converting them into auth.KeyPairs for use in authentication configurations.

// Parse creates an auth.KeyPairs from the provided issuer and xkey seeds.
// The issuerSeed is required and must be a valid NATS account seed (starting with 'SA').
// The xkeySeed is optional; if provided, it must be a valid NATS xkey seed (starting with 'SX').
// Returns an error if either seed is invalid or cannot be parsed.
func Parse(issuerSeed, xkeySeed string) (*auth.KeyPairs, error) {
	if issuerSeed == "" {
		return nil, fmt.Errorf("issuer seed cannot be empty")
	}

	kp := &auth.KeyPairs{}

	// Parse issuer seed
	issuer, err := nkeys.FromSeed([]byte(issuerSeed))
	if err != nil {
		return nil, fmt.Errorf("parsing issuer seed %q: %w", truncateSeed(issuerSeed), err)
	}
	if !strings.HasPrefix(issuerSeed, "SA") {
		return nil, fmt.Errorf("issuer seed %q must start with 'SA'", truncateSeed(issuerSeed))
	}
	kp.Issuer = issuer

	// Parse optional xkey seed
	if xkeySeed != "" {
		curve, err := nkeys.FromSeed([]byte(xkeySeed))
		if err != nil {
			return nil, fmt.Errorf("parsing xkey seed %q: %w", truncateSeed(xkeySeed), err)
		}
		if !strings.HasPrefix(xkeySeed, "SX") {
			return nil, fmt.Errorf("xkey seed %q must start with 'SX'", truncateSeed(xkeySeed))
		}
		kp.Curve = curve
		kp.HasXKey = true
	}

	return kp, nil
}

// truncateSeed returns a truncated version of the seed for safe error reporting.
func truncateSeed(seed string) string {
	if len(seed) > 3 {
		return seed[:3] + "..."
	}
	return seed
}
