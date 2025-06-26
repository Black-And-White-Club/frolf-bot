package leaderboardhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers defines the interface for leaderboard event handlers.
type Handlers interface {
	HandleLeaderboardUpdateRequested(msg *message.Message) ([]*message.Message, error)
	// HandleTagAssignment(msg *message.Message) ([]*message.Message, error)
	HandleTagSwapRequested(msg *message.Message) ([]*message.Message, error)
	HandleGetLeaderboardRequest(msg *message.Message) ([]*message.Message, error)
	HandleGetTagByUserIDRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagAvailabilityCheckRequested(msg *message.Message) ([]*message.Message, error)
	HandleBatchTagAssignmentRequested(msg *message.Message) ([]*message.Message, error)
	HandleRoundGetTagRequest(msg *message.Message) ([]*message.Message, error)
}
