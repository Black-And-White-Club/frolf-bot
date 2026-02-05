package clubhandlers

import (
	"context"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for club event handlers.
type Handlers interface {
	// HandleClubInfoRequest handles requests for club info.
	HandleClubInfoRequest(ctx context.Context, payload *clubevents.ClubInfoRequestPayloadV1) ([]handlerwrapper.Result, error)

	// HandleGuildSetup handles guild setup events to sync club info.
	HandleGuildSetup(ctx context.Context, payload *guildevents.GuildSetupPayloadV1) ([]handlerwrapper.Result, error)

	// HandleUserSignupRequest handles user signup requests to sync club info.
	HandleUserSignupRequest(ctx context.Context, payload *userevents.UserSignupRequestedPayloadV1) ([]handlerwrapper.Result, error)
}
