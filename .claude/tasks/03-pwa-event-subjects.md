# Task 03: PWA Event Subject Alignment

**Agent**: Sonnet
**Estimated Tokens**: ~10K input, ~3K output
**Dependencies**: None (can run parallel with Task 01)

---

## Objective

Audit and align existing event subjects with the PWA-required subject patterns. Ensure all events include `{guild_id}` suffix for permission scoping.

---

## Required Subject Patterns

### Round Events
- `round.created.v1.{guild_id}`
- `round.updated.v1.{guild_id}`
- `round.started.v1.{guild_id}`
- `round.finalized.v1.{guild_id}`
- `round.deleted.v1.{guild_id}`

### Participant Events
- `round.participant.joined.v1.{guild_id}`
- `round.participant.score.updated.v1.{guild_id}`

### Leaderboard Events
- `leaderboard.updated.v1.{guild_id}`
- `leaderboard.tag.updated.v1.{guild_id}`

---

## Audit Steps

1. **Search existing event topics** in `frolf-bot-shared` package
2. **Map current topics** to required PWA patterns
3. **Identify gaps** where guild_id suffix is missing
4. **Document changes** needed in shared package

---

## Files to Audit

1. Event topic definitions in shared package
2. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/round/infrastructure/router/router.go` - Topic subscriptions
3. `/Users/jace/Documents/GitHub/frolf-bot/app/modules/leaderboard/infrastructure/router/router.go` - Topic subscriptions
4. Handler files that publish events

---

## Output

Create a mapping document showing:

| Current Subject | Required Subject | Action |
|-----------------|------------------|--------|
| `round.created.v1` | `round.created.v1.{guild_id}` | Add guild suffix |
| ... | ... | ... |

---

## Implementation Approach

If subjects need modification:

1. **Option A**: Modify shared package event constants to include guild_id placeholder
2. **Option B**: Add publishing wrapper that appends guild_id dynamically
3. **Option C**: Create PWA-specific event republisher

Recommend **Option B** - Dynamic suffix at publish time:

```go
func PublishWithGuildScope(bus EventBus, baseTopic string, guildID string, payload any) error {
    topic := fmt.Sprintf("%s.%s", baseTopic, guildID)
    return bus.Publish(topic, payload)
}
```

---

## Acceptance Criteria

- [ ] All round events include guild_id suffix
- [ ] All leaderboard events include guild_id suffix
- [ ] All participant events include guild_id suffix
- [ ] Existing consumers continue to work (wildcards or migration)
- [ ] Document any shared package changes needed

---

## Compatibility Consideration

Existing consumers may subscribe with wildcards:
- `round.created.v1.*` catches all guilds
- `round.created.v1.{specific_guild}` catches one guild

This is the intended pattern for PWA permission scoping.
