package leaderboardhandlers

import (
	"context"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for leaderboard event handlers using pure transformation pattern.
type Handlers interface {
	HandleLeaderboardUpdateRequested(ctx context.Context, payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagSwapRequested(ctx context.Context, payload *leaderboardevents.TagSwapRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetLeaderboardRequest(ctx context.Context, payload *leaderboardevents.GetLeaderboardRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGetTagByUserIDRequest(ctx context.Context, payload *sharedevents.DiscordTagLookupRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleTagAvailabilityCheckRequested(ctx context.Context, payload *leaderboardevents.TagAvailabilityCheckRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleBatchTagAssignmentRequested(ctx context.Context, payload *sharedevents.BatchTagAssignmentRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleRoundGetTagRequest(ctx context.Context, payload *sharedevents.RoundTagLookupRequestedPayloadV1) ([]handlerwrapper.Result, error)
	HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error)
}
