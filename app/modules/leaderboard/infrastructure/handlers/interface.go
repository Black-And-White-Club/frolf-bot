package leaderboardhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for leaderboard event handlers.
type Handlers interface {
	HandleRoundFinalized(msg *message.Message) error
	HandleLeaderboardUpdateRequested(msg *message.Message) error
	HandleTagAssigned(msg *message.Message) error
	HandleTagAssignmentRequested(msg *message.Message) error
	HandleTagSwapRequested(msg *message.Message) error
	HandleTagSwapInitiated(msg *message.Message) error
	HandleGetLeaderboardRequest(msg *message.Message) error
	HandleGetTagByDiscordIDRequest(msg *message.Message) error
	HandleTagAvailabilityCheckRequested(msg *message.Message) error
	HandleAssignTag(msg *message.Message) error
}
