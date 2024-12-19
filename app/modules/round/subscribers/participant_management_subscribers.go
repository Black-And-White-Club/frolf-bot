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

// SubscribeToParticipantManagementEvents subscribes to participant management events.
func (s *RoundSubscribers) SubscribeToParticipantManagementEvents(ctx context.Context) error {
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

	// 1. Subscribe to ParticipantResponseEvent events
	log.Println("Subscribing to ParticipantResponseEvent events...")
	if _, err := s.JS.Subscribe(roundevents.ParticipantResponseSubject, func(msg *nats.Msg) {
		log.Printf("Received ParticipantResponseEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleParticipantResponse(wmMsg); err != nil {
			log.Printf("Error handling ParticipantResponseEvent: %v\n", err)
		} else {
			log.Println("ParticipantResponseEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to ParticipantResponseEvent events: %w", err)
	}
	log.Println("Subscribed to ParticipantResponseEvent events")

	// 2. Subscribe to ScoreUpdatedEvent events
	log.Println("Subscribing to ScoreUpdatedEvent events...")
	if _, err := s.JS.Subscribe(roundevents.ScoreUpdatedSubject, func(msg *nats.Msg) {
		log.Printf("Received ScoreUpdatedEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleScoreUpdated(wmMsg); err != nil {
			log.Printf("Error handling ScoreUpdatedEvent: %v\n", err)
		} else {
			log.Println("ScoreUpdatedEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to ScoreUpdatedEvent events: %w", err)
	}
	log.Println("Subscribed to ScoreUpdatedEvent events")

	return nil
}
