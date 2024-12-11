// round/subscriber.go

package subscribers

import (
	"context"
	"fmt"

	natsjetstream "github.com/Black-And-White-Club/tcr-bot/nats"
	roundinterface "github.com/Black-And-White-Club/tcr-bot/round/commandsinterface"
	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	queries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	"github.com/ThreeDotsLabs/watermill"
)

type RoundSubscriber struct {
	commandService     roundinterface.CommandService
	queryService       *queries.RoundQueryService
	natsConnectionPool *natsjetstream.NatsConnectionPool
}

func NewRoundSubscriber(commandService roundinterface.CommandService, queryService *queries.RoundQueryService, natsConnectionPool *natsjetstream.NatsConnectionPool) *RoundSubscriber {
	return &RoundSubscriber{
		commandService:     commandService,
		queryService:       queryService,
		natsConnectionPool: natsConnectionPool,
	}
}

func (s *RoundSubscriber) Start(ctx context.Context) error {
	subscriber, err := natsjetstream.NewSubscriber(s.natsConnectionPool.GetURL(), watermill.NewStdLogger(false, false))
	if err != nil {
		return fmt.Errorf("failed to create subscriber: %w", err)
	}

	// Get the publisher from the eventbus package
	publisher := natsjetstream.GetPublisher()

	// Create an instance of the RoundEventHandler implementation
	eventHandler := roundevents.NewRoundEventHandler(s.commandService, publisher)

	// Subscribe to events and dispatch to handlers
	if err := SubscribeToRoundEvents(ctx, subscriber, eventHandler); err != nil {
		return fmt.Errorf("failed to subscribe to round events: %w", err)
	}
	if err := SubscribeToParticipantEvents(ctx, subscriber, eventHandler); err != nil {
		return fmt.Errorf("failed to subscribe to participant events: %w", err)
	}
	if err := SubscribeToScoreEvents(ctx, subscriber, eventHandler); err != nil {
		return fmt.Errorf("failed to subscribe to score events: %w", err)
	}

	return nil
}
