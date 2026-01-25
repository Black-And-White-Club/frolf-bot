package eventbus

import (
	"fmt"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/ThreeDotsLabs/watermill/message"
)

// PublishWithGuildScope publishes an event with a guild_id suffix for PWA permission scoping.
// It appends the guild_id to the base topic using the pattern: {baseTopic}.{guildID}
//
// Example:
//   - baseTopic: "round.created.v1"
//   - guildID: "123456789"
//   - result: "round.created.v1.123456789"
//
// This allows PWA consumers to subscribe with wildcards:
//   - "round.created.v1.*" catches all guilds
//   - "round.created.v1.123456789" catches one specific guild
func PublishWithGuildScope(bus eventbus.EventBus, baseTopic string, guildID string, msg *message.Message) error {
	if guildID == "" {
		return fmt.Errorf("guildID cannot be empty for guild-scoped publish")
	}

	topic := fmt.Sprintf("%s.%s", baseTopic, guildID)
	return bus.Publish(topic, msg)
}

// FormatGuildScopedTopic formats a topic with guild_id suffix without publishing.
// Useful for generating subscription patterns or testing.
func FormatGuildScopedTopic(baseTopic string, guildID string) string {
	return fmt.Sprintf("%s.%s", baseTopic, guildID)
}
