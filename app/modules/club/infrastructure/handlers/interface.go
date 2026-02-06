package clubhandlers

import (
	"context"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for club event handlers.
type Handlers interface {
	// HandleClubInfoRequest handles requests for club info.
	HandleClubInfoRequest(ctx context.Context, payload *clubevents.ClubInfoRequestPayloadV1) ([]handlerwrapper.Result, error)

	// HandleGuildSetup handles guild setup events to sync club info.
	HandleGuildSetup(ctx context.Context, payload *guildevents.GuildSetupPayloadV1) ([]handlerwrapper.Result, error)

	// HandleClubSyncFromDiscord handles cross-module club sync events from user signup.
	HandleClubSyncFromDiscord(ctx context.Context, payload *sharedevents.ClubSyncFromDiscordRequestedPayloadV1) ([]handlerwrapper.Result, error)
}
