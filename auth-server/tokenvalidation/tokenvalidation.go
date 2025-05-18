// Package tokenvalidation provides functionality for validating JWT tokens used in NATS authentication.
package tokenvalidation

import (
	"errors"
	"fmt"
	"os"

	"github.com/dgrijalva/jwt-go"
)

// NatsTokenClaims represents the claims in a JWT nats_token, including user ID and permissions.
type NatsTokenClaims struct {
	UserID      string         `json:"user_id"`
	Permissions map[string]any `json:"permissions"`
	Account     string         `json:"account"`
	jwt.StandardClaims
}

// ValidateNatsToken validates a JWT nats_token and returns the user ID and permissions.
// It checks the token's signature and expiration using the NATS_TOKEN_SECRET environment variable.
// Returns an error if the token is invalid, expired, or uses an unexpected signing method.
func ValidateNatsToken(tokenString string) (string, map[string]any, error) {
	secret := os.Getenv("NATS_TOKEN_SECRET")
	if secret == "" {
		return "", nil, fmt.Errorf("NATS_TOKEN_SECRET environment variable is not set")
	}

	claims := &NatsTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to validate token: %w", err)
	}
	if !token.Valid {
		return "", nil, fmt.Errorf("token is invalid")
	}
	if claims.UserID == "" {
		return "", nil, errors.New("missing user_id in token")
	}

	return claims.UserID, claims.Permissions, nil
}
