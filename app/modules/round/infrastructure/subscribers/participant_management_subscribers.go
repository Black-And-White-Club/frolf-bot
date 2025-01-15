package roundsubscribers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscribeToParticipantManagementEvents subscribes to participant management events.
func (s *RoundEventSubscribers) SubscribeToParticipantManagementEvents(ctx context.Context) error {
	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.ParticipantResponse, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleParticipantResponse(ctx, msg) // Pass context to the handler
	}); err != nil {
		return fmt.Errorf("failed to subscribe to ParticipantResponseEvent events: %w", err)
	}

	if err := s.eventBus.Subscribe(ctx, roundevents.RoundStreamName, roundevents.ScoreUpdated, func(ctx context.Context, msg *message.Message) error {
		return s.handlers.HandleScoreUpdated(ctx, msg) // Pass context to the handler
	}); err != nil {
		return fmt.Errorf("failed to subscribe to ScoreUpdatedEvent events: %w", err)
	}

	return nil
}
