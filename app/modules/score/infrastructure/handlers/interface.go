package scorehandlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface to uncouple handlers from specific implementations.
type Handlers interface {
	HandleScoreCorrected(ctx context.Context, msg *message.Message) error
}
