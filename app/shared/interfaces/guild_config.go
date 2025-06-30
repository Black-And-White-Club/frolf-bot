package sharedinterface

import (
	"context"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
)

type GuildConfigReader interface {
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guilddb.GuildConfig, error)
}
