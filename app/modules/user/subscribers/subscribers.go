package usersubscribers

import (
	"context"
	"fmt"
	"log"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/events"
	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// SubscribeToUserEvents subscribes to user-related events and routes them to handlers.
func SubscribeToUserEvents(ctx context.Context, js nats.JetStreamContext, handlers userhandlers.Handlers) error {
	// Ensure the stream exists
	if err := jetstream.CreateStream(js, userevents.UserStream); err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	// 1. Subscribe to UserSignupRequest events
	log.Println("Subscribing to UserSignupRequest events...")

	if _, err := js.Subscribe(userevents.UserSignupRequestSubject, func(msg *nats.Msg) {
		log.Printf("Received UserSignupRequest event: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		log.Printf("Calling HandleUserSignupRequest with message: %+v\n", wmMsg)

		if err := handlers.HandleUserSignupRequest(wmMsg); err != nil {
			log.Printf("Error handling UserSignupRequest: %v\n", err)
		} else {
			log.Println("UserSignupRequest handled successfully")
		}
	}, nats.BindStream(userevents.UserStream)); err != nil {
		return fmt.Errorf("failed to subscribe to UserSignupRequest events: %w", err)
	}

	log.Println("Subscribed to UserSignupRequest events")

	// 2. Subscribe to UserRoleUpdateRequest events
	log.Println("Subscribing to UserRoleUpdateRequest events...")

	if _, err := js.Subscribe(userevents.UserRoleUpdateRequestSubject, func(msg *nats.Msg) {
		log.Printf("Received UserRoleUpdateRequest event: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		log.Printf("Calling HandleUserRoleUpdateRequest with message: %+v\n", wmMsg)

		if err := handlers.HandleUserRoleUpdateRequest(wmMsg); err != nil {
			log.Printf("Error handling UserRoleUpdateRequest: %v\n", err)
		} else {
			log.Println("UserRoleUpdateRequest handled successfully")
		}
	}, nats.BindStream(userevents.UserStream)); err != nil {
		return fmt.Errorf("failed to subscribe to UserRoleUpdateRequest events: %w", err)
	}

	log.Println("Subscribed to UserRoleUpdateRequest events")

	return nil
}
