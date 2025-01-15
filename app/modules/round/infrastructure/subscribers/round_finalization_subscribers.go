package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscribeToRoundFinalizationEvents subscribes to round finalization events.
func (s *RoundEventSubscribers) SubscribeToRoundFinalizationEvents(ctx context.Context) error {
	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.RoundFinalized, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleFinalizeRound(ctx, msg) // Pass context to the handler
	}); err != nil {
		return fmt.Errorf("failed to subscribe to RoundFinalizedEvent events: %w", err)
	}

	return nil
}
