// Package authresponse handles NATS authorization request processing, including
// user authentication, JWT generation, and response creation with optional xkey
// encryption. It integrates with a UserRepository to validate user credentials
// and uses key pairs for signing and encrypting responses.
package authresponse

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/tokenvalidation"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go/micro"
	"github.com/sirupsen/logrus"
)

// Handler processes NATS authorization requests.
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

// HandleRequest processes an incoming NATS authorization request.
// It decodes the request, validates the user, generates a user JWT, and responds
// with a signed authorization response, optionally encrypted with xkey.
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

	// Validate user credentials
	user, userID, err := h.validateUser(rc)
	if err != nil {
		h.respond(req, rc.UserNkey, rc.Server.ID, "", err.Error())
		return
	}

	// Generate user JWT, using userID from token or rc.ConnectOptions.Username
	username := userID
	if username == "" {
		username = rc.ConnectOptions.Username
	}
	userJWT, err := h.generateUserJWT(rc.UserNkey, username, user)
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

// validateUser validates the user based on the AuthorizationRequestClaims.
// It supports token-based authentication using nats_token (extracting user_id from token)
// and username/password authentication. For token-based auth, it converts permissions
// from map[string]any to jwt.Permissions, including resp permissions.
func (h *Handler) validateUser(rc *jwt.AuthorizationRequestClaims) (*auth.User, string, error) {
	// Token-based authentication
	if rc.ConnectOptions.Token != "" {
		// userID, permissions, err := tokenvalidation.ValidateNatsToken(rc.ConnectOptions.Token)
		user, err := tokenvalidation.ValidateNatsToken(rc.ConnectOptions.Token)
		if err != nil {
			logrus.WithError(err).Error("Failed to validate nats_token")
			return nil, "", fmt.Errorf("validating nats_token: %v", err)
		}
		userID := user.UserID
		permissions := user.Permissions

		// Convert permissions to jwt.Permissions
		jwtPerms := jwt.Permissions{}
		if pub, ok := permissions["pub"].(map[string]any); ok {
			var pubPerm jwt.Permission
			if allow, ok := pub["allow"].([]any); ok {
				allowStrings := make([]string, len(allow))
				for i, v := range allow {
					allowStrings[i] = v.(string)
				}
				pubPerm.Allow = allowStrings
			}
			if deny, ok := pub["deny"].([]any); ok {
				denyStrings := make([]string, len(deny))
				for i, v := range deny {
					denyStrings[i] = v.(string)
				}
				pubPerm.Deny = denyStrings
			}
			if len(pubPerm.Allow) > 0 || len(pubPerm.Deny) > 0 {
				jwtPerms.Pub = pubPerm
			}
		}
		if sub, ok := permissions["sub"].(map[string]any); ok {
			var subPerm jwt.Permission
			if allow, ok := sub["allow"].([]any); ok {
				allowStrings := make([]string, len(allow))
				for i, v := range allow {
					allowStrings[i] = v.(string)
				}
				subPerm.Allow = allowStrings
			}
			if deny, ok := sub["deny"].([]any); ok {
				denyStrings := make([]string, len(deny))
				for i, v := range deny {
					denyStrings[i] = v.(string)
				}
				subPerm.Deny = denyStrings
			}
			if len(subPerm.Allow) > 0 || len(subPerm.Deny) > 0 {
				jwtPerms.Sub = subPerm
			}
		}
		if resp, ok := permissions["resp"].(map[string]any); ok {
			if maxMsgs, ok := resp["max"].(float64); ok {
				jwtPerms.Resp = &jwt.ResponsePermission{MaxMsgs: int(maxMsgs)}
			}
		}
		logrus.WithFields(logrus.Fields{
			"user_id":    userID,
			"token_hash": fmt.Sprintf("%x", sha256.Sum256([]byte(rc.ConnectOptions.Token)))[:8],
		}).Info("Validated nats_token")

		return &auth.User{
			Permissions: jwtPerms,
			Pass:        "",           // Password not used for token auth
			Account:     user.Account, // Match alice's account from New()
		}, userID, nil
	}

	// Username/password authentication
	if rc.ConnectOptions.Username == "" || rc.ConnectOptions.Password == "" {
		logrus.Error("Username or password missing")
		return nil, "", errors.New("username or password missing")
	}
	user, exists := h.userRepo.Get(rc.ConnectOptions.Username)
	if !exists {
		logrus.WithFields(logrus.Fields{
			"username": rc.ConnectOptions.Username,
		}).Error("User not found")
		return nil, "", errors.New("user not found")
	}
	if user.Pass != rc.ConnectOptions.Password {
		logrus.WithFields(logrus.Fields{
			"username": rc.ConnectOptions.Username,
		}).Error("Invalid credentials")
		return nil, "", errors.New("invalid credentials")
	}
	logrus.WithFields(logrus.Fields{
		"username": rc.ConnectOptions.Username,
		"Pass":     rc.ConnectOptions.Password,
		"Account":  user.Account,
	}).Info("Validated user login/pass")

	return user, "", nil
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
		if err := req.Respond([]byte("Failed to encoding response JWT")); err != nil {
			log.Printf("failed to send response: %v", err)
		}
		return
	}

	// Encrypt response if xkey is present
	xkey := req.Headers().Get("Nats-Server-Xkey")
	if xkey != "" {
		if h.keyPairs.Curve == nil {
			log.Printf("xkey encryption not supported: no curve key pair")
			if err := req.Respond([]byte("Encryption not supported: missing curve key pair")); err != nil {
				log.Printf("failed to send response: %v", err)
			}
			return
		}
		encrypted, err := h.keyPairs.Curve.Seal([]byte(data), xkey)
		if err != nil {
			log.Printf("encrypting response JWT: %v", err)
			if err := req.Respond([]byte("Failed to encrypt response")); err != nil {
				log.Printf("failed to send response: %v", err)
			}
			return
		}
		data = string(encrypted)
	}
	// Send the final response
	if err := req.Respond([]byte(data)); err != nil {
		log.Printf("failed to send response: %v", err)
	}
}
