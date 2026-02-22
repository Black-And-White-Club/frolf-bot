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
	RoundQueue    RoundQueueConfig    `yaml:"round_queue"`
	NATS          NATSConfig          `yaml:"nats"`
	JWT           JWTConfig           `yaml:"jwt"`
	PWA           PWAConfig           `yaml:"pwa"`
	AuthCallout   AuthCalloutConfig   `yaml:"auth_callout"`
	DiscordOAuth  DiscordOAuthConfig  `yaml:"discord_oauth"`
	GoogleOAuth   GoogleOAuthConfig   `yaml:"google_oauth"`
	Observability ObservabilityConfig `yaml:"observability"`
	HTTP          HTTPConfig          `yaml:"http"`
}

type HTTPConfig struct {
	Port              string   `yaml:"port" env:"HTTP_PORT"`
	AllowedOrigins    []string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
	TrustedProxyCIDRs []string `yaml:"trusted_proxy_cidrs" env:"TRUSTED_PROXY_CIDRS"`
}

// PostgresConfig holds Postgres configuration.
type PostgresConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// RoundQueueConfig holds River queue pool and worker settings.
type RoundQueueConfig struct {
	PoolMaxConns      int           `yaml:"pool_max_conns"`
	DefaultWorkers    int           `yaml:"default_workers"`
	RoundWorkers      int           `yaml:"round_workers"`
	FetchPollInterval time.Duration `yaml:"fetch_poll_interval"`
}

// NATSConfig holds NATS configuration.
type NATSConfig struct {
	URL string `yaml:"url"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret     string        `yaml:"secret" env:"JWT_SECRET"`
	DefaultTTL time.Duration `yaml:"default_ttl" env:"JWT_DEFAULT_TTL"`
	Issuer     string        `yaml:"issuer" env:"JWT_ISSUER"`
	Audience   string        `yaml:"audience" env:"JWT_AUDIENCE"`
}

var weakJWTSecrets = map[string]struct{}{
	"":                {},
	"changeme":        {},
	"change-me":       {},
	"default":         {},
	"secret":          {},
	"password":        {},
	"jwt-secret":      {},
	"jwt_secret":      {},
	"your-secret":     {},
	"your-jwt-secret": {},
}

const (
	defaultPostgresMaxOpenConns      = 25
	defaultPostgresMaxIdleConns      = 25
	defaultPostgresConnMaxLifetime   = 5 * time.Minute
	defaultRoundQueueDefaultWorkers  = 50
	defaultRoundQueueRoundWorkers    = 25
	defaultRoundQueuePoolMaxConnsMin = 2
)

// PWAConfig holds PWA configuration.
type PWAConfig struct {
	BaseURL string `yaml:"base_url" env:"PWA_BASE_URL"`
}

// DiscordOAuthConfig holds Discord OAuth2 application credentials.
// Leave ClientID/ClientSecret empty to disable Discord OAuth login.
type DiscordOAuthConfig struct {
	ClientID     string `yaml:"client_id"     env:"DISCORD_OAUTH_CLIENT_ID"`
	ClientSecret string `yaml:"client_secret" env:"DISCORD_OAUTH_CLIENT_SECRET"`
	RedirectURL  string `yaml:"redirect_url"  env:"DISCORD_OAUTH_REDIRECT_URL"`
}

// GoogleOAuthConfig holds Google OAuth2 application credentials.
// Leave ClientID/ClientSecret empty to disable Google OAuth login.
type GoogleOAuthConfig struct {
	ClientID     string `yaml:"client_id"     env:"GOOGLE_OAUTH_CLIENT_ID"`
	ClientSecret string `yaml:"client_secret" env:"GOOGLE_OAUTH_CLIENT_SECRET"`
	RedirectURL  string `yaml:"redirect_url"  env:"GOOGLE_OAUTH_REDIRECT_URL"`
}

// AuthCalloutConfig holds NATS auth callout configuration.
type AuthCalloutConfig struct {
	Enabled         bool   `yaml:"enabled" env:"AUTH_CALLOUT_ENABLED"`
	Subject         string `yaml:"subject" env:"AUTH_CALLOUT_SUBJECT"`
	IssuerNKey      string `yaml:"issuer_nkey" env:"AUTH_CALLOUT_ISSUER_NKEY"`
	SigningNKey     string `yaml:"signing_nkey" env:"AUTH_CALLOUT_SIGNING_NKEY"`
	ServerPublicKey string `yaml:"server_public_key" env:"AUTH_CALLOUT_SERVER_PUBLIC_KEY"` // NATS server's public NKey (e.g. "NA...")
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
	if v := os.Getenv("POSTGRES_MAX_OPEN_CONNS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Postgres.MaxOpenConns = parsed
		}
	}
	if v := os.Getenv("POSTGRES_MAX_IDLE_CONNS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.Postgres.MaxIdleConns = parsed
		}
	}
	if v := os.Getenv("POSTGRES_CONN_MAX_LIFETIME"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			cfg.Postgres.ConnMaxLifetime = parsed
		}
	}
	if v := os.Getenv("ROUND_QUEUE_POOL_MAX_CONNS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.RoundQueue.PoolMaxConns = parsed
		}
	}
	if v := os.Getenv("ROUND_QUEUE_DEFAULT_WORKERS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.RoundQueue.DefaultWorkers = parsed
		}
	}
	if v := os.Getenv("ROUND_QUEUE_ROUND_WORKERS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			cfg.RoundQueue.RoundWorkers = parsed
		}
	}
	if v := os.Getenv("ROUND_QUEUE_FETCH_POLL_INTERVAL"); v != "" {
		if parsed, err := time.ParseDuration(v); err == nil {
			cfg.RoundQueue.FetchPollInterval = parsed
		}
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
	if v := os.Getenv("HTTP_PORT"); v != "" {
		cfg.HTTP.Port = v
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
	if cfg.PWA.BaseURL == "" {
		cfg.PWA.BaseURL = "https://pwa.frolf-bot.com"
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
	if v := os.Getenv("AUTH_CALLOUT_SERVER_PUBLIC_KEY"); v != "" {
		cfg.AuthCallout.ServerPublicKey = v
	}
	if v := os.Getenv("DISCORD_OAUTH_CLIENT_ID"); v != "" {
		cfg.DiscordOAuth.ClientID = v
	}
	if v := os.Getenv("DISCORD_OAUTH_CLIENT_SECRET"); v != "" {
		cfg.DiscordOAuth.ClientSecret = v
	}
	if v := os.Getenv("DISCORD_OAUTH_REDIRECT_URL"); v != "" {
		cfg.DiscordOAuth.RedirectURL = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_ID"); v != "" {
		cfg.GoogleOAuth.ClientID = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET"); v != "" {
		cfg.GoogleOAuth.ClientSecret = v
	}
	if v := os.Getenv("GOOGLE_OAUTH_REDIRECT_URL"); v != "" {
		cfg.GoogleOAuth.RedirectURL = v
	}
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.HTTP.AllowedOrigins = strings.Split(v, ",")
	} else if len(cfg.HTTP.AllowedOrigins) == 0 && cfg.PWA.BaseURL != "" {
		cfg.HTTP.AllowedOrigins = []string{cfg.PWA.BaseURL}
	}
	if v := os.Getenv("TRUSTED_PROXY_CIDRS"); v != "" {
		cfg.HTTP.TrustedProxyCIDRs = strings.Split(v, ",")
	}

	applyDefaults(&cfg)

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
	if v := os.Getenv("POSTGRES_MAX_OPEN_CONNS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid POSTGRES_MAX_OPEN_CONNS value: %v", err)
		}
		cfg.Postgres.MaxOpenConns = parsed
	}
	if v := os.Getenv("POSTGRES_MAX_IDLE_CONNS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid POSTGRES_MAX_IDLE_CONNS value: %v", err)
		}
		cfg.Postgres.MaxIdleConns = parsed
	}
	if v := os.Getenv("POSTGRES_CONN_MAX_LIFETIME"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid POSTGRES_CONN_MAX_LIFETIME value: %v", err)
		}
		cfg.Postgres.ConnMaxLifetime = parsed
	}
	if v := os.Getenv("ROUND_QUEUE_POOL_MAX_CONNS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ROUND_QUEUE_POOL_MAX_CONNS value: %v", err)
		}
		cfg.RoundQueue.PoolMaxConns = parsed
	}
	if v := os.Getenv("ROUND_QUEUE_DEFAULT_WORKERS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ROUND_QUEUE_DEFAULT_WORKERS value: %v", err)
		}
		cfg.RoundQueue.DefaultWorkers = parsed
	}
	if v := os.Getenv("ROUND_QUEUE_ROUND_WORKERS"); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ROUND_QUEUE_ROUND_WORKERS value: %v", err)
		}
		cfg.RoundQueue.RoundWorkers = parsed
	}
	if v := os.Getenv("ROUND_QUEUE_FETCH_POLL_INTERVAL"); v != "" {
		parsed, err := time.ParseDuration(v)
		if err != nil {
			return nil, fmt.Errorf("invalid ROUND_QUEUE_FETCH_POLL_INTERVAL value: %v", err)
		}
		cfg.RoundQueue.FetchPollInterval = parsed
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
	cfg.AuthCallout.ServerPublicKey = os.Getenv("AUTH_CALLOUT_SERVER_PUBLIC_KEY")

	cfg.DiscordOAuth.ClientID = os.Getenv("DISCORD_OAUTH_CLIENT_ID")
	cfg.DiscordOAuth.ClientSecret = os.Getenv("DISCORD_OAUTH_CLIENT_SECRET")
	cfg.DiscordOAuth.RedirectURL = os.Getenv("DISCORD_OAUTH_REDIRECT_URL")

	cfg.GoogleOAuth.ClientID = os.Getenv("GOOGLE_OAUTH_CLIENT_ID")
	cfg.GoogleOAuth.ClientSecret = os.Getenv("GOOGLE_OAUTH_CLIENT_SECRET")
	cfg.GoogleOAuth.RedirectURL = os.Getenv("GOOGLE_OAUTH_REDIRECT_URL")

	cfg.HTTP.Port = os.Getenv("HTTP_PORT")
	if v := os.Getenv("CORS_ALLOWED_ORIGINS"); v != "" {
		cfg.HTTP.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("TRUSTED_PROXY_CIDRS"); v != "" {
		cfg.HTTP.TrustedProxyCIDRs = strings.Split(v, ",")
	}

	applyDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

var defaultTrustedProxyCIDRs = []string{
	"127.0.0.1/32",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
}

func applyDefaults(cfg *Config) {
	if len(cfg.HTTP.TrustedProxyCIDRs) == 0 {
		cfg.HTTP.TrustedProxyCIDRs = defaultTrustedProxyCIDRs
	}
	if cfg.Postgres.MaxOpenConns <= 0 {
		cfg.Postgres.MaxOpenConns = defaultPostgresMaxOpenConns
	}
	if cfg.Postgres.MaxIdleConns <= 0 {
		cfg.Postgres.MaxIdleConns = defaultPostgresMaxIdleConns
	}
	if cfg.Postgres.ConnMaxLifetime <= 0 {
		cfg.Postgres.ConnMaxLifetime = defaultPostgresConnMaxLifetime
	}

	if cfg.RoundQueue.DefaultWorkers <= 0 {
		cfg.RoundQueue.DefaultWorkers = defaultRoundQueueDefaultWorkers
	}
	if cfg.RoundQueue.RoundWorkers <= 0 {
		cfg.RoundQueue.RoundWorkers = defaultRoundQueueRoundWorkers
	}
	if cfg.RoundQueue.PoolMaxConns <= 0 {
		derived := cfg.Postgres.MaxOpenConns / 2
		if derived < defaultRoundQueuePoolMaxConnsMin {
			derived = defaultRoundQueuePoolMaxConnsMin
		}
		cfg.RoundQueue.PoolMaxConns = derived
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	secret := strings.TrimSpace(c.JWT.Secret)
	if _, weak := weakJWTSecrets[strings.ToLower(secret)]; weak {
		return fmt.Errorf("JWT_SECRET must be set to a strong value and must not use a known default")
	}
	if len(secret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if c.AuthCallout.Enabled && secret == "" {
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
		OTLPInsecure:    appCfg.Observability.TempoInsecure,
		LogsEnabled:     appCfg.Observability.OTLPLogsEnabled,
	}
}
