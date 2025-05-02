package authkeys

import (
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"strings"
	"testing"

	"github.com/nats-io/nkeys"
)

// TestParse tests the Parse function for various seed inputs.
func TestParse(t *testing.T) {
	// Generate valid seeds for testing
	accountKP, err := nkeys.CreatePair(nkeys.PrefixByteAccount)
	if err != nil {
		t.Fatalf("Failed to create account key pair: %v", err)
	}
	accountSeed, err := accountKP.Seed()
	if err != nil {
		t.Fatalf("Failed to get account seed: %v", err)
	}

	curveKP, err := nkeys.CreatePair(nkeys.PrefixByteCurve)
	if err != nil {
		t.Fatalf("Failed to create curve key pair: %v", err)
	}
	curveSeed, err := curveKP.Seed()
	if err != nil {
		t.Fatalf("Failed to get curve seed: %v", err)
	}

	// Test cases
	tests := []struct {
		name          string
		issuerSeed    string
		xkeySeed      string
		expectError   bool
		expectedError string
		validateKP    func(t *testing.T, kp *auth.KeyPairs)
	}{
		{
			name:        "valid issuer and xkey",
			issuerSeed:  string(accountSeed),
			xkeySeed:    string(curveSeed),
			expectError: false,
			validateKP: func(t *testing.T, kp *auth.KeyPairs) {
				if kp == nil {
					t.Fatal("Expected non-nil KeyPairs")
				}
				if kp.Issuer == nil {
					t.Error("Expected non-nil Issuer")
				}
				if kp.Curve == nil {
					t.Error("Expected non-nil Curve")
				}
				if !kp.HasXKey {
					t.Error("Expected HasXKey to be true")
				}
			},
		},
		{
			name:        "valid issuer only",
			issuerSeed:  string(accountSeed),
			xkeySeed:    "",
			expectError: false,
			validateKP: func(t *testing.T, kp *auth.KeyPairs) {
				if kp == nil {
					t.Fatal("Expected non-nil KeyPairs")
				}
				if kp.Issuer == nil {
					t.Error("Expected non-nil Issuer")
				}
				if kp.Curve != nil {
					t.Error("Expected nil Curve")
				}
				if kp.HasXKey {
					t.Error("Expected HasXKey to be false")
				}
			},
		},
		{
			name:          "empty issuer seed",
			issuerSeed:    "",
			xkeySeed:      string(curveSeed),
			expectError:   true,
			expectedError: "issuer seed cannot be empty",
		},
		{
			name:          "invalid issuer seed",
			issuerSeed:    "INVALID_SEED",
			xkeySeed:      "",
			expectError:   true,
			expectedError: "parsing issuer seed \"INV...\":",
		},
		{
			name:          "wrong issuer seed prefix",
			issuerSeed:    string(curveSeed), // Using xkey seed as issuer
			xkeySeed:      "",
			expectError:   true,
			expectedError: "issuer seed \"SXA...\" must start with 'SA'",
		},
		{
			name:          "invalid xkey seed",
			issuerSeed:    string(accountSeed),
			xkeySeed:      "INVALID_XKEY",
			expectError:   true,
			expectedError: "parsing xkey seed \"INV...\":",
		},
		{
			name:          "wrong xkey seed prefix",
			issuerSeed:    string(accountSeed),
			xkeySeed:      string(accountSeed), // Using account seed as xkey
			expectError:   true,
			expectedError: "xkey seed \"SAA...\" must start with 'SX'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kp, err := Parse(tt.issuerSeed, tt.xkeySeed)
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected an error, but got none")
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tt.validateKP != nil {
					tt.validateKP(t, kp)
				}
			}
		})
	}
}
