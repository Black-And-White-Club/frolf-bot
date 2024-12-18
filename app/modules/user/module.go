package user

import (
	"context"
	"fmt"
	"log"

	userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	userservice "github.com/Black-And-White-Club/tcr-bot/app/modules/user/service"
	usersubscribers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/subscribers"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/nats-io/nats.go"
)

// Module represents the user module.
type Module struct {
	UserService  userservice.Service
	UserHandlers userhandlers.UserHandlers
}

// Initialize initializes the user module.
func (m *Module) Initialize(ctx context.Context, js nats.JetStreamContext) error { // Remove publisher argument
	// 1. Initialize dependencies
	userDB := &userdb.UserDBImpl{}

	// Initialize Watermill publisher
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: 1000,
		},
		watermill.NopLogger{},
	)

	userService := &userservice.UserService{
		UserDB:    userDB,
		JS:        js,
		Publisher: js, // Use the NATS connection as the publisher
	}

	userHandlers := &userhandlers.UserHandlers{
		UserService: userService,
		Publisher:   pubSub, // Use the GoChannel pub/sub as the publisher
	}

	m.UserService = userService
	m.UserHandlers = *userHandlers

	// 2. Set up subscriptions
	if err := usersubscribers.SubscribeToUserEvents(ctx, js, userHandlers); err != nil {
		return fmt.Errorf("failed to subscribe to user events: %w", err)
	}

	return nil
}

// Init initializes the user module.
func Init(ctx context.Context, js nats.JetStreamContext) (*Module, error) { // Remove publisher argument
	module := &Module{}
	if err := module.Initialize(ctx, js); err != nil {
		log.Fatal(err)
	}
	return module, nil
}
