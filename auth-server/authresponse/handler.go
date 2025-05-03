// Package authresponse to get request and make response
package authresponse

import (
	"errors"
	"fmt"
	"log"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go/micro"
)

// Package authresponse handles NATS authorization request processing, including
// user authentication, JWT generation, and response creation with optional xkey
// encryption. It integrates with a UserRepository to validate user credentials
// and uses key pairs for signing and encrypting responses.

/*
Handler processes NATS authorization requests, validates user credentials,
and generates signed JWT responses.
*/
type Handler struct {
	keyPairs *auth.KeyPairs
	userRepo UserRepository
}

// UserRepository defines the interface for retrieving user information.
type UserRepository interface {
	Get(username string) (*auth.User, bool)
}

// NewHandler creates a new Handler with the provided key pairs and user repository.
func NewHandler(keyPairs *auth.KeyPairs, userRepo UserRepository) *Handler {
	return &Handler{
		keyPairs: keyPairs,
		userRepo: userRepo,
	}
}

/*
HandleRequest processes an incoming NATS authorization request.
It decodes the request, validates the user, generates a user JWT, and responds
with a signed authorization response, optionally encrypted with xkey.
*/
func (h *Handler) HandleRequest(req micro.Request) {
	// Decode the request token, handling xkey decryption if present
	token, err := h.decodeRequest(req)
	if err != nil {
		h.respond(req, "", "", "", err.Error())
		return
	}

	// Decode authorization request claims
	rc, err := jwt.DecodeAuthorizationRequestClaims(string(token))
	if err != nil {
		h.respond(req, "", "", "", fmt.Sprintf("decoding authorization request: %v", err))
		return
	}
	// log.Printf("Decoded AuthorizationRequestClaims: %+v", rc)
	// Validate user credentials
	user, err := h.validateUser(rc)
	if err != nil {
		h.respond(req, rc.UserNkey, rc.Server.ID, "", err.Error())
		return
	}

	// Generate user JWT
	userJWT, err := h.generateUserJWT(rc.UserNkey, rc.ConnectOptions.Username, user)
	if err != nil {
		h.respond(req, rc.UserNkey, rc.Server.ID, "", fmt.Sprintf("generating user JWT: %v", err))
		return
	}

	// Respond with the signed JWT
	h.respond(req, rc.UserNkey, rc.Server.ID, userJWT, "")
}

// decodeRequest extracts and decodes the request token, handling xkey decryption if needed.
func (h *Handler) decodeRequest(req micro.Request) ([]byte, error) {
	xkey := req.Headers().Get("Nats-Server-Xkey")
	if xkey == "" {
		return req.Data(), nil
	}

	if h.keyPairs.Curve == nil {
		return nil, errors.New("xkey not supported")
	}

	token, err := h.keyPairs.Curve.Open(req.Data(), xkey)
	if err != nil {
		return nil, fmt.Errorf("decrypting message: %w", err)
	}
	return token, nil
}

// validateUser checks if the user exists and has valid credentials.
func (h *Handler) validateUser(rc *jwt.AuthorizationRequestClaims) (*auth.User, error) {
	user, exists := h.userRepo.Get(rc.ConnectOptions.Username)
	if !exists {
		return nil, errors.New("user not found")
	}
	if user.Pass != rc.ConnectOptions.Password {
		return nil, errors.New("invalid credentials")
	}
	return user, nil
}

// generateUserJWT creates and signs a user JWT for the given user.
func (h *Handler) generateUserJWT(userNkey, username string, user *auth.User) (string, error) {
	uc := jwt.NewUserClaims(userNkey)
	uc.Name = username
	uc.Audience = user.Account
	uc.Permissions = user.Permissions

	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) > 0 {
		return "", errors.New("validating claims")
	}

	return uc.Encode(h.keyPairs.Issuer)
}

// respond sends an authorization response with the provided JWT or error message,
// optionally encrypting with xkey.
func (h *Handler) respond(req micro.Request, userNkey, serverID, userJwt, errMsg string) {
	rc := jwt.NewAuthorizationResponseClaims(userNkey)
	rc.Audience = serverID
	rc.Error = errMsg
	rc.Jwt = userJwt

	data, err := rc.Encode(h.keyPairs.Issuer)
	if err != nil {
		log.Printf("encoding response JWT: %v", err)
		req.Respond(nil)
		return
	}

	// Encrypt response if xkey is present
	xkey := req.Headers().Get("Nats-Server-Xkey")
	if xkey != "" {
		if h.keyPairs.Curve == nil {
			log.Printf("xkey encryption not supported: no curve key pair")
			req.Respond([]byte("Encryption not supported: missing curve key pair"))
			return
		}
		encrypted, err := h.keyPairs.Curve.Seal([]byte(data), xkey)
		if err != nil {
			log.Printf("encrypting response JWT: %v", err)
			req.Respond([]byte("Failed to encrypt response"))
			return
		}
		data = string(encrypted)
	}
	req.Respond([]byte(data))
}
