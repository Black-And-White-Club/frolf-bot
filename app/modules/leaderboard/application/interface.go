package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// Service handles leaderboard logic.
type Service interface {
	// Tag Assignment
	BatchTagAssignmentRequested(ctx context.Context, payload leaderboardevents.BatchTagAssignmentRequestedPayload) (LeaderboardOperationResult, error)
	TagAssignmentRequested(ctx context.Context, payload leaderboardevents.TagAssignmentRequestedPayload) (LeaderboardOperationResult, error)

	// Tag Swapping
	TagSwapRequested(ctx context.Context, payload leaderboardevents.TagSwapRequestedPayload) (LeaderboardOperationResult, error)

	// Other Operations
	GetLeaderboard(ctx context.Context) (LeaderboardOperationResult, error)
	GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (LeaderboardOperationResult, error)
	RoundGetTagByUserID(ctx context.Context, payload sharedevents.RoundTagLookupRequestPayload) (LeaderboardOperationResult, error)
	CheckTagAvailability(ctx context.Context, payload leaderboardevents.TagAvailabilityCheckRequestedPayload) (*leaderboardevents.TagAvailabilityCheckResultPayload, *leaderboardevents.TagAvailabilityCheckFailedPayload, error)
	UpdateLeaderboard(ctx context.Context, roundID sharedtypes.RoundID, sortedParticipantTags []string) (LeaderboardOperationResult, error)
}
