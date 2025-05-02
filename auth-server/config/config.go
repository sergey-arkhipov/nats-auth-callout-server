package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type NatsConfig struct {
	URL  string `yaml:"url"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

type AuthConfig struct {
	IssuerSeed string `yaml:"issuer_seed"`
	XKeySeed   string `yaml:"xkey_seed"`
	UsersFile  string `yaml:"users_file"`
}

type Config struct {
	Nats NatsConfig `yaml:"nats"`
	Auth AuthConfig `yaml:"auth"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults if needed
	if cfg.Nats.URL == "" {
		cfg.Nats.URL = "nats://localhost:4222"
	}

	return &cfg, nil
}

func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	return cfg
}
