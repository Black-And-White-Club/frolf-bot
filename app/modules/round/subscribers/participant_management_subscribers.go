package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// SubscribeToParticipantManagementEvents subscribes to participant management events.
func (s *RoundSubscribers) SubscribeToParticipantManagementEvents(ctx context.Context) error {
	messages, err := s.Subscriber.Subscribe(ctx, roundevents.ParticipantResponseSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to ParticipantResponseEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				if err := s.Handlers.HandleParticipantResponse(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	scoreMessages, err := s.Subscriber.Subscribe(ctx, roundevents.ScoreUpdatedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to ScoreUpdatedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-scoreMessages:
				if err := s.Handlers.HandleScoreUpdated(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}
