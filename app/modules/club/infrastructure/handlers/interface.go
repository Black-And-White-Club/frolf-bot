package clubhandlers

import (
	"context"

	clubevents "github.com/Black-And-White-Club/frolf-bot-shared/events/club"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
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

	HandleChallengeListRequest(ctx context.Context, payload *clubevents.ChallengeListRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeDetailRequest(ctx context.Context, payload *clubevents.ChallengeDetailRequestPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeOpenRequested(ctx context.Context, payload *clubevents.ChallengeOpenRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeRespondRequested(ctx context.Context, payload *clubevents.ChallengeRespondRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeWithdrawRequested(ctx context.Context, payload *clubevents.ChallengeWithdrawRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeHideRequested(ctx context.Context, payload *clubevents.ChallengeHideRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeRoundLinkRequested(ctx context.Context, payload *clubevents.ChallengeRoundLinkRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeRoundUnlinkRequested(ctx context.Context, payload *clubevents.ChallengeRoundUnlinkRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeMessageBindRequested(ctx context.Context, payload *clubevents.ChallengeMessageBindRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleChallengeExpireRequested(ctx context.Context, payload *clubevents.ChallengeExpireRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundFinalized(ctx context.Context, payload *roundevents.RoundFinalizedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundDeleted(ctx context.Context, payload *roundevents.RoundDeletedPayloadV1) ([]handlerwrapper.Result, error)
	HandleLeaderboardTagUpdated(ctx context.Context, payload *leaderboardevents.LeaderboardTagUpdatedPayloadV1) ([]handlerwrapper.Result, error)
}
