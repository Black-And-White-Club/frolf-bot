package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

// Service handles leaderboard logic.
type Service interface {
	// Single unified tag assignment method
	ProcessTagAssignments(ctx context.Context, source interface{}, requests []sharedtypes.TagAssignmentRequest, requestingUserID *sharedtypes.DiscordID, operationID uuid.UUID, batchID uuid.UUID) (LeaderboardOperationResult, error)

	// Tag Swapping
	TagSwapRequested(ctx context.Context, payload leaderboardevents.TagSwapRequestedPayload) (LeaderboardOperationResult, error)

	// Other Operations
	GetLeaderboard(ctx context.Context) (LeaderboardOperationResult, error)
	GetTagByUserID(ctx context.Context, userID sharedtypes.DiscordID) (LeaderboardOperationResult, error)
	RoundGetTagByUserID(ctx context.Context, payload sharedevents.RoundTagLookupRequestPayload) (LeaderboardOperationResult, error)
	CheckTagAvailability(ctx context.Context, payload leaderboardevents.TagAvailabilityCheckRequestedPayload) (*leaderboardevents.TagAvailabilityCheckResultPayload, *leaderboardevents.TagAvailabilityCheckFailedPayload, error)
}
