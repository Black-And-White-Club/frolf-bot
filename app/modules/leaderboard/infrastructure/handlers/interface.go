package leaderboardhandlers

import (
	"context"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// Handlers defines the interface for leaderboard event handlers.
// these handlers act as "Traffic Cops,"
// coordinating between the Application Funnel and the Swap Saga.
type Handlers interface {
	// --- MUTATIONS ---

	// HandleBatchTagAssignmentRequested processes signups or administrative changes.
	HandleBatchTagAssignmentRequested(ctx context.Context, payload *sharedevents.BatchTagAssignmentRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleLeaderboardUpdateRequested processes score results after round completion.
	HandleLeaderboardUpdateRequested(ctx context.Context, payload *leaderboardevents.LeaderboardUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleTagSwapRequested manages manual intent for one user to claim another's tag.
	HandleTagSwapRequested(ctx context.Context, payload *leaderboardevents.TagSwapRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// --- READS ---

	// HandleGetLeaderboardRequest returns the full current state of the leaderboard.
	HandleGetLeaderboardRequest(ctx context.Context, payload *leaderboardevents.GetLeaderboardRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleGetTagByUserIDRequest performs a general tag lookup for a specific user.
	HandleGetTagByUserIDRequest(ctx context.Context, payload *sharedevents.DiscordTagLookupRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleRoundGetTagRequest provides specialized lookups for the Round module's lifecycle.
	HandleRoundGetTagRequest(ctx context.Context, payload *sharedevents.RoundTagLookupRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleTagAvailabilityCheckRequested checks whether a tag is available for a user.
	HandleTagAvailabilityCheckRequested(ctx context.Context, payload *sharedevents.TagAvailabilityCheckRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// --- ADMIN OPERATIONS ---

	// HandlePointHistoryRequested returns point history for a member.
	HandlePointHistoryRequested(ctx context.Context, payload *leaderboardevents.PointHistoryRequestedPayloadV1) ([]handlerwrapper.Result, error)

	// HandleManualPointAdjustment processes a manual point adjustment.
	HandleManualPointAdjustment(ctx context.Context, payload *leaderboardevents.ManualPointAdjustmentPayloadV1) ([]handlerwrapper.Result, error)

	// HandleRecalculateRound triggers recalculation for a round.
	HandleRecalculateRound(ctx context.Context, payload *leaderboardevents.RecalculateRoundPayloadV1) ([]handlerwrapper.Result, error)

	// HandleStartNewSeason creates a new season.
	HandleStartNewSeason(ctx context.Context, payload *leaderboardevents.StartNewSeasonPayloadV1) ([]handlerwrapper.Result, error)

	// HandleGetSeasonStandings returns standings for a specific season.
	HandleGetSeasonStandings(ctx context.Context, payload *leaderboardevents.GetSeasonStandingsPayloadV1) ([]handlerwrapper.Result, error)

	// --- REQUEST-REPLY (PWA) ---

	// HandleListSeasonsRequest returns all seasons for a guild via request-reply.
	HandleListSeasonsRequest(ctx context.Context, payload *leaderboardevents.ListSeasonsRequestPayloadV1) ([]handlerwrapper.Result, error)

	// HandleSeasonStandingsRequest returns standings for a season via request-reply.
	HandleSeasonStandingsRequest(ctx context.Context, payload *leaderboardevents.SeasonStandingsRequestPayloadV1) ([]handlerwrapper.Result, error)

	// --- INFRASTRUCTURE ---

	// HandleGuildConfigCreated ensures a leaderboard exists when a new guild is configured.
	HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error)
}
