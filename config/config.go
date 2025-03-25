package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// Config struct to hold the configuration settings
type Config struct {
	Postgres      PostgresConfig      `yaml:"postgres"`
	NATS          NATSConfig          `yaml:"nats"`
	Observability ObservabilityConfig `yaml:"observability"`
	// Discord     DiscordConfig      `yaml:"discord"`
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

// ObservabilityConfig holds configuration for observability components
type ObservabilityConfig struct {
	LokiURL         string  `yaml:"loki_url"`
	LokiTenantID    string  `yaml:"loki_tenant_id"`
	MetricsAddress  string  `yaml:"metrics_address"`
	TempoEndpoint   string  `yaml:"tempo_endpoint"`
	TempoInsecure   bool    `yaml:"tempo_insecure"`
	TempoSampleRate float64 `yaml:"tempo_sample_rate"`
	Environment     string  `yaml:"environment"`
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

	// Load Observability settings
	cfg.Observability.LokiURL = os.Getenv("LOKI_URL")
	cfg.Observability.LokiTenantID = os.Getenv("LOKI_TENANT_ID")
	cfg.Observability.MetricsAddress = os.Getenv("METRICS_ADDRESS")
	if cfg.Observability.MetricsAddress == "" {
		return nil, fmt.Errorf("METRICS_ADDRESS environment variable not set")
	}
	cfg.Observability.TempoEndpoint = os.Getenv("TEMPO_ENDPOINT")
	if cfg.Observability.TempoEndpoint == "" {
		return nil, fmt.Errorf("TEMPO_ENDPOINT environment variable not set")
	}
	cfg.Observability.Environment = os.Getenv("ENV")
	tempoInsecure := os.Getenv("TEMPO_INSECURE")
	if tempoInsecure == "" {
		cfg.Observability.TempoInsecure = false // Default value
	} else {
		cfg.Observability.TempoInsecure = tempoInsecure == "true"
	}

	tempoSampleRate := os.Getenv("TEMPO_SAMPLE_RATE")
	if tempoSampleRate == "" {
		cfg.Observability.TempoSampleRate = 0.1 // Default value
	} else {
		// Convert tempoSampleRate to float64
		var err error
		cfg.Observability.TempoSampleRate, err = strconv.ParseFloat(tempoSampleRate, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TEMPO_SAMPLE_RATE value: %v", err)
		}
	}

	// Load Discord token
	// cfg.Discord.Token = os.Getenv("DISCORD_TOKEN")
	// if cfg.Discord.Token == "" {
	// 	return nil, fmt.Errorf("DISCORD_TOKEN environment variable not set")
	// }

	return &cfg, nil
}
