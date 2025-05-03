// Package config handles application configuration loading and validation.
// It supports YAML configuration files with required field validation
// and provides safe defaults where appropriate.
package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config defines the structure for the application configuration.
type Config struct {
	Nats struct {
		URL  string `mapstructure:"url"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
	} `mapstructure:"nats"`

	Auth struct {
		IssuerSeed string `mapstructure:"issuer_seed"`
		XKeySeed   string `mapstructure:"xkey_seed"`
		UsersFile  string `mapstructure:"users_file"`
	} `mapstructure:"auth"`

	Environment string `mapstructure:"environment"`
}

// Load loads the configuration using viper, supporting YAML and environment variables.
func Load(configPath string) (*Config, error) {
	// Initialize viper
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Enable environment variable overrides without prefix
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config into struct: %w", err)
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

	log.Printf("Loaded config: %+v", cfg)
	return &cfg, nil
}

// MustLoad loads the configuration and panics on error.
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}
