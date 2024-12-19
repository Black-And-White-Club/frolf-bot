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

// SubscribeToRoundFinalizationEvents subscribes to round finalization events.
func (s *RoundSubscribers) SubscribeToRoundFinalizationEvents(ctx context.Context) error {
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

	// Subscribe to RoundFinalizedEvent events
	log.Println("Subscribing to RoundFinalizedEvent events...")
	if _, err := s.JS.Subscribe(roundevents.RoundFinalizedSubject, func(msg *nats.Msg) {
		log.Printf("Received RoundFinalizedEvent: %+v\n", msg)

		wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		if err := s.Handlers.HandleFinalizeRound(wmMsg); err != nil {
			log.Printf("Error handling RoundFinalizedEvent: %v\n", err)
		} else {
			log.Println("RoundFinalizedEvent handled successfully")
		}
	}, nats.BindStream(roundevents.RoundStream)); err != nil {
		return fmt.Errorf("failed to subscribe to RoundFinalizedEvent events: %w", err)
	}
	log.Println("Subscribed to RoundFinalizedEvent events")

	return nil
}
