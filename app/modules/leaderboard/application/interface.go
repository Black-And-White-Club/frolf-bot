package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Service handles leaderboard logic.
type Service interface {
	// Tag Assignment
	TagAssignmentRequested(ctx context.Context, msg *message.Message, payload leaderboardevents.TagAssignmentRequestedPayload) (LeaderboardOperationResult, error)

	// Tag Swapping
	TagSwapRequested(ctx context.Context, msg *message.Message, payload leaderboardevents.TagSwapRequestedPayload) (LeaderboardOperationResult, error)

	// Other Operations
	GetLeaderboard(ctx context.Context, msg *message.Message) (LeaderboardOperationResult, error)
	GetTagByUserID(ctx context.Context, msg *message.Message, userID sharedtypes.DiscordID, roundID sharedtypes.RoundID) (LeaderboardOperationResult, error)
	CheckTagAvailability(ctx context.Context, msg *message.Message, payload leaderboardevents.TagAvailabilityCheckRequestedPayload) (*leaderboardevents.TagAvailabilityCheckResultPayload, *leaderboardevents.TagAvailabilityCheckFailedPayload, error)
	UpdateLeaderboard(ctx context.Context, msg *message.Message, roundID sharedtypes.RoundID, sortedParticipantTags []string) (LeaderboardOperationResult, error)
}
