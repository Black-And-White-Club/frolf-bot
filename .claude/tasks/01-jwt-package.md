# Task 01: JWT Package Creation

**Agent**: Sonnet
**Estimated Tokens**: ~8K input, ~4K output
**Dependencies**: None

---

## Objective

Create a new `pkg/jwt` package for JWT generation and validation used by magic link auth and NATS Auth Callout.

---

## Context

- Existing config pattern: `/Users/jace/Documents/GitHub/frolf-bot/config/config.go`
- Uses `github.com/golang-jwt/jwt/v5` (already in ecosystem)
- JWT must contain claims: `sub` (user ID), `guild`, `role`, `exp`, `iat`

---

## Implementation

### File Structure

```
pkg/
└── jwt/
    ├── jwt.go         # Core JWT service
    ├── claims.go      # Claims type definitions
    └── errors.go      # Typed errors
```

### Claims Definition (`claims.go`)

```go
type PWAClaims struct {
    jwt.RegisteredClaims
    Guild string `json:"guild"`
    Role  string `json:"role"`  // viewer | player | editor
}

type Role string

const (
    RoleViewer Role = "viewer"
    RolePlayer Role = "player"
    RoleEditor Role = "editor"
)
```

### Service Interface (`jwt.go`)

```go
type Service interface {
    GenerateToken(userID, guildID string, role Role, ttl time.Duration) (string, error)
    ValidateToken(tokenString string) (*PWAClaims, error)
    GenerateMagicLink(userID, guildID string, role Role) (string, error)
}
```

### Configuration Addition

Add to `config/config.go`:

```go
type JWTConfig struct {
    Secret      string `yaml:"secret" env:"JWT_SECRET"`
    DefaultTTL  time.Duration `yaml:"default_ttl" env:"JWT_DEFAULT_TTL" default:"24h"`
    PWABaseURL  string `yaml:"pwa_base_url" env:"PWA_BASE_URL" default:"https://pwa.frolf-bot.com"`
}
```

---

## Acceptance Criteria

- [ ] `PWAClaims` struct with `Guild` and `Role` fields
- [ ] `GenerateToken` signs with HS256 using `JWT_SECRET`
- [ ] `ValidateToken` verifies signature and expiry, returns parsed claims
- [ ] `GenerateMagicLink` returns full URL with token as `?t=` param
- [ ] Typed errors: `ErrInvalidToken`, `ErrExpiredToken`, `ErrInvalidSignature`
- [ ] Config struct added with environment variable binding

---

## Files to Read First

1. `/Users/jace/Documents/GitHub/frolf-bot/config/config.go` - Config patterns
2. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/user/infrastructure/repositories/models.go` - User/role patterns

---

## Do Not

- Do not create HTTP handlers (separate task)
- Do not integrate with NATS yet (separate task)
- Do not write tests (user will handle separately)
