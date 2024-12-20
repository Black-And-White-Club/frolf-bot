package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
)

// SubscribeToRoundManagementEvents subscribes to round management events.
func (s *RoundSubscribers) SubscribeToRoundManagementEvents(ctx context.Context) error {
	messages, err := s.Subscriber.Subscribe(ctx, roundevents.RoundCreatedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to RoundCreatedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				if err := s.Handlers.HandleCreateRound(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	updateMessages, err := s.Subscriber.Subscribe(ctx, roundevents.RoundUpdatedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to RoundUpdatedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-updateMessages:
				if err := s.Handlers.HandleUpdateRound(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	deleteMessages, err := s.Subscriber.Subscribe(ctx, roundevents.RoundDeletedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to RoundDeletedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-deleteMessages:
				if err := s.Handlers.HandleDeleteRound(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}

// SubscribeToRoundStartedEvents subscribes to RoundStartedEvent events.
func (s *RoundSubscribers) SubscribeToRoundStartedEvents(ctx context.Context) error {
	messages, err := s.Subscriber.Subscribe(ctx, roundevents.RoundStartedSubject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to RoundStartedEvent events: %w", err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				if err := s.Handlers.HandleStartRound(msg); err != nil {
					msg.Nack()
				} else {
					msg.Ack()
				}
			}
		}
	}()

	return nil
}
