package guilddb

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

type GuildDB interface {
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*GuildConfig, error)
	SaveConfig(ctx context.Context, config *GuildConfig) error
	UpdateConfig(ctx context.Context, guildID sharedtypes.GuildID, updates map[string]interface{}) error
	DeleteConfig(ctx context.Context, guildID sharedtypes.GuildID) error
}
