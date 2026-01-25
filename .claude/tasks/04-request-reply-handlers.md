# Task 04: Request/Reply Handlers for PWA Initial Load

**Agent**: Sonnet
**Estimated Tokens**: ~12K input, ~6K output
**Dependencies**: Task 03 (Event Subject Alignment)

---

## Objective

Add NATS request/reply handlers for PWA initial data loading. These provide snapshots when the PWA connects, before real-time events take over.

---

## Required Handlers

### 1. Round List Request

**Subject**: `round.list.request.v1.{guild_id}`

**Request**:
```json
{ "guild_id": "987654321" }
```

**Response**:
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

### 2. Leaderboard Snapshot Request

**Subject**: `leaderboard.snapshot.request.v1.{guild_id}`

**Request**:
```json
{ "guild_id": "987654321" }
```

**Response**:
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

## Implementation Location

Add to existing modules as new handler methods:

### Round Module

File: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/infrastructure/handlers/`

```go
// In handlers package
func (h *RoundHandlers) HandleRoundListRequest(
    ctx context.Context,
    req *RoundListRequest,
) (*RoundListResponse, error) {
    rounds, err := h.service.GetRoundsForGuild(ctx, req.GuildID)
    if err != nil {
        return nil, err
    }
    return &RoundListResponse{Rounds: mapToDTO(rounds)}, nil
}
```

### Leaderboard Module

File: `/Users/jace/Documents/GitHub/frolf-bot/app/modules/leaderboard/infrastructure/handlers/`

```go
func (h *LeaderboardHandlers) HandleSnapshotRequest(
    ctx context.Context,
    req *SnapshotRequest,
) (*SnapshotResponse, error) {
    lb, err := h.service.GetActiveLeaderboard(ctx, req.GuildID)
    if err != nil {
        return nil, err
    }
    return &SnapshotResponse{Leaderboard: mapToDTO(lb)}, nil
}
```

---

## Router Registration

Use NATS request/reply pattern via Watermill or raw NATS:

### Option A: Watermill Request/Reply (Preferred)

```go
// In router setup
router.AddHandler(
    "round-list-request-handler",
    "round.list.request.v1.>",  // Wildcard for all guilds
    subscriber,
    "round.list.response.v1",   // Reply topic (or use msg.Reply)
    publisher,
    h.handleRoundListRequest,
)
```

### Option B: Raw NATS Request/Reply

```go
// Using raw NATS subscription for request/reply
nc.QueueSubscribe("round.list.request.v1.*", "backend", func(msg *nats.Msg) {
    // Parse guild from subject
    // Call handler
    // msg.Respond(response)
})
```

---

## DTO Definitions

Create PWA-specific DTOs in a new package:

```
pkg/pwa/
├── dto.go          # Response DTOs
└── mappers.go      # Domain -> DTO mappers
```

### Round DTO

```go
type RoundDTO struct {
    ID           string         `json:"id"`
    Title        string         `json:"title"`
    State        string         `json:"state"`
    StartTime    *time.Time     `json:"startTime,omitempty"`
    Participants []ParticipantDTO `json:"participants"`
}

type ParticipantDTO struct {
    UserID      string  `json:"userId"`
    DisplayName string  `json:"displayName"`
    TagNumber   *int    `json:"tagNumber,omitempty"`
    Score       *int    `json:"score,omitempty"`
}
```

### Leaderboard DTO

```go
type LeaderboardEntryDTO struct {
    UserID      string `json:"userId"`
    DisplayName string `json:"displayName"`
    Tag         int    `json:"tag"`
    Points      int    `json:"points"`
    Movement    int    `json:"movement"`
}
```

---

## Service Layer Additions

### Round Service

Add to `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/application/interface.go`:

```go
// Add to RoundService interface
GetRoundsForGuild(ctx context.Context, guildID GuildID) ([]Round, error)
GetUpcomingRounds(ctx context.Context, guildID GuildID) ([]Round, error)
GetActiveRounds(ctx context.Context, guildID GuildID) ([]Round, error)
```

### Leaderboard Service

Add to `/Users/jace/Documents/GitHub/frolf-bot/app/modules/leaderboard/application/interface.go`:

```go
// Add to LeaderboardService interface
GetActiveLeaderboard(ctx context.Context, guildID GuildID) (*Leaderboard, error)
GetLeaderboardSnapshot(ctx context.Context, guildID GuildID) (*SnapshotResponse, error)
```

---

## Repository Additions

May need new query methods:

### Round Repository

```go
// Get rounds by guild with optional state filter
GetByGuildID(ctx context.Context, guildID GuildID, states ...RoundState) ([]Round, error)
```

### Leaderboard Repository

```go
// Get active leaderboard for guild
GetActiveByGuildID(ctx context.Context, guildID GuildID) (*Leaderboard, error)
```

---

## Acceptance Criteria

- [ ] Round list handler responds to `round.list.request.v1.{guild_id}`
- [ ] Leaderboard snapshot handler responds to `leaderboard.snapshot.request.v1.{guild_id}`
- [ ] Responses match specified JSON structure
- [ ] Handlers extract guild_id from subject suffix
- [ ] Service methods added with telemetry wrappers
- [ ] Repository methods added if needed
- [ ] DTOs in `pkg/pwa/` package

---

## Files to Read First

1. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/infrastructure/router/router.go`
2. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/application/service.go`
3. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/leaderboard/application/service.go`
4. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/infrastructure/repositories/interface.go`
