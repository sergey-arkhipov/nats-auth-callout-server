package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/authkeys"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/authresponse"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/config"
	"sergey-arkhipov/nats-auth-callout-server/auth-server/usersdebug"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configFile := flag.String("config", "config.yml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	natsURL := cfg.Nats.URL
	natsUser := cfg.Nats.User
	natsPass := cfg.Nats.Pass

	issuerSeed := cfg.Auth.IssuerSeed
	xkeySeed := cfg.Auth.XKeySeed

	// Usage:
	keyPairs, err := authkeys.Parse(issuerSeed, xkeySeed)
	if err != nil {
		return err
	}

	// Open the NATS connection passing the auth account creds file.
	nc, err := nats.Connect(natsURL, nats.UserInfo(natsUser, natsPass))
	if err != nil {
		return err
	}
	defer nc.Drain()

	// Helper function to construct an authorization response.
	// respondMsg := func(req micro.Request, userNkey, serverID, userJwt, errMsg string) {
	// 	rc := jwt.NewAuthorizationResponseClaims(userNkey)
	// 	rc.Audience = serverID
	// 	rc.Error = errMsg
	// 	rc.Jwt = userJwt
	//
	// 	token, err := rc.Encode(keyPairs.Issuer)
	// 	if err != nil {
	// 		log.Printf("error encoding response JWT: %s", err)
	// 		req.Respond(nil)
	// 		return
	// 	}
	//
	// 	data := []byte(token)
	//
	// 	// Check if encryption is required.
	// 	xkey := req.Headers().Get("Nats-Server-Xkey")
	// 	if len(xkey) > 0 {
	// 		data, err = keyPairs.Curve.Seal(data, xkey)
	// 		if err != nil {
	// 			log.Printf("error encrypting response JWT: %s", err)
	// 			req.Respond(nil)
	// 			return
	// 		}
	// 	}
	//
	// 	req.Respond(data)
	// }
	//
	// // Define the message handler for the authorization request.
	// msgHandler := func(req micro.Request) {
	// 	var token []byte
	//
	// 	// Check for Xkey header and decrypt
	// 	xkey := req.Headers().Get("Nats-Server-Xkey")
	// 	if len(xkey) > 0 {
	// 		if keyPairs.Curve == nil {
	// 			respondMsg(req, "", "", "", "xkey not supported")
	// 			return
	// 		}
	//
	// 		// Decrypt the message.
	// 		token, err = keyPairs.Curve.Open(req.Data(), xkey)
	// 		if err != nil {
	// 			respondMsg(req, "", "", "", "error decrypting message")
	// 			return
	// 		}
	// 	} else {
	// 		token = req.Data()
	// 	}
	//
	// 	// Decode the authorization request claims.
	// 	rc, err := jwt.DecodeAuthorizationRequestClaims(string(token))
	// 	if err != nil {
	// 		respondMsg(req, "", "", "", err.Error())
	// 		return
	// 	}
	// 	fmt.Println("Request claim->", rc)
	// 	// Used for creating the auth response.
	// 	userNkey := rc.UserNkey
	// 	serverID := rc.Server.ID
	//
	// 	// Check if the user exists.
	// 	// Get test user UserInfo
	// 	user, exists := usersdebug.Get(rc.ConnectOptions.Username)
	// 	if !exists {
	// 		respondMsg(req, userNkey, serverID, "", "user not found")
	// 		return // Just return without error
	// 	}
	//
	// 	// userProfile, ok := users[rc.ConnectOptions.Username]
	// 	fmt.Println("UserInfo-->", user)
	// 	// if !ok {
	// 	// 	respondMsg(req, userNkey, serverID, "", "user not found")
	// 	// 	return
	// 	// }
	//
	// 	// Check if the credential is valid.
	// 	if user.Pass != rc.ConnectOptions.Password {
	// 		respondMsg(req, userNkey, serverID, "", "invalid credentials")
	// 		return
	// 	}
	//
	// 	// Prepare a user JWT.
	// 	uc := jwt.NewUserClaims(rc.UserNkey)
	// 	uc.Name = rc.ConnectOptions.Username
	//
	// 	// Audience contains the account in non-operator mode.
	// 	uc.Audience = user.Account
	//
	// 	// Set the associated permissions if present.
	// 	uc.Permissions = user.Permissions
	//
	// 	// Validate the claims.
	// 	vr := jwt.CreateValidationResults()
	// 	uc.Validate(vr)
	// 	if len(vr.Errors()) > 0 {
	// 		respondMsg(req, userNkey, serverID, "", "error validating claims")
	// 		return
	// 	}
	//
	// 	// Sign it with the issuer key since this is non-operator mode.
	// 	ejwt, err := uc.Encode(keyPairs.Issuer)
	// 	if err != nil {
	// 		respondMsg(req, userNkey, serverID, "", "error signing user JWT")
	// 		return
	// 	}
	//
	// 	fmt.Println("User claim is : ", ejwt)
	// 	respondMsg(req, userNkey, serverID, ejwt, "")
	// }

	// Create a service for auth callout with an endpoint binding to
	// the required subject. This allows for running multiple instances
	// to distribute the load, observe stats, and provide high availability.
	srv, err := micro.AddService(nc, micro.Config{
		Name:        "auth-callout",
		Version:     "0.0.1",
		Description: "Auth callout service.",
	})
	if err != nil {
		return err
	}

	g := srv.
		AddGroup("$SYS").
		AddGroup("REQ").
		AddGroup("USER")

		// Initialize dependencies
	userRepo := usersdebug.New() // Assuming you've added construct
	authHandler := authresponse.NewHandler(keyPairs, userRepo)

	err = g.AddEndpoint("AUTH", micro.HandlerFunc(authHandler.HandleRequest))
	if err != nil {
		return err
	}

	// Block and wait for interrupt.
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, os.Interrupt)
	<-sigch

	return nil
}
