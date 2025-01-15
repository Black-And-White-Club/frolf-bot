package roundhandlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers handles round-related events.
type Handlers interface {
	HandleCreateRound(ctx context.Context, msg *message.Message) error
	HandleUpdateRound(ctx context.Context, msg *message.Message) error
	HandleDeleteRound(ctx context.Context, msg *message.Message) error
	HandleParticipantResponse(ctx context.Context, msg *message.Message) error
	HandleScoreUpdated(ctx context.Context, msg *message.Message) error
	HandleFinalizeRound(ctx context.Context, msg *message.Message) error
	HandleStartRound(ctx context.Context, msg *message.Message) error
}
