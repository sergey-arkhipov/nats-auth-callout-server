// Package tokenvalidation provides functionality for validating NATS JWT tokens.
// It verifies the token's signature, expiration, and claims, ensuring secure
// authentication and authorization for NATS-based applications. The package
// supports HMAC-SHA256 signature verification and custom claims for user ID and
// permissions. It uses structured logging for debugging and error reporting.
//
// The main function, ValidateNatsToken, takes a JWT token string, validates its
// format, signature, and claims, and returns the user ID and permissions if valid.
// It relies on the NATS_TOKEN_SECRET environment variable for the signing key.
package tokenvalidation

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

// NatsTokenClaims represents the custom claims structure for NATS JWT tokens.
// It includes user ID, permissions, account details, and standard JWT registered claims.
type NatsTokenClaims struct {
	UserID               string         `json:"user_id"`     // Unique identifier for the user
	Permissions          map[string]any `json:"permissions"` // User permissions for NATS subjects
	Account              string         `json:"account"`     // Associated NATS account
	jwt.RegisteredClaims                // Standard JWT claims (e.g., exp, iat)
}

// ValidateNatsToken validates a NATS JWT token and extracts its user ID and permissions.
//
// It performs the following checks:
// 1. Ensures the NATS_TOKEN_SECRET environment variable is set.
// 2. Verifies the token format (three parts: header, payload, signature).
// 3. Parses and validates the JWT claims, including signature and expiration.
// 4. Ensures the user ID is present in the claims.
// 5. Returns the user ID and permissions if all checks pass.
//
// Args:
//
//	tokenString (string): The JWT token to validate.
//
// Returns:
//
//	string: The user ID extracted from the token claims.
//	map[string]any: The permissions extracted from the token claims.
//	error: An error if validation fails (e.g., invalid format, signature, or expired token).
func ValidateNatsToken(tokenString string) (string, map[string]any, error) {
	// Retrieve the secret key from environment variable
	secret := os.Getenv("NATS_TOKEN_SECRET")
	if secret == "" {
		logrus.Error("NATS_TOKEN_SECRET environment variable is not set")
		return "", nil, errors.New("NATS_TOKEN_SECRET environment variable is not set")
	}

	// Check basic token format
	if len(strings.Split(tokenString, ".")) != 3 {
		logrus.WithField("token", tokenString[:10]+"...").Debug("Invalid token format")
		return "", nil, errors.New("invalid token format")
	}

	// Parse JWT with custom claims
	claims := &NatsTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logrus.WithField("method", token.Header["alg"]).Debug("Unexpected signing method")
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	// Log token validation details
	logrus.WithFields(logrus.Fields{
		"token":   tokenString[:10] + "...",
		"error":   err,
		"valid":   token != nil && token.Valid,
		"user_id": claims.UserID,
		"raw":     token.Raw,
		"exp":     claims.ExpiresAt,
	}).Debug("Token validation result")

	if err != nil {
		logrus.WithError(err).Debug("JWT parsing failed")
		return "", nil, err
	}
	if !token.Valid {
		logrus.Debug("Token is not valid")
		return "", nil, errors.New("invalid token signature")
	}

	// Check token expiration
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		logrus.WithField("exp", claims.ExpiresAt).Debug("Token expired")
		return "", nil, errors.New("token expired")
	}

	// Ensure user ID is present
	if claims.UserID == "" {
		logrus.Debug("Missing user_id in token")
		return "", nil, errors.New("missing user_id in token")
	}

	return claims.UserID, claims.Permissions, nil
}
