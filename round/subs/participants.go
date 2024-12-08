package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/Black-And-White-Club/tcr-bot/round"
	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	roundqueries "github.com/Black-And-White-Club/tcr-bot/round/queries"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	participantSubscriber     message.Subscriber
	participantSubscriberOnce sync.Once
)

// SubscribeToParticipantEvents subscribes to participant-related events.
func SubscribeToParticipantEvents(ctx context.Context, subscriber message.Subscriber, handler round.RoundEventHandler) error {
	var err error
	participantSubscriberOnce.Do(func() {
		participantSubscriber = subscriber

		// Subscribe to PlayerAddedToRoundEvent
		msgChan, err := subscriber.Subscribe(ctx, roundevents.PlayerAddedToRoundEvent{}.Topic()) // Use roundevents.PlayerAddedToRoundEvent
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", roundevents.PlayerAddedToRoundEvent{}.Topic(), err) // Use roundevents.PlayerAddedToRoundEvent
			return
		}

		go func() {
			for msg := range msgChan {
				var evt roundevents.PlayerAddedToRoundEvent // Use roundevents.PlayerAddedToRoundEvent
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
		tagChan, err := subscriber.Subscribe(ctx, roundevents.TagNumberRetrievedEvent{}.Topic()) // Use roundevents.TagNumberRetrievedEvent
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", roundevents.TagNumberRetrievedEvent{}.Topic(), err) // Use roundevents.TagNumberRetrievedEvent
			return
		}

		go func() {
			for msg := range tagChan {
				var evt roundevents.TagNumberRetrievedEvent // Use roundevents.TagNumberRetrievedEvent
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

	return err
}

// StartParticipantSubscribers starts the participant subscribers if there are active rounds.
func StartParticipantSubscribers(ctx context.Context, roundQueryService *roundqueries.RoundQueryService, handler round.RoundEventHandler) error { // Use roundevents.RoundEventHandler
	// Check if there are any active rounds
	hasActiveRounds, err := roundQueryService.HasActiveRounds(ctx)
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
