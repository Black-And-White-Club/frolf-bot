package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// SubscribeToRoundFinalizationEvents subscribes to round finalization events.
func (s *RoundSubscribers) SubscribeToRoundFinalizationEvents(ctx context.Context) error {
	messages, err := s.Subscriber.Subscribe(ctx, roundevents.RoundFinalizedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to RoundFinalizedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				if err := s.Handlers.HandleFinalizeRound(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}
