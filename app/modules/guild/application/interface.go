package guildservice

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Service defines the contract for the guild service layer.
type Service interface {
	CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildOperationResult, error)
	GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildOperationResult, error)
	UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildOperationResult, error)
	DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildOperationResult, error)
}
