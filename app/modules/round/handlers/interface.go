package roundhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundHandlers handles round-related events.
type Handlers interface {
	HandleCreateRound(msg *message.Message) error
	HandleUpdateRound(msg *message.Message) error
	HandleDeleteRound(msg *message.Message) error
	HandleParticipantResponse(msg *message.Message) error
	HandleScoreUpdated(msg *message.Message) error
	HandleFinalizeRound(msg *message.Message) error
}
