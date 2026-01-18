package guildservice

import (
	"context"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

// Service defines the interface for guild operations.
type Service interface {
	CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (results.OperationResult, error)
	GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error)
	UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (results.OperationResult, error)
	DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error)
}
