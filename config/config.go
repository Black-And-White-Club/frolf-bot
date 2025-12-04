package config

import (
	"fmt"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"

	obs "github.com/Black-And-White-Club/frolf-bot-shared/observability"
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
	OTLPEndpoint    string  `yaml:"otlp_endpoint"`
	OTLPTransport   string  `yaml:"otlp_transport"` // grpc|http
	OTLPLogsEnabled bool    `yaml:"otlp_logs_enabled"`
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
		return loadConfigFromEnv()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// --- OVERRIDE WITH ENV VARS IF PRESENT ---
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Postgres.DSN = v
	}
	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("LOKI_URL"); v != "" {
		cfg.Observability.LokiURL = v
	}
	if v := os.Getenv("LOKI_TENANT_ID"); v != "" {
		cfg.Observability.LokiTenantID = v
	}
	if v := os.Getenv("METRICS_ADDRESS"); v != "" {
		cfg.Observability.MetricsAddress = v
	}
	if v := os.Getenv("TEMPO_ENDPOINT"); v != "" {
		cfg.Observability.TempoEndpoint = v
	}
	if v := os.Getenv("OTLP_ENDPOINT"); v != "" {
		cfg.Observability.OTLPEndpoint = v
	}
	if v := os.Getenv("OTLP_TRANSPORT"); v != "" {
		cfg.Observability.OTLPTransport = v
	}
	if v := os.Getenv("OTLP_LOGS_ENABLED"); v != "" {
		cfg.Observability.OTLPLogsEnabled = v == "true"
	}
	if v := os.Getenv("ENV"); v != "" {
		cfg.Observability.Environment = v
	}
	if v := os.Getenv("TEMPO_INSECURE"); v != "" {
		cfg.Observability.TempoInsecure = v == "true"
	}
	if v := os.Getenv("TEMPO_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Observability.TempoSampleRate = f
		}
	}
	if v := os.Getenv("OTLP_LOGS_ENABLED"); v != "" {
		cfg.Observability.OTLPLogsEnabled = v == "true"
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
	cfg.Observability.MetricsAddress = os.Getenv("METRICS_ADDRESS") // optional; empty disables metrics
	cfg.Observability.TempoEndpoint = os.Getenv("TEMPO_ENDPOINT")   // optional; empty disables tracing
	cfg.Observability.OTLPEndpoint = os.Getenv("OTLP_ENDPOINT")     // optional; shared collector endpoint
	cfg.Observability.OTLPTransport = os.Getenv("OTLP_TRANSPORT")   // optional; default set later
	cfg.Observability.OTLPLogsEnabled = os.Getenv("OTLP_LOGS_ENABLED") == "true"
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

func ToObsConfig(appCfg *Config) obs.Config {
	return obs.Config{
		ServiceName:     "frolf-bot", // Or dynamic from app build
		Environment:     appCfg.Observability.Environment,
		Version:         "1.2.3", // Could inject via `ldflags`
		LokiURL:         appCfg.Observability.LokiURL,
		MetricsAddress:  appCfg.Observability.MetricsAddress,
		TempoEndpoint:   appCfg.Observability.TempoEndpoint,
		TempoInsecure:   appCfg.Observability.TempoInsecure,
		TempoSampleRate: appCfg.Observability.TempoSampleRate,
		OTLPEndpoint:    appCfg.Observability.OTLPEndpoint,
		OTLPTransport:   appCfg.Observability.OTLPTransport,
		LogsEnabled:     appCfg.Observability.OTLPLogsEnabled,
	}
}
