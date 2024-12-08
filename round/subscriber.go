// round/subscriber.go
package round

import (
	"context"
	"fmt"

	events "github.com/Black-And-White-Club/tcr-bot/event_bus"
	"github.com/Black-And-White-Club/tcr-bot/nats"
	subscribers "github.com/Black-And-White-Club/tcr-bot/round/subs"
	"github.com/ThreeDotsLabs/watermill"
)

type RoundSubscriber struct {
	commandService *RoundCommandService
	queryService   *RoundQueryService

	natsConnectionPool *nats.NatsConnectionPool
}

func NewRoundSubscriber(commandService *RoundCommandService, queryService *RoundQueryService, natsConnectionPool *nats.NatsConnectionPool) *RoundSubscriber {
	return &RoundSubscriber{
		commandService:     commandService,
		queryService:       queryService,
		natsConnectionPool: natsConnectionPool,
	}
}

func (s *RoundSubscriber) Start(ctx context.Context) error {
	subscriber, err := events.NewSubscriber(s.natsConnectionPool.GetURL(), watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Subscribe to events and dispatch to handlers
	if err := subscribers.SubscribeToRoundEvents(ctx, subscriber, s.commandService); err != nil { // Use subscribers.
		return fmt.Errorf("failed to subscribe to round events: %w", err)
	}
	if err := subscribers.SubscribeToParticipantEvents(ctx, subscriber, s.commandService); err != nil { // Use subscribers.
		return fmt.Errorf("failed to subscribe to participant events: %w", err)
	}
	if err := subscribers.SubscribeToScoreEvents(ctx, subscriber, s.commandService); err != nil { // Use subscribers.
		return fmt.Errorf("failed to subscribe to score events: %w", err)
	}

	return nil
}
