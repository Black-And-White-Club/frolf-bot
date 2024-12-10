package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	"github.com/ThreeDotsLabs/watermill/message"
)

// SubscribeToRoundEvents subscribes to round-related events.
func SubscribeToRoundEvents(ctx context.Context, subscriber message.Subscriber, handler *roundevents.RoundEventHandlerImpl) error {
	roundStartedChan, err := subscriber.Subscribe(ctx, roundevents.RoundStartedEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundStartedEvent{}.Topic(), err)
	}
	go handleRoundStartedEvents(ctx, roundStartedChan, handler)

	oneHourChan, err := subscriber.Subscribe(ctx, roundevents.RoundStartingOneHourEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundStartingOneHourEvent{}.Topic(), err)
	}
	go handleRoundStartingOneHourEvents(ctx, oneHourChan, handler)

	thirtyMinutesChan, err := subscriber.Subscribe(ctx, roundevents.RoundStartingThirtyMinutesEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundStartingThirtyMinutesEvent{}.Topic(), err)
	}
	go handleRoundStartingThirtyMinutesEvents(ctx, thirtyMinutesChan, handler)

	roundCreateChan, err := subscriber.Subscribe(ctx, roundevents.RoundCreateEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundCreateEvent{}.Topic(), err)
	}
	go handleRoundCreateEvents(ctx, roundCreateChan, handler)

	roundUpdatedChan, err := subscriber.Subscribe(ctx, roundevents.RoundUpdatedEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundUpdatedEvent{}.Topic(), err)
	}
	go handleRoundUpdatedEvents(ctx, roundUpdatedChan, handler)

	roundDeletedChan, err := subscriber.Subscribe(ctx, roundevents.RoundDeletedEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundDeletedEvent{}.Topic(), err)
	}
	go handleRoundDeletedEvents(ctx, roundDeletedChan, handler)

	roundFinalizedChan, err := subscriber.Subscribe(ctx, roundevents.RoundFinalizedEvent{}.Topic())
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", roundevents.RoundFinalizedEvent{}.Topic(), err)
	}
	go handleRoundFinalizedEvents(ctx, roundFinalizedChan, handler)

	return nil // Return nil to indicate success
}

func handleRoundStartedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) { // Changed handler type
	for msg := range msgChan {
		var evt roundevents.RoundStartedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundStartedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundStarted(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundStartedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundStartingOneHourEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundStartingOneHourEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundStartingOneHourEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundStartingOneHour(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundStartingOneHourEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundStartingThirtyMinutesEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundStartingThirtyMinutesEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundStartingThirtyMinutesEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundStartingThirtyMinutes(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundStartingThirtyMinutesEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundCreateEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundCreateEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundCreateEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundCreate(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundCreateEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundUpdatedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundUpdatedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundUpdatedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundUpdated(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundUpdatedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundDeletedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundDeletedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundDeletedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundDeleted(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundDeletedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}

func handleRoundFinalizedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.RoundFinalizedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal RoundFinalizedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleRoundFinalized(ctx, &evt); err != nil {
			log.Printf("Failed to handle RoundFinalizedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}
