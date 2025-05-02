package authresponse

import (
	"fmt"
	"log"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/auth"

	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats.go/micro"
)

type Handler struct {
	keyPairs *auth.KeyPairs
	userRepo UserRepository
}

type UserRepository interface {
	Get(username string) (*auth.User, bool)
}

func NewHandler(keyPairs *auth.KeyPairs, userRepo UserRepository) *Handler {
	return &Handler{
		keyPairs: keyPairs,
		userRepo: userRepo,
	}
}

// Define the message handler for the authorization request.
func (h *Handler) HandleRequest(req micro.Request) {
	var token []byte
	var err error // Declare error variable

	// Check for Xkey header and decrypt
	xkey := req.Headers().Get("Nats-Server-Xkey")
	if len(xkey) > 0 {
		if h.keyPairs.Curve == nil {
			h.respond(req, "", "", "", "xkey not supported")
			return
		}

		// Decrypt the message.
		token, err = h.keyPairs.Curve.Open(req.Data(), xkey)
		if err != nil {
			h.respond(req, "", "", "", "error decrypting message")
			return
		}
	} else {
		token = req.Data()
	}

	// Decode the authorization request claims.
	rc, err := jwt.DecodeAuthorizationRequestClaims(string(token))
	if err != nil {
		h.respond(req, "", "", "", err.Error())
		return
	}
	fmt.Println("Request claim->", rc)
	// Used for creating the auth response.
	userNkey := rc.UserNkey
	serverID := rc.Server.ID

	// Check if the user exists.
	// Get test user UserInfo
	user, exists := h.userRepo.Get(rc.ConnectOptions.Username)
	if !exists {
		h.respond(req, userNkey, serverID, "", "user not found")
		return // Just return without error
	}

	// userProfile, ok := users[rc.ConnectOptions.Username]
	fmt.Println("UserInfo-->", user)
	// if !ok {
	// 	respondMsg(req, userNkey, serverID, "", "user not found")
	// 	return
	// }

	// Check if the credential is valid.
	if user.Pass != rc.ConnectOptions.Password {
		h.respond(req, userNkey, serverID, "", "invalid credentials")
		return
	}

	// Prepare a user JWT.
	uc := jwt.NewUserClaims(rc.UserNkey)
	uc.Name = rc.ConnectOptions.Username

	// Audience contains the account in non-operator mode.
	uc.Audience = user.Account

	// Set the associated permissions if present.
	uc.Permissions = user.Permissions

	// Validate the claims.
	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) > 0 {
		h.respond(req, userNkey, serverID, "", "error validating claims")
		return
	}

	// Sign it with the issuer key since this is non-operator mode.
	ejwt, err := uc.Encode(h.keyPairs.Issuer)
	if err != nil {
		h.respond(req, userNkey, serverID, "", "error signing user JWT")
		return
	}

	fmt.Println("User claim is : ", ejwt)
	h.respond(req, userNkey, serverID, ejwt, "")
}

// Helper function to construct an authorization response.
func (h *Handler) respond(req micro.Request, userNkey, serverID, userJwt, errMsg string) {
	rc := jwt.NewAuthorizationResponseClaims(userNkey)
	rc.Audience = serverID
	rc.Error = errMsg
	rc.Jwt = userJwt

	token, err := rc.Encode(h.keyPairs.Issuer)
	if err != nil {
		log.Printf("error encoding response JWT: %s", err)
		req.Respond(nil)
		return
	}

	data := []byte(token)

	// Check if encryption is required.
	xkey := req.Headers().Get("Nats-Server-Xkey")
	if len(xkey) > 0 {
		data, err = h.keyPairs.Curve.Seal(data, xkey)
		if err != nil {
			log.Printf("error encrypting response JWT: %s", err)
			req.Respond(nil)
			return
		}
	}

	req.Respond(data)
}
