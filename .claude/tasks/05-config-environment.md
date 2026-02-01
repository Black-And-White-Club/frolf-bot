# Task 05: Configuration & Environment Updates

**Agent**: Haiku
**Estimated Tokens**: ~4K input, ~2K output
**Dependencies**: Task 01 (JWT Package)

---

## Objective

Update configuration to support PWA backend integration. Add JWT secrets, PWA URLs, and auth callout settings.

---

## Configuration Additions

### File: `/Users/jace/Documents/GitHub/frolf-bot/config/config.go`

Add new config sections:

```go
type Config struct {
    Postgres      PostgresConfig
    NATS          NATSConfig
    Observability ObservabilityConfig
    JWT           JWTConfig           // NEW
    PWA           PWAConfig           // NEW
    AuthCallout   AuthCalloutConfig   // NEW
}

type JWTConfig struct {
    Secret     string        `yaml:"secret" env:"JWT_SECRET"`
    DefaultTTL time.Duration `yaml:"default_ttl" env:"JWT_DEFAULT_TTL"`
}

type PWAConfig struct {
    BaseURL string `yaml:"base_url" env:"PWA_BASE_URL"`
}

type AuthCalloutConfig struct {
    Enabled       bool   `yaml:"enabled" env:"AUTH_CALLOUT_ENABLED"`
    Subject       string `yaml:"subject" env:"AUTH_CALLOUT_SUBJECT"`
    IssuerNKey    string `yaml:"issuer_nkey" env:"AUTH_CALLOUT_ISSUER_NKEY"`
    SigningNKey   string `yaml:"signing_nkey" env:"AUTH_CALLOUT_SIGNING_NKEY"`
}
```

---

## Default Values

```go
func DefaultConfig() *Config {
    return &Config{
        JWT: JWTConfig{
            DefaultTTL: 24 * time.Hour,
        },
        PWA: PWAConfig{
            BaseURL: "https://pwa.frolf-bot.com",
        },
        AuthCallout: AuthCalloutConfig{
            Enabled: false,
            Subject: "$SYS.REQ.USER.AUTH",
        },
    }
}
```

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `JWT_SECRET` | Shared secret for JWT signing/verification | Required |
| `JWT_DEFAULT_TTL` | Token expiration duration | `24h` |
| `PWA_BASE_URL` | Base URL for magic links | `https://pwa.frolf-bot.com` |
| `AUTH_CALLOUT_ENABLED` | Enable NATS auth callout handler | `false` |
| `AUTH_CALLOUT_SUBJECT` | NATS subject for auth requests | `$SYS.REQ.USER.AUTH` |
| `AUTH_CALLOUT_ISSUER_NKEY` | NKey for signing auth responses | Optional |
| `AUTH_CALLOUT_SIGNING_NKEY` | NKey seed for response signing | Optional |

---

## Validation

Add validation in config loading:

```go
func (c *Config) Validate() error {
    if c.AuthCallout.Enabled && c.JWT.Secret == "" {
        return errors.New("JWT_SECRET required when auth callout is enabled")
    }
    return nil
}
```

---

## YAML Example

Add to `config.yaml.example`:

```yaml
jwt:
  secret: ""  # Set via JWT_SECRET env var
  default_ttl: 24h

pwa:
  base_url: https://pwa.frolf-bot.com

auth_callout:
  enabled: false
  subject: "$SYS.REQ.USER.AUTH"
  issuer_nkey: ""
  signing_nkey: ""
```

---

## Acceptance Criteria

- [ ] JWTConfig struct added to Config
- [ ] PWAConfig struct added to Config
- [ ] AuthCalloutConfig struct added to Config
- [ ] Environment variable bindings work
- [ ] Default values set appropriately
- [ ] Validation for required fields when auth enabled
- [ ] Example YAML updated

---

## Files to Modify

1. `/Users/jace/Documents/GitHub/frolf-bot/config/config.go` - Main config
2. Create `/Users/jace/Documents/GitHub/frolf-bot/config.yaml.example` if not exists
