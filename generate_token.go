package main

import (
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type NatsTokenClaims struct {
	UserID      string                 `json:"user_id"`
	Permissions map[string]interface{} `json:"permissions"`
	Account     string                 `json:"account"`
	jwt.StandardClaims
}

func main() {
	// Секретный ключ (должен совпадать с NATS_TOKEN_SECRET)
	secret := "8K7j9zX5pQw2mL4vN6tY3rF8hG1kJ0eD2uC9xW3aB4y"

	// Формируем пермишены для alice
	permissions := map[string]interface{}{
		"pub": map[string]interface{}{
			"allow": []string{"$JS.API.STREAM.LIST"},
			"deny":  []string{},
		},
		"sub": map[string]interface{}{
			"allow": []string{"_INBOX.>", "TEST.test"},
			"deny":  []string{},
		},
		"resp": map[string]interface{}{
			"max": float64(1),
		},
	}

	// Создаем claims
	claims := &NatsTokenClaims{
		UserID:      "alice",
		Permissions: permissions,
		Account:     "DEVELOPMENT",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(), // Срок действия 24 часа
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Создаем токен с HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Подписываем токен
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		fmt.Printf("Error signing token: %v\n", err)
		return
	}

	// Выводим токен
	fmt.Printf("nats_token: %s\n", tokenString)
}
