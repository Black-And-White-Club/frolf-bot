package leaderboardhandlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface to uncouple handlers from specific implementations.
type Handlers interface {
	HandleLeaderboardUpdate(ctx context.Context, msg *message.Message) error
	HandleTagAssigned(ctx context.Context, msg *message.Message) error
	HandleTagSwapRequest(ctx context.Context, msg *message.Message) error
	HandleGetLeaderboardRequest(ctx context.Context, msg *message.Message) error
	HandleGetTagByDiscordIDRequest(ctx context.Context, msg *message.Message) error
	HandleCheckTagAvailabilityRequest(ctx context.Context, msg *message.Message) error
}
