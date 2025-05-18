package tokenvalidation

import (
	"crypto/rand"
	"crypto/rsa"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func TestValidateNatsToken(t *testing.T) {
	// Установка NATS_TOKEN_SECRET
	if err := os.Setenv("NATS_TOKEN_SECRET", "test-secret-1234567890"); err != nil {
		t.Fatalf("Failed to set NATS_TOKEN_SECRET: %v", err)
	}
	// Очистка переменной после теста
	defer func() {
		if err := os.Unsetenv("NATS_TOKEN_SECRET"); err != nil {
			t.Errorf("Failed to unset NATS_TOKEN_SECRET: %v", err)
		}
	}()
	// Valid token
	t.Run("ValidToken", func(t *testing.T) {
		claims := &NatsTokenClaims{
			UserID:      "user123",
			Permissions: map[string]any{"publish": []string{"topic1"}, "subscribe": []string{"topic2"}},
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte("test-secret-1234567890"))
		if err != nil {
			t.Fatalf("Failed to create test token: %v", err)
		}

		userID, perms, err := ValidateNatsToken(tokenString)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if userID != "user123" {
			t.Errorf("Expected userID 'user123', got %v", userID)
		}
		if perms["publish"].([]any)[0] != "topic1" {
			t.Errorf("Expected permission 'topic1', got %v", perms["publish"])
		}
	})

	// Invalid signature
	t.Run("InvalidSignature", func(t *testing.T) {
		claims := &NatsTokenClaims{
			UserID:      "user123",
			Permissions: map[string]any{"publish": []string{"topic1"}},
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("wrong-secret"))

		_, _, err := ValidateNatsToken(tokenString)
		if err == nil || !strings.Contains(err.Error(), "signature is invalid") {
			t.Errorf("Expected signature error, got %v", err)
		}
	})

	// Expired token
	t.Run("ExpiredToken", func(t *testing.T) {
		claims := &NatsTokenClaims{
			UserID:      "user123",
			Permissions: map[string]any{"publish": []string{"topic1"}},
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(-time.Hour).Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("test-secret-1234567890"))

		_, _, err := ValidateNatsToken(tokenString)
		if err == nil || !strings.Contains(err.Error(), "token is expired") {
			t.Errorf("Expected expiration error, got %v", err)
		}
	})

	// Wrong signing algorithm
	t.Run("WrongAlgorithm", func(t *testing.T) {
		claims := &NatsTokenClaims{
			UserID:      "user123",
			Permissions: map[string]any{"publish": []string{"topic1"}},
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
			},
		}
		// Generate RSA key for RS256 signing
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("Failed to generate RSA key: %v", err)
		}
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		tokenString, err := token.SignedString(privateKey)
		if err != nil {
			t.Fatalf("Failed to create RS256 token: %v", err)
		}

		_, _, err = ValidateNatsToken(tokenString)
		if err == nil || !strings.Contains(err.Error(), "unexpected signing method") {
			t.Errorf("Expected algorithm error, got %v", err)
		}
	})

	// Missing NATS_TOKEN_SECRET
	t.Run("MissingSecret", func(t *testing.T) {
		if err := os.Unsetenv("NATS_TOKEN_SECRET"); err != nil {
			t.Fatalf("Failed to unset NATS_TOKEN_SECRET: %v", err)
		}
		_, _, err := ValidateNatsToken("dummy-token")
		if err == nil || !strings.Contains(err.Error(), "NATS_TOKEN_SECRET environment variable is not set") {
			t.Errorf("Expected missing secret error, got %v", err)
		}
	})
}
