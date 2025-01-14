package leaderboardsubscribers

import (
	"context"
	"fmt"
	"log/slog"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/handlers"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardSubscribers subscribes to leaderboard-related leaderboardevents.
type LeaderboardSubscribers struct {
	EventBus shared.EventBus
	Handlers leaderboardhandlers.Handlers
	logger   *slog.Logger
}

// NewLeaderboardSubscribers creates a new LeaderboardSubscribers instance.
func NewLeaderboardSubscribers(eventBus shared.EventBus, handlers leaderboardhandlers.Handlers, logger *slog.Logger) *LeaderboardSubscribers {
	return &LeaderboardSubscribers{
		EventBus: eventBus,
		Handlers: handlers,
		logger:   logger,
	}
}

// SubscribeToLeaderboardEvents subscribes to leaderboard-related events and routes them to handlers.
func (s *LeaderboardSubscribers) SubscribeToLeaderboardEvents(ctx context.Context) error {
	// Subscribe to LeaderboardUpdatedSubject events
	s.logger.Debug("Subscribing to LeaderboardUpdatedSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.LeaderboardUpdatedSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleLeaderboardUpdate(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle LeaderboardUpdatedSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to LeaderboardUpdatedSubject: %w", err)
	}

	// Subscribe to TagAssignedSubject events
	s.logger.Debug("Subscribing to TagAssignedSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.TagAssignedSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleTagAssigned(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle TagAssignedSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to TagAssignedSubject: %w", err)
	}

	// Subscribe to TagSwapRequestedSubject events
	s.logger.Debug("Subscribing to TagSwapRequestedSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.TagSwapRequestedSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleTagSwapRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle TagSwapRequestedSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to TagSwapRequestedSubject: %w", err)
	}

	// Subscribe to GetLeaderboardRequestSubject events
	s.logger.Debug("Subscribing to GetLeaderboardRequestSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.GetLeaderboardRequestSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleGetLeaderboardRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle GetLeaderboardRequestSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to GetLeaderboardRequestSubject: %w", err)
	}

	// Subscribe to GetTagByDiscordIDRequestSubject events
	s.logger.Debug("Subscribing to GetTagByDiscordIDRequestSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.GetTagByDiscordIDRequestSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleGetTagByDiscordIDRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle GetTagByDiscordIDRequestSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to GetTagByDiscordIDRequestSubject: %w", err)
	}

	// Subscribe to CheckTagAvailabilityRequestSubject events
	s.logger.Debug("Subscribing to CheckTagAvailabilityRequestSubject")
	if err := s.EventBus.Subscribe(ctx, leaderboardevents.LeaderboardStreamName, leaderboardevents.CheckTagAvailabilityRequestSubject, func(ctx context.Context, msg *message.Message) error {
		if err := s.Handlers.HandleCheckTagAvailabilityRequest(ctx, msg); err != nil {
			return fmt.Errorf("failed to handle CheckTagAvailabilityRequestSubject: %w", err)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to subscribe to CheckTagAvailabilityRequestSubject: %w", err)
	}

	return nil
}
