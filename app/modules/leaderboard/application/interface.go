package leaderboardservice

import (
	"context"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	"github.com/ThreeDotsLabs/watermill/message"
)

// Service handles leaderboard logic.
type Service interface {
	// Leaderboard Updates
	RoundFinalized(ctx context.Context, msg *message.Message) error
	LeaderboardUpdateRequested(ctx context.Context, msg *message.Message) error

	// Tag Assignment
	TagAssigned(ctx context.Context, msg *message.Message) error
	TagAssignmentRequested(ctx context.Context, msg *message.Message) error
	PublishTagAvailable(_ context.Context, msg *message.Message, payload *leaderboardevents.TagAssignedPayload) error

	// Tag Swapping
	TagSwapRequested(ctx context.Context, msg *message.Message) error
	TagSwapInitiated(ctx context.Context, msg *message.Message) error

	// Other Operations
	GetLeaderboardRequest(ctx context.Context, msg *message.Message) error
	GetTagByUserIDRequest(ctx context.Context, msg *message.Message) error
	TagAvailabilityCheckRequested(ctx context.Context, msg *message.Message) error
}
