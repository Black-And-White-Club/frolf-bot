package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config struct to hold the configuration settings
type Config struct {
	Postgres PostgresConfig `yaml:"postgres"`
	NATS     NATSConfig     `yaml:"nats"`
	// Discord  DiscordConfig  `yaml:"discord"`
	// ... other configuration fields ...
}

// PostgresConfig holds Postgres configuration.
type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

// NATSConfig holds NATS configuration.
type NATSConfig struct {
	URL string `yaml:"url"`
}

// DiscordConfig holds Discord configuration.
// type DiscordConfig struct {
// 	Token string `yaml:"token"`
// }

// LoadConfig loads the configuration from a YAML file.
func LoadConfig(filename string) (*Config, error) {
	// Try reading configuration from the file first
	data, err := os.ReadFile(filename)
	if err != nil {
		// If the file is not found, try loading from environment variables
		fmt.Printf("Failed to read config file: %v\n", err)
		fmt.Println("Trying to load configuration from environment variables...")

		return loadConfigFromEnv()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// loadConfigFromEnv loads the configuration from environment variables.
func loadConfigFromEnv() (*Config, error) {
	var cfg Config

	// Load Postgres DSN
	cfg.Postgres.DSN = os.Getenv("DATABASE_URL")
	if cfg.Postgres.DSN == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	// Load NATS URL
	cfg.NATS.URL = os.Getenv("NATS_URL")
	if cfg.NATS.URL == "" {
		return nil, fmt.Errorf("NATS_URL environment variable not set")
	}

	// Load Discord token
	// cfg.Discord.Token = os.Getenv("DISCORD_TOKEN")
	// if cfg.Discord.Token == "" {
	// 	return nil, fmt.Errorf("DISCORD_TOKEN environment variable not set")
	// }

	return &cfg, nil
}
