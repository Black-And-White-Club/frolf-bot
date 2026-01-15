package saga

import (
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// SwapIntent represents a user's intent to acquire a specific tag.
type SwapIntent struct {
	UserID     sharedtypes.DiscordID `json:"user_id"`
	CurrentTag sharedtypes.TagNumber `json:"current_tag"`
	TargetTag  sharedtypes.TagNumber `json:"target_tag"`
	GuildID    sharedtypes.GuildID   `json:"guild_id"`
}
