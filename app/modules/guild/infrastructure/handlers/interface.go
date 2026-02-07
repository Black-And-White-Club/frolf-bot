package guildhandlers

import (
	"context"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the contract for guild event handlers following the pure transformation pattern.
type Handlers interface {
	HandleCreateGuildConfig(ctx context.Context, payload *guildevents.GuildConfigCreationRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRetrieveGuildConfig(ctx context.Context, payload *guildevents.GuildConfigRetrievalRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleUpdateGuildConfig(ctx context.Context, payload *guildevents.GuildConfigUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleDeleteGuildConfig(ctx context.Context, payload *guildevents.GuildConfigDeletionRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildSetup(ctx context.Context, payload *guildtypes.GuildConfig) ([]handlerwrapper.Result, error)
}
