# Backend Integration Requirements

> What your Go backend needs to support the PWA.

---

## 1. Magic Link Generation

### Endpoint or Command

When user runs `/dashboard` in Discord, your bot should:

1. Generate a JWT with these claims:
```json
{
  "sub": "user:123456789",    // Discord User ID
  "guild": "987654321",       // Guild ID where command was run
  "role": "editor",           // User's permission level
  "exp": 1706140800,          // Expiry (24h recommended)
  "iat": 1706054400
}
```

2. Sign with your secret key (same key Auth Callout uses to verify)

3. Send DM or ephemeral message with link:
```
https://pwa.frolf-bot.com/?t=<JWT>
```

### Example Go Code

```go
import "github.com/golang-jwt/jwt/v5"

type PWAClaims struct {
    jwt.RegisteredClaims
    Guild string `json:"guild"`
    Role  string `json:"role"`
}

func GenerateMagicLink(userID, guildID, role string) (string, error) {
    claims := PWAClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            Subject:   "user:" + userID,
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
        Guild: guildID,
        Role:  role,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
    if err != nil {
        return "", err
    }

    return fmt.Sprintf("https://pwa.frolf-bot.com/?t=%s", signed), nil
}
```

---

## 2. NATS Auth Callout

NATS calls your Auth Callout service when a client connects with a password (the JWT).

### NATS Server Config

```conf
authorization {
  auth_callout {
    issuer: "your-issuer-nkey"
    auth_users: ["auth-service"]
    account: "$G"
  }
}
```

### Auth Callout Handler

Your Go service should:

1. Receive the auth request from NATS
2. Extract and validate the JWT (signature, expiry)
3. Return permissions based on claims

```go
func HandleAuthCallout(req *nats.Msg) {
    // Parse the auth request
    var authReq AuthRequest
    json.Unmarshal(req.Data, &authReq)

    // The password field contains the JWT
    token := authReq.ConnectOpts.Password

    // Validate JWT
    claims, err := validateJWT(token)
    if err != nil {
        // Return deny
        respondDeny(req, "invalid token")
        return
    }

    // Build permissions based on guild
    guildID := claims.Guild
    userID := strings.TrimPrefix(claims.Subject, "user:")

    permissions := Permissions{
        Subscribe: PermissionSet{
            Allow: []string{
                fmt.Sprintf("round.*.%s", guildID),
                fmt.Sprintf("leaderboard.*.%s", guildID),
                fmt.Sprintf("user.*.%s", userID),
            },
        },
        Publish: PermissionSet{
            Allow: []string{
                // Read-only for now - no publish permissions
            },
        },
    }

    respondAllow(req, permissions)
}
```

### Permission Patterns

| Role | Subscribe | Publish |
|------|-----------|---------|
| viewer | `round.*.{guild}`, `leaderboard.*.{guild}` | None |
| player | + `score.*.{user}` | `round.participant.join.*` |
| editor | + all | `round.create.*`, `round.update.*` |

---

## 3. Event Publishers (Already Have?)

Your backend should publish these events when things change:

### Round Events
- `round.created.v1.{guild_id}` - New round created
- `round.updated.v1.{guild_id}` - Round details changed
- `round.started.v1.{guild_id}` - Round started
- `round.finalized.v1.{guild_id}` - Round completed
- `round.deleted.v1.{guild_id}` - Round removed

### Participant Events
- `round.participant.joined.v1.{guild_id}` - Player joined round
- `round.participant.score.updated.v1.{guild_id}` - Score changed

### Leaderboard Events
- `leaderboard.updated.v1.{guild_id}` - Full leaderboard refresh
- `leaderboard.tag.updated.v1.{guild_id}` - Single tag changed

---

## 4. Request/Reply Handlers (Optional)

PWA requests initial data on connect. If not implemented, it waits for events.

### Round List Request

Subject: `round.list.request.v1.{guild_id}`

Request:
```json
{ "guild_id": "987654321" }
```

Response:
```json
{
  "rounds": [
    {
      "id": "round-123",
      "title": "Weekly Tag",
      "state": "scheduled",
      "startTime": "2026-01-26T10:00:00Z",
      "participants": []
    }
  ]
}
```

### Leaderboard Snapshot Request

Subject: `leaderboard.snapshot.request.v1.{guild_id}`

Request:
```json
{ "guild_id": "987654321" }
```

Response:
```json
{
  "leaderboard": [
    {
      "userId": "123",
      "displayName": "Player One",
      "tag": 1,
      "points": 2450,
      "movement": 0
    }
  ]
}
```

---

## 5. Testing the Integration

### Step 1: Generate a Test Token

```bash
# Using your token generator
go run cmd/token-gen/main.go \
  --user-id=YOUR_DISCORD_ID \
  --guild-id=YOUR_GUILD_ID \
  --role=editor
```

### Step 2: Start PWA with Token

```bash
cd frolf-bot-pwa
bun run dev

# Open browser
open "http://localhost:5173/?t=YOUR_TOKEN"
```

### Step 3: Verify Connection

1. Check browser DevTools Console for `[frolf-pwa] Connected to NATS`
2. Check ConnectionStatus shows "Connected"
3. Trigger an event from Discord and watch PWA update

---

## Checklist

### Required
- [ ] JWT generation with `sub`, `guild`, `role`, `exp` claims
- [ ] Discord command to send magic link
- [ ] NATS Auth Callout validates JWT
- [ ] Auth Callout returns guild-scoped permissions
- [ ] Events publish to `*.v1.{guild_id}` subjects

### Optional (Recommended)
- [ ] Request/reply handler for round list
- [ ] Request/reply handler for leaderboard snapshot
- [ ] Guild info endpoint (name, icon) for PWA header

---

## Environment Variables

PWA expects:
```bash
VITE_NATS_URL=wss://nats.frolf-bot.com:443
```

Backend needs:
```bash
JWT_SECRET=your-shared-secret
NATS_URL=nats://nats.frolf-bot.com:4222
```
