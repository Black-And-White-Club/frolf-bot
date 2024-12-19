package round

import (
	"context"
	"fmt"

	rounddb "github.com/Black-And-White-Club/tcr-bot/app/modules/round/db"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	roundservice "github.com/Black-And-White-Club/tcr-bot/app/modules/round/service"
	roundsubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/subscribers"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream" // Assuming this is your JetStream helper package
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/nats-io/nats.go"
)

// Module represents the round module.
type Module struct {
	RoundService  roundservice.Service
	RoundHandlers roundhandlers.Handlers
}

// Initialize initializes the round module.
func (m *Module) Initialize(ctx context.Context, js nats.JetStreamContext) error {
	// 1. Initialize dependencies
	roundDB := &rounddb.RoundDBImpl{} // Replace with your actual DB initialization

	// Initialize Watermill publisher
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: 1000,
		},
		watermill.NopLogger{},
	)

	roundService := &roundservice.RoundService{
		RoundDB:   roundDB,
		JS:        js,
		Publisher: js, // Use the NATS connection as the publisher
	}

	roundHandlers := &roundhandlers.RoundHandlers{
		RoundService: roundService,
		Publisher:    pubSub, // Use the GoChannel pub/sub as the publisher
	}

	m.RoundService = roundService
	m.RoundHandlers = roundHandlers // Assign the pointer to the interface

	// 2. Create the necessary stream
	if err := jetstream.CreateStream(js, roundevents.RoundStream); err != nil {
		return fmt.Errorf("failed to create round stream: %w", err)
	}

	// 3. Set up subscribers
	subscribers := &roundsubscribers.RoundSubscribers{
		JS:       js,
		Handlers: *roundHandlers, // Dereference the pointer before assigning
	}
	if err := subscribers.SubscribeToRoundManagementEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to round management events: %w", err)
	}
	if err := subscribers.SubscribeToParticipantManagementEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to participant management events: %w", err)
	}
	if err := subscribers.SubscribeToRoundFinalizationEvents(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to round finalization events: %w", err)
	}

	return nil
}

// Init initializes the round module.
func Init(ctx context.Context, js nats.JetStreamContext) (*Module, error) {
	module := &Module{}
	if err := module.Initialize(ctx, js); err != nil {
		return nil, fmt.Errorf("failed to initialize round module: %w", err)
	}
	return module, nil
}
