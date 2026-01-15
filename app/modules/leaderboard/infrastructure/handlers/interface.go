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

	// --- INFRASTRUCTURE ---

	// HandleGuildConfigCreated ensures a leaderboard exists when a new guild is configured.
	HandleGuildConfigCreated(ctx context.Context, payload *guildevents.GuildConfigCreatedPayloadV1) ([]handlerwrapper.Result, error)
}
