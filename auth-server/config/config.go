// Package config handles application configuration loading and validation.
// It supports YAML configuration files with required field validation
// and provides safe defaults where appropriate.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration structure.
// It includes NATS connection details, authentication keys,
// and environment settings.
//
// Example YAML:
//
//	environment: production
//	nats:
//	  url: nats://localhost:4222
//	  user: auth
//	  pass: securepass
//	auth:
//	  issuer_seed: SAAG...
//	  xkey_seed: SXAK...
//	  users_file: path/to/users.json
type Config struct {
	Nats struct {
		URL  string `yaml:"url"`  // NATS server URL (e.g., "nats://localhost:4222")
		User string `yaml:"user"` // NATS username
		Pass string `yaml:"pass"` // NATS password
	} `yaml:"nats"`

	Auth struct {
		IssuerSeed string `yaml:"issuer_seed"` // Seed for issuer key pair (required)
		XKeySeed   string `yaml:"xkey_seed"`   // Seed for curve key pair (required)
		UsersFile  string `yaml:"users_file"`  // Path to users JSON file
	} `yaml:"auth"`

	Environment string `yaml:"environment"` // Runtime environment (development|production)
}

// Load reads and parses the configuration file from the given path.
// It validates required fields and sets default values where appropriate.
// Returns an error if the file cannot be read, parsed, or if required fields are missing.
//
// Example:
//
//	cfg, err := config.Load("config.yml")
//	if err != nil {
//	    log.Fatal(err)
//	}
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	// Explicit check for empty file
	if len(data) == 0 {
		return nil, fmt.Errorf("config file is empty")
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validation
	if cfg.Auth.IssuerSeed == "" {
		return nil, fmt.Errorf("auth.issuer_seed is required")
	}
	if cfg.Auth.XKeySeed == "" {
		return nil, fmt.Errorf("auth.xkey_seed is required")
	}
	if cfg.Environment == "" {
		cfg.Environment = "development" // Default value
	}

	return &cfg, nil
}

// MustLoad is similar to Load but panics if the configuration cannot be loaded.
// Suitable for use during application initialization where configuration errors
// should terminate the application.
//
// Example:
//
//	cfg := config.MustLoad("config.yml")
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}
