// package Main create auth service and run
package main

import (
	"context"
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
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel) // Включаем Debug
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	// Configuration
	configFile := flag.String("config", "config.yml", "Path to config file")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Validation
	if cfg.Nats.URL == "" || cfg.Auth.IssuerSeed == "" {
		return fmt.Errorf("missing required configuration")
	}

	// Initialize auth
	keyPairs, err := authkeys.Parse(cfg.Auth.IssuerSeed, cfg.Auth.XKeySeed)
	if err != nil {
		return fmt.Errorf("parse auth keys: %w", err)
	}
	// NATS Connection
	nc, err := nats.Connect(
		cfg.Nats.URL,
		nats.UserInfo(cfg.Nats.User, cfg.Nats.Pass),
		nats.Name("auth-service"),
	)
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	defer func() {
		if err := nc.Drain(); err != nil {
			log.Printf("failed to drain NATS connection: %v", err)
		}
	}()

	// Microservice setup
	srv, err := micro.AddService(nc, micro.Config{
		Name:        "auth-callout",
		Version:     "0.0.1",
		Description: "Authentication service",
		Metadata: map[string]string{
			"env":    cfg.Environment,
			"region": "Russia", // Optional additional metadata},
		},
	})
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}

	// Endpoint setup
	userRepo, err := usersdebug.New()
	if err != nil {
		userRepo = usersdebug.FakeRepository
	}

	log.Print("Repo %w", userRepo)
	authHandler := authresponse.NewHandler(keyPairs, userRepo)

	err = srv.
		AddGroup("$SYS").
		AddGroup("REQ").
		AddGroup("USER").
		AddEndpoint("AUTH", micro.HandlerFunc(authHandler.HandleRequest))
	if err != nil {
		return fmt.Errorf("add endpoint: %w", err)
	}
	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	log.Printf("Service started, waiting for shutdown signal")
	<-ctx.Done()
	log.Printf("Shutting down")

	return nil
}
