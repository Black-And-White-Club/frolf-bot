# Task 02: NATS Auth Callout Module

**Agent**: Opus
**Estimated Tokens**: ~15K input, ~8K output
**Dependencies**: Task 01 (JWT Package)

---

## Objective

Create a new `app/modules/authcallout` module that handles NATS Auth Callout requests, validates JWTs, and returns guild-scoped permissions.

---

## Context

- Module pattern: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/module.go`
- Service pattern: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/application/service.go`
- Router pattern: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/infrastructure/router/router.go`
- NATS Auth Callout sends requests to a designated subject when clients connect

---

## NATS Auth Callout Protocol

When a client connects with a password (the JWT), NATS sends an auth request to the auth callout service. The service must respond with either:

1. **Allow** - With pub/sub permissions
2. **Deny** - With error reason

---

## Implementation

### File Structure

```
app/modules/authcallout/
├── module.go                           # Module initialization
├── application/
│   ├── interface.go                    # Service interface
│   └── service.go                      # Auth validation logic
└── infrastructure/
    ├── handlers/
    │   └── auth_handler.go             # NATS message handler
    └── permissions/
        └── builder.go                  # Permission set builder
```

### Permission Matrix

| Role | Subscribe Patterns | Publish Patterns |
|------|-------------------|------------------|
| `viewer` | `round.*.{guild}`, `leaderboard.*.{guild}` | None |
| `player` | + `score.*.{user}` | `round.participant.join.*` |
| `editor` | All guild patterns | `round.create.*`, `round.update.*` |

### Service Interface (`application/interface.go`)

```go
type AuthCalloutService interface {
    HandleAuthRequest(ctx context.Context, req *AuthRequest) (*AuthResponse, error)
}

type AuthRequest struct {
    ConnectOpts ConnectOptions `json:"connect_opts"`
}

type ConnectOptions struct {
    Password string `json:"pass"` // Contains the JWT
}

type AuthResponse struct {
    Allowed     bool        `json:"allowed"`
    Permissions *Permissions `json:"permissions,omitempty"`
    Error       string      `json:"error,omitempty"`
}

type Permissions struct {
    Subscribe PermissionSet `json:"subscribe"`
    Publish   PermissionSet `json:"publish"`
}

type PermissionSet struct {
    Allow []string `json:"allow"`
    Deny  []string `json:"deny,omitempty"`
}
```

### Permission Builder (`infrastructure/permissions/builder.go`)

```go
type Builder struct{}

func (b *Builder) ForRole(role jwt.Role, guildID, userID string) *Permissions {
    // Build permission sets based on role
}
```

### Handler Pattern

Follow existing handler wrapper pattern from round module:

```go
func (h *AuthHandler) HandleAuthCallout(ctx context.Context, msg *nats.Msg) {
    // 1. Parse auth request
    // 2. Extract JWT from password field
    // 3. Call service.HandleAuthRequest
    // 4. Respond with auth response
}
```

---

## Module Registration

Add to `/Users/jace/Documents/GitHub/frolf-bot/app/app.go`:

```go
// In Initialize or Run
authCalloutModule, err := authcallout.NewModule(
    ctx,
    app.Config,
    obs,
    jwtService, // From Task 01
    app.EventBus,
    routerCtx,
)
```

---

## NATS Subscription Subject

Subscribe to the auth callout subject configured in NATS server:

```
$SYS.REQ.USER.AUTH
```

Or custom subject if using account-based auth callout.

---

## Acceptance Criteria

- [ ] Module follows existing module pattern with `NewModule`, `Run`, `Close`
- [ ] Service validates JWT using pkg/jwt
- [ ] Permission builder generates correct patterns per role
- [ ] Handler subscribes to auth callout subject
- [ ] Handler responds with signed auth response (if using NKey signing)
- [ ] Proper error responses for invalid/expired tokens
- [ ] Telemetry integration (tracing, metrics, logging)

---

## Files to Read First

1. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/module.go` - Module pattern
2. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/application/service.go` - Service pattern
3. `/Users/jace/Documents/GitHub/frolf-bot/app/app.go` - Module registration

---

## Implementation Notes

- NATS Auth Callout may require NKey signing for responses in production
- Consider adding a separate config section for auth callout settings
- The auth callout service runs as a special NATS user with `auth_users` permission
