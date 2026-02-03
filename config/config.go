package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	obs "github.com/Black-And-White-Club/frolf-bot-shared/observability"
)

// Config struct to hold the configuration settings
type Config struct {
	Postgres      PostgresConfig      `yaml:"postgres"`
	NATS          NATSConfig          `yaml:"nats"`
	JWT           JWTConfig           `yaml:"jwt"`
	PWA           PWAConfig           `yaml:"pwa"`
	AuthCallout   AuthCalloutConfig   `yaml:"auth_callout"`
	Observability ObservabilityConfig `yaml:"observability"`
	HTTP          HTTPConfig          `yaml:"http"`
}

type HTTPConfig struct {
	Port           string   `yaml:"port" env:"HTTP_PORT"`
	AllowedOrigins []string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
}

// PostgresConfig holds Postgres configuration.
type PostgresConfig struct {
	DSN string `yaml:"dsn"`
}

// NATSConfig holds NATS configuration.
type NATSConfig struct {
	URL string `yaml:"url"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret     string        `yaml:"secret" env:"JWT_SECRET"`
	DefaultTTL time.Duration `yaml:"default_ttl" env:"JWT_DEFAULT_TTL"`
}

// PWAConfig holds PWA configuration.
type PWAConfig struct {
	BaseURL string `yaml:"base_url" env:"PWA_BASE_URL"`
}

// AuthCalloutConfig holds NATS auth callout configuration.
type AuthCalloutConfig struct {
	Enabled     bool   `yaml:"enabled" env:"AUTH_CALLOUT_ENABLED"`
	Subject     string `yaml:"subject" env:"AUTH_CALLOUT_SUBJECT"`
	IssuerNKey  string `yaml:"issuer_nkey" env:"AUTH_CALLOUT_ISSUER_NKEY"`
	SigningNKey string `yaml:"signing_nkey" env:"AUTH_CALLOUT_SIGNING_NKEY"`
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
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_DEFAULT_TTL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.DefaultTTL = d
		}
	}
	if v := os.Getenv("PWA_BASE_URL"); v != "" {
		cfg.PWA.BaseURL = v
	}
	if v := os.Getenv("AUTH_CALLOUT_ENABLED"); v != "" {
		cfg.AuthCallout.Enabled = v == "true"
	}
	if v := os.Getenv("AUTH_CALLOUT_SUBJECT"); v != "" {
		cfg.AuthCallout.Subject = v
	}
	if v := os.Getenv("AUTH_CALLOUT_ISSUER_NKEY"); v != "" {
		cfg.AuthCallout.IssuerNKey = v
	}
	if v := os.Getenv("AUTH_CALLOUT_SIGNING_NKEY"); v != "" {
		cfg.AuthCallout.SigningNKey = v
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.HTTP.AllowedOrigins = strings.Split(v, ",")
	} else if len(cfg.HTTP.AllowedOrigins) == 0 && cfg.PWA.BaseURL != "" {
		cfg.HTTP.AllowedOrigins = []string{cfg.PWA.BaseURL}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
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

	// Load JWT settings
	cfg.JWT.Secret = os.Getenv("JWT_SECRET")
	jwtDefaultTTL := os.Getenv("JWT_DEFAULT_TTL")
	if jwtDefaultTTL == "" {
		cfg.JWT.DefaultTTL = 24 * time.Hour
	} else {
		var err error
		cfg.JWT.DefaultTTL, err = time.ParseDuration(jwtDefaultTTL)
		if err != nil {
			return nil, fmt.Errorf("invalid JWT_DEFAULT_TTL value: %v", err)
		}
	}
	pwaBaseURL := os.Getenv("PWA_BASE_URL")
	if pwaBaseURL == "" {
		cfg.PWA.BaseURL = "https://pwa.frolf-bot.com"
	} else {
		cfg.PWA.BaseURL = pwaBaseURL
	}

	// Load Auth Callout settings
	cfg.AuthCallout.Enabled = os.Getenv("AUTH_CALLOUT_ENABLED") == "true"
	authCalloutSubject := os.Getenv("AUTH_CALLOUT_SUBJECT")
	if authCalloutSubject == "" {
		cfg.AuthCallout.Subject = "$SYS.REQ.USER.AUTH"
	} else {
		cfg.AuthCallout.Subject = authCalloutSubject
	}
	cfg.AuthCallout.IssuerNKey = os.Getenv("AUTH_CALLOUT_ISSUER_NKEY")
	cfg.AuthCallout.SigningNKey = os.Getenv("AUTH_CALLOUT_SIGNING_NKEY")

	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.HTTP.AllowedOrigins = strings.Split(v, ",")
	} else if cfg.PWA.BaseURL != "" {
		cfg.HTTP.AllowedOrigins = []string{cfg.PWA.BaseURL}
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.AuthCallout.Enabled && c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET required when auth callout is enabled")
	}
	return nil
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
