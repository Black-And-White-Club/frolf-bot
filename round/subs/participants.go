package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Black-And-White-Club/tcr-bot/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	participantSubscriber     message.Subscriber
	participantSubscriberOnce sync.Once
)

// SubscribeToParticipantEvents subscribes to participant-related events.
func SubscribeToParticipantEvents(ctx context.Context, subscriber message.Subscriber, handler *round.RoundEventHandler) error {
	var err error // Declare err variable outside the closure
	participantSubscriberOnce.Do(func() {
		participantSubscriber = subscriber // Store the subscriber for later use

		// Subscribe to PlayerAddedToRoundEvent
		msgChan, err := subscriber.Subscribe(ctx, round.PlayerAddedToRoundEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.PlayerAddedToRoundEvent{}.Topic(), err)
			return
		}

		go func() {
			for msg := range msgChan {
				var evt round.PlayerAddedToRoundEvent
				if err := json.Unmarshal(msg.Payload, &evt); err != nil {
					log.Printf("Failed to unmarshal PlayerAddedToRoundEvent: %v", err)
					msg.Nack()
					continue
				}

				if err := handler.HandlePlayerAddedToRound(ctx, msg); err != nil {
					log.Printf("Failed to handle PlayerAddedToRoundEvent: %v", err)
					msg.Nack()
					continue
				}

				msg.Ack()
			}
		}()

		// Subscribe to TagNumberRetrievedEvent
		tagChan, err := subscriber.Subscribe(ctx, round.TagNumberRetrievedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.TagNumberRetrievedEvent{}.Topic(), err)
			return
		}

		go func() {
			for msg := range tagChan {
				var evt round.TagNumberRetrievedEvent
				if err := json.Unmarshal(msg.Payload, &evt); err != nil {
					log.Printf("Failed to unmarshal TagNumberRetrievedEvent: %v", err)
					msg.Nack()
					continue
				}

				if err := handler.HandleTagNumberRetrieved(ctx, msg); err != nil {
					log.Printf("Failed to handle TagNumberRetrievedEvent: %v", err)
					msg.Nack()
					continue
				}

				msg.Ack()
			}
		}()

		// ... add subscriptions for other participant-related events and start goroutines in the same way ...
	})

	return err // Return the error from the closure
}

// StartParticipantSubscribers starts the participant subscribers if there are active rounds.
func StartParticipantSubscribers(ctx context.Context, roundService *round.RoundService, handler *round.RoundEventHandler) error {
	// Check if there are any active rounds
	hasActiveRounds, err := roundService.HasActiveRounds(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for active rounds: %w", err)
	}

	if hasActiveRounds {
		if err := SubscribeToParticipantEvents(ctx, participantSubscriber, handler); err != nil {
			return fmt.Errorf("failed to subscribe to participant events: %w", err)
		}
	}

	return nil
}
