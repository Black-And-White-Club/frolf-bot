package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscribeToRoundManagementEvents subscribes to round management events.
func (s *RoundEventSubscribers) SubscribeToRoundManagementEvents(ctx context.Context) error {
	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.RoundCreated, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleCreateRound(ctx, msg)
	}); err != nil {
		return fmt.Errorf("failed to subscribe to RoundCreatedEvent events: %w", err)
	}

	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.RoundUpdated, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleUpdateRound(ctx, msg)
	}); err != nil {
		return fmt.Errorf("failed to subscribe to RoundUpdatedEvent events: %w", err)
	}

	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.RoundDeleted, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleDeleteRound(ctx, msg)
	}); err != nil {
		return fmt.Errorf("failed to subscribe to RoundDeletedEvent events: %w", err)
	}

	return nil
}

// SubscribeToRoundStartedEvents subscribes to RoundStartedEvent events.
func (s *RoundEventSubscribers) SubscribeToRoundStartedEvents(ctx context.Context) error {
	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.RoundStarted, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleStartRound(ctx, msg)
	}); err != nil {
		return fmt.Errorf("failed to subscribe to RoundStartedEvent events: %w", err)
	}

	return nil
}
