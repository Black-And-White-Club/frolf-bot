package roundsubscribers

import (
	"context"
	"fmt"
	"log"

	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/events"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// SubscribeToRoundManagementEvents subscribes to round management events.
func (s *RoundSubscribers) SubscribeToRoundManagementEvents(ctx context.Context) error {
	// Check if the stream exists
	stream, err := s.JS.StreamInfo(roundevents.RoundStream)
	if err != nil {
		if err == nats.ErrStreamNotFound {
			// Stream doesn't exist, handle accordingly (e.g., log a warning or attempt to create it)
			log.Printf("Warning: Stream '%s' not found. Attempting to create it...\n", roundevents.RoundStream)
			if err := jetstream.CreateStream(s.JS, roundevents.RoundStream); err != nil {
				return fmt.Errorf("failed to create stream: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get stream info: %w", err)
		}
	} else {
		// Stream exists, log some info about it
		log.Printf("Stream '%s' found. Config: %+v\n", stream.Config.Name, stream.Config)
	}

	// 1. Subscribe to RoundCreatedEvent events
	log.Println("Subscribing to RoundCreatedEvent events...")
	if _, err := s.JS.Subscribe(roundevents.RoundCreatedSubject, func(msg *nats.Msg) {
		log.Printf("Received RoundCreatedEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleCreateRound(wmMsg); err != nil {
			log.Printf("Error handling RoundCreatedEvent: %v\n", err)
		} else {
			log.Println("RoundCreatedEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to RoundCreatedEvent events: %w", err)
	}
	log.Println("Subscribed to RoundCreatedEvent events")

	// 2. Subscribe to RoundUpdatedEvent events
	log.Println("Subscribing to RoundUpdatedEvent events...")
	if _, err := s.JS.Subscribe(roundevents.RoundUpdatedSubject, func(msg *nats.Msg) {
		log.Printf("Received RoundUpdatedEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleUpdateRound(wmMsg); err != nil {
			log.Printf("Error handling RoundUpdatedEvent: %v\n", err)
		} else {
			log.Println("RoundUpdatedEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to RoundUpdatedEvent events: %w", err)
	}
	log.Println("Subscribed to RoundUpdatedEvent events")

	// 3. Subscribe to RoundDeletedEvent events
	log.Println("Subscribing to RoundDeletedEvent events...")
	if _, err := s.JS.Subscribe(roundevents.RoundDeletedSubject, func(msg *nats.Msg) {
		log.Printf("Received RoundDeletedEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleDeleteRound(wmMsg); err != nil {
			log.Printf("Error handling RoundDeletedEvent: %v\n", err)
		} else {
			log.Println("RoundDeletedEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to RoundDeletedEvent events: %w", err)
	}
	log.Println("Subscribed to RoundDeletedEvent events")

	return nil
}
