package guilddb

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type GuildDB interface {
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
	SaveConfig(ctx context.Context, config *guildtypes.GuildConfig) error
	UpdateConfig(ctx context.Context, guildID sharedtypes.GuildID, updates map[string]interface{}) error
}
