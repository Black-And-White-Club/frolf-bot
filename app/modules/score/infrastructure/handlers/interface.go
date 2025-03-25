package scorehandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface defines the methods that a set of score handlers should implement.
type Handlers interface {
	HandleProcessRoundScoresRequest(msg *message.Message) ([]*message.Message, error)
	HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error)
}
