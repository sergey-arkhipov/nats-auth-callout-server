package tokenvalidation

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestMinimalJwtValidation(t *testing.T) {
	secret := "test-secret-1234567890"
	claims := &NatsTokenClaims{
		UserID:  "alice",
		Account: "DEVELOPMENT",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}

	// Оригинальный токен
	parsedClaims := &NatsTokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(tokenString, parsedClaims, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil || !parsedToken.Valid {
		t.Errorf("Expected valid original token, got error: %v, valid: %v", err, parsedToken.Valid)
	}
	if parsedClaims.UserID != "alice" {
		t.Errorf("Expected userID alice, got %v", parsedClaims.UserID)
	}

	// Измененный токен (последний символ → 8)
	modifiedToken := tokenString[:len(tokenString)-1] + "6"
	parsedClaims = &NatsTokenClaims{}
	parsedToken, err = jwt.ParseWithClaims(modifiedToken, parsedClaims, func(_ *jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err == nil || parsedToken.Valid {
		t.Errorf("Expected invalid signature for modified token, got error: %v, valid: %v", err, parsedToken.Valid)
	}
	if !strings.Contains(err.Error(), "signature is invalid") {
		t.Errorf("Expected signature is invalid, got %v", err)
	}
}
