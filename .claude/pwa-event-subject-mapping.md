# PWA Event Subject Mapping

## Executive Summary

This document maps current event subjects to PWA-required subjects with `{guild_id}` suffixes for permission scoping.

**Status**: ✅ Implementation Complete
**Approach**: Option B - Dynamic guild suffix via dual publishing (maintains backward compatibility)

### Implementation Summary

**Files Modified**: 9 handlers + 2 helper files created

**Key Changes**:
- Created `pkg/eventbus/guild_scoped.go` with helper functions
- Added `addGuildScopedResult()` helper to round and leaderboard handler packages
- Modified 8 event handlers to publish dual events (original + guild-scoped)
- Zero breaking changes - all existing consumers continue to work

**Events Now Publishing Guild-Scoped Versions**:
- ✅ round.created.v1.{guild_id}
- ✅ round.updated.v1.{guild_id}
- ✅ round.finalized.v1.{guild_id}
- ✅ round.deleted.v1.{guild_id}
- ✅ round.participant.joined.v1.{guild_id}
- ✅ round.participant.score.updated.v1.{guild_id}
- ✅ leaderboard.updated.v1.{guild_id}

**Deferred Events** (not currently needed for PWA):
- ⚠️ round.started.v1 - Backend event not published (Discord variant used)
- ❌ leaderboard.tag.updated.v1 - Event defined but never published

---

## Required PWA Event Patterns

### Round Events
| Current Subject | Required Subject | Status | Publisher Location |
|-----------------|------------------|--------|-------------------|
| `round.created.v1` | `round.created.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/create_round.go:42` |
| `round.updated.v1` | `round.updated.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/update_round.go:121` |
| `round.started.v1` | `round.started.v1.{guild_id}` | ⚠️ **DEFERRED** | Event defined but unused (Discord variant used instead) |
| `round.finalized.v1` | `round.finalized.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/finalize_round.go:78` |
| `round.deleted.v1` | `round.deleted.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/delete_round.go:63` |

### Participant Events
| Current Subject | Required Subject | Status | Publisher Location |
|-----------------|------------------|--------|-------------------|
| `round.participant.joined.v1` | `round.participant.joined.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/participant_status.go:163,280` |
| `round.participant.score.updated.v1` | `round.participant.score.updated.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/round/infrastructure/handlers/score_round.go:44` & `import_handlers.go:113` |

### Leaderboard Events
| Current Subject | Required Subject | Status | Publisher Location |
|-----------------|------------------|--------|-------------------|
| `leaderboard.updated.v1` | `leaderboard.updated.v1.{guild_id}` | ✅ **IMPLEMENTED** | `app/modules/leaderboard/infrastructure/handlers/leaderboard_update.go:86` |
| `leaderboard.tag.updated.v1` | `leaderboard.tag.updated.v1.{guild_id}` | ❌ **OUT OF SCOPE** | Event defined but not currently published |

---

## Implementation Approach

### Selected: Option B - Dual Publishing Pattern

**Strategy**: Publish both original and guild-scoped versions of events during migration period.

**Benefits**:
- ✅ Maintains backward compatibility with existing consumers
- ✅ Enables PWA to subscribe with guild-specific patterns
- ✅ Allows gradual migration of internal consumers to wildcard patterns
- ✅ No breaking changes to existing functionality

**Implementation**:
1. Create `pkg/eventbus/guild_scoped.go` helper functions
2. Modify handlers to publish additional guild-scoped Result
3. No changes required to router subscriptions (continue using base topics)
4. PWA subscribes to `{topic}.{guild_id}` or `{topic}.*` patterns

### Alternative Options Considered

**Option A**: Modify shared package constants to include placeholder
- ❌ Rejected: Breaking change, affects all services

**Option C**: Create PWA-specific event republisher
- ❌ Rejected: Adds unnecessary complexity and message duplication overhead

---

## Backward Compatibility Strategy

### Current Consumers
Internal handlers subscribe to base topics without wildcards:
- `round.created.v1` ← Round module
- `leaderboard.updated.v1` ← Leaderboard module

### Migration Path
1. **Phase 1** (This PR): Dual publishing - original + guild-scoped
   - Internal consumers: Continue using base topics ✅
   - PWA consumers: Use guild-scoped topics ✅

2. **Phase 2** (Future): Migrate to guild-scoped only
   - Update internal consumers to use wildcard subscriptions
   - Deprecate original base topics
   - Publish only guild-scoped versions

---

## Event Payload Analysis

All identified events already contain `GuildID` field in their payloads:

| Event | Payload Type | GuildID Field |
|-------|-------------|---------------|
| RoundCreatedV1 | `RoundEntityCreatedPayloadV1` | ✅ `payload.GuildID` |
| RoundUpdatedV1 | `RoundEntityUpdatedPayloadV1` | ✅ `payload.GuildID` or `payload.Round.GuildID` |
| RoundFinalizedV1 | `RoundFinalizedPayloadV1` | ✅ `payload.GuildID` |
| RoundDeletedV1 | `RoundDeletedPayloadV1` | ✅ `payload.GuildID` |
| RoundParticipantJoinedV1 | `RoundParticipantJoinedPayloadV1` | ✅ `payload.GuildID` |
| RoundParticipantScoreUpdatedV1 | `RoundParticipantScoreUpdatedPayloadV1` | ✅ `payload.GuildID` |
| LeaderboardUpdatedV1 | `LeaderboardUpdatedPayloadV1` | ✅ `payload.GuildID` |

---

## Implementation Checklist

- [x] Create `pkg/eventbus/guild_scoped.go` helper package
- [x] Update `create_round.go` to publish guild-scoped `RoundCreatedV1`
- [x] Update `update_round.go` to publish guild-scoped `RoundUpdatedV1`
- [x] Update `finalize_round.go` to publish guild-scoped `RoundFinalizedV1`
- [x] Update `delete_round.go` to publish guild-scoped `RoundDeletedV1`
- [x] Update `participant_status.go` to publish guild-scoped `RoundParticipantJoinedV1`
- [x] Update `score_round.go` to publish guild-scoped `RoundParticipantScoreUpdatedV1`
- [x] Update `import_handlers.go` to publish guild-scoped `RoundParticipantScoreUpdatedV1`
- [x] Update `leaderboard_update.go` to publish guild-scoped `LeaderboardUpdatedV1`
- [x] Test internal consumers still function (no wildcard changes needed)
- [x] Document PWA subscription patterns

---

## Notes

### RoundStartedV1 Status
- Event is defined in shared package but not currently published
- Current implementation publishes `RoundStartedDiscordV1` instead
- **Recommendation**: Add RoundStartedV1 publication in future ticket when PWA needs it
- **Decision**: Mark as out of scope for this task

### LeaderboardTagUpdatedV1 Status
- Event is defined in shared package but not currently published
- No handlers currently emit this event
- **Recommendation**: Add when batch tag assignment or tag swap operations need PWA notification
- **Decision**: Mark as out of scope for this task

---

## Subscription Patterns for PWA

PWA consumers should use one of these patterns:

```go
// Subscribe to specific guild
nats.Subscribe("round.created.v1.123456789", handler)

// Subscribe to all guilds
nats.Subscribe("round.created.v1.*", handler)

// Subscribe with NATS subject filtering (if using JetStream)
nats.Subscribe("round.*.v1.123456789", handler) // All round events for one guild
```

---

## Testing Strategy

1. **Unit Tests**: Verify guild-scoped topics are formatted correctly
2. **Integration Tests**: Verify both original and guild-scoped events are published
3. **Backward Compatibility**: Verify existing consumers continue to receive original events
4. **PWA Integration**: Verify PWA can subscribe to guild-scoped events

