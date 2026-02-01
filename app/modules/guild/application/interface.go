package guildservice

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// GuildConfigResult is a type alias to reduce generic verbosity.
// Defined here so it's available to both the interface and the implementation.
type GuildConfigResult = results.OperationResult[*guildtypes.GuildConfig, error]

// Service defines the interface for guild operations.
type Service interface {
	CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildConfigResult, error)
	GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildConfigResult, error)
	UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildConfigResult, error)
	DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildConfigResult, error)
}
