// Package main generates a NATS JWT token from a JSON input string and optionally tests
// connectivity to a NATS server. The program accepts the JSON input via the -input flag,
// the NATS server URL via the -server flag, and a -test flag to control whether to test
// the connection. It validates the input, generates a signed JWT token using HMAC-SHA256,
// and, if -test is true, uses the token to connect to the NATS server and list all streams.
// The program is designed for NATS-based applications requiring secure authentication
// and authorization.
//
// The JSON input must include a non-empty user_id. Permissions, account, and TTL are
// optional. If permissions are absent or incomplete, publish and subscribe permissions
// default to denying all (empty allow and deny lists). If TTL is not specified, the token
// expires after 2 minutes. The token is signed using the NATS_TOKEN_SECRET environment
// variable. For NATS request-reply patterns, the permissions.sub.allow field must include
// "_INBOX.>" to allow subscriptions to reply subjects.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/nats-io/nats.go"
)

// TestNatsTokenClaims represents the custom claims structure for NATS JWT tokens.
// It includes user ID, permissions, account details, TTL, and standard JWT
// registered claims.
type TestNatsTokenClaims struct {
	UserID               string         `json:"user_id"`     // Unique identifier for the user (required)
	Permissions          map[string]any `json:"permissions"` // User permissions for NATS subjects (optional)
	Account              string         `json:"account"`     // Associated NATS account (optional)
	TTL                  int            `json:"ttl"`         // Token time-to-live in seconds (optional)
	jwt.RegisteredClaims                // Standard JWT claims (e.g., exp, iat)
}

// GenerateNatsToken generates a NATS JWT token from a JSON input string.
//
// The input JSON must include a non-empty user_id. Permissions, account, and TTL
// are optional. If permissions are absent or incomplete, pub and sub permissions
// default to denying all (empty allow and deny lists). If TTL is not provided,
// the token expires after 2 minutes. The token is signed using the
// NATS_TOKEN_SECRET environment variable with HMAC-SHA256.
//
// For NATS request-reply patterns, the permissions.sub.allow field must include
// "_INBOX.>" to allow subscriptions to reply subjects (e.g., "_INBOX.<random>.*").
// Failing to include this permission may result in subscription errors.
//
// Args:
//
//	inputJSON (string): JSON string containing user_id, permissions, account, and ttl.
//
// Returns:
//
//	string: The signed JWT token string.
//	error: An error if the input is invalid, the secret is missing, or token generation fails.
func GenerateNatsToken(inputJSON string) (string, error) {
	// Parse JSON input
	var claims TestNatsTokenClaims
	if err := json.Unmarshal([]byte(inputJSON), &claims); err != nil {
		return "", fmt.Errorf("failed to parse JSON input: %w", err)
	}

	// Validate user_id
	if claims.UserID == "" {
		return "", errors.New("user_id is required")
	}

	// Initialize permissions if not provided
	if claims.Permissions == nil {
		claims.Permissions = map[string]any{
			"pub": map[string]any{
				"allow": []string{},
				"deny":  []string{},
			},
			"sub": map[string]any{
				"allow": []string{},
				"deny":  []string{},
			},
		}
	} else {
		// Ensure pub permissions default to deny all if not specified
		if _, ok := claims.Permissions["pub"]; !ok {
			claims.Permissions["pub"] = map[string]any{
				"allow": []string{},
				"deny":  []string{},
			}
		}
		// Ensure sub permissions default to deny all if not specified
		if _, ok := claims.Permissions["sub"]; !ok {
			claims.Permissions["sub"] = map[string]any{
				"allow": []string{},
				"deny":  []string{},
			}
		}
		// Handle resp permissions, renaming max to maxMsgs
		if resp, ok := claims.Permissions["resp"].(map[string]any); ok {
			if maxMsgs, ok := resp["max"].(float64); ok {
				resp["maxMsgs"] = maxMsgs
				delete(resp, "max")
				claims.Permissions["resp"] = resp
			}
		}
	}

	// Set default TTL if not provided (2 minutes)
	if claims.TTL <= 0 {
		claims.TTL = 120 // 2 minutes in seconds
	}

	// Set registered claims
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(claims.TTL) * time.Second)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	// Retrieve secret from environment variable
	secret := os.Getenv("NATS_TOKEN_SECRET")
	if secret == "" {
		return "", errors.New("NATS_TOKEN_SECRET environment variable is not set")
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	return tokenString, nil
}

// TestNatsConnection tests connectivity to a NATS server using the provided JWT token.
//
// It connects to the specified NATS server using the JWT token for authentication
// and attempts to list all streams (equivalent to `nats stream ls -a`). The function
// returns the list of stream names or an error if the connection or stream listing fails.
//
// Args:
//
//	serverURL (string): The NATS server URL (e.g., "nats://localhost:4222").
//	jwtToken (string): The JWT token for authentication.
//
// Returns:
//
//	[]string: List of stream names if successful.
//	error: An error if the connection or stream listing fails.
func TestNatsConnection(serverURL, jwtToken string) ([]string, error) {
	// Connect to NATS server with JWT token
	nc, err := nats.Connect(serverURL, nats.Token(jwtToken))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS server: %w", err)
	}
	defer nc.Close()

	// Get JetStream context
	js, err := nc.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// List all streams
	var streams []string
	for stream := range js.Streams() {
		if stream == nil {
			continue
		}
		streams = append(streams, stream.Config.Name)
	}

	return streams, nil
}

func main() {
	// Define command-line flags
	inputJSON := flag.String("input", "", "JSON string containing user_id, permissions, account, and ttl")
	serverURL := flag.String("server", "nats://localhost:4222", "NATS server URL")
	testConn := flag.Bool("test", false, "Test NATS connection with the generated token (true/false)")
	flag.Parse()

	// Default JSON input, including "_INBOX.>" in sub permissions to support NATS request-reply
	defaultJSON := `{
		"user_id": "bob",
		"permissions": {
			"pub": {
				"allow": ["$JS.API.>"],
				"deny": []
			},
			"resp": {
				"max": 1
			},
			"sub": {
				"allow": ["_INBOX.>", "TEST.>"],
				"deny": []
			}
		},
		"account": "DEVELOPMENT",
		"ttl": 600
	}`

	// Use provided input or default
	jsonInput := *inputJSON
	if jsonInput == "" {
		jsonInput = defaultJSON
		fmt.Println("No input provided; using default JSON with _INBOX.> permission for NATS request-reply")
	}

	// Generate token
	tokenString, err := GenerateNatsToken(jsonInput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating token: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated token: %s\n", tokenString)

	// Test NATS connection if -test is true
	if *testConn {
		streams, err := TestNatsConnection(*serverURL, tokenString)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error testing NATS connection: %v\n", err)
			os.Exit(1)
		}

		if len(streams) == 0 {
			fmt.Println("No Streams defined")
		} else {
			fmt.Println("Streams found:")
			for _, stream := range streams {
				fmt.Printf("- %s\n", stream)
			}
		}
	}
}
