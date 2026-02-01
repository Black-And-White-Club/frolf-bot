# Task 06: Magic Link Generation Service

**Agent**: Haiku
**Estimated Tokens**: ~3K input, ~2K output
**Dependencies**: Task 01 (JWT Package), Task 05 (Config)

---

## Objective

Create a service method that generates magic links for the `/dashboard` Discord command. The Discord bot will call this via NATS request/reply.

---

## Context

The Discord bot (separate repo) sends a request when user runs `/dashboard`. This backend responds with a generated magic link.

---

## Implementation

### Request/Reply Subject

```
pwa.magic-link.request.v1
```

### Request Payload

```go
type MagicLinkRequest struct {
    UserID  string `json:"user_id"`
    GuildID string `json:"guild_id"`
    Role    string `json:"role"`  // Determined by Discord bot from user's permissions
}
```

### Response Payload

```go
type MagicLinkResponse struct {
    Success bool   `json:"success"`
    URL     string `json:"url,omitempty"`
    Error   string `json:"error,omitempty"`
}
```

---

## Handler Location

Add to user module or create dedicated `pwa` module:

### Option A: User Module Addition

File: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/user/infrastructure/handlers/`

```go
func (h *UserHandlers) HandleMagicLinkRequest(
    ctx context.Context,
    req *MagicLinkRequest,
) (*MagicLinkResponse, error) {
    url, err := h.jwtService.GenerateMagicLink(
        req.UserID,
        req.GuildID,
        jwt.Role(req.Role),
    )
    if err != nil {
        return &MagicLinkResponse{
            Success: false,
            Error:   err.Error(),
        }, nil
    }
    return &MagicLinkResponse{
        Success: true,
        URL:     url,
    }, nil
}
```

### Option B: New PWA Module

Create minimal module:

```
app/modules/pwa/
├── module.go
└── handlers.go
```

---

## Router Registration

```go
// Subscribe to magic link requests
nc.QueueSubscribe("pwa.magic-link.request.v1", "backend", func(msg *nats.Msg) {
    var req MagicLinkRequest
    json.Unmarshal(msg.Data, &req)

    resp, _ := handler.HandleMagicLinkRequest(ctx, &req)

    data, _ := json.Marshal(resp)
    msg.Respond(data)
})
```

---

## Acceptance Criteria

- [ ] Handler responds to `pwa.magic-link.request.v1`
- [ ] Uses JWT service to generate token
- [ ] Returns full magic link URL
- [ ] Error handling for invalid requests
- [ ] Logging for audit trail

---

## Files to Read First

1. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/user/module.go`
2. Task 01 output (JWT package)
