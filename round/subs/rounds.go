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
	roundSubscriber     message.Subscriber
	roundSubscriberOnce sync.Once
)

// SubscribeToRoundEvents subscribes to round-related events.
func SubscribeToRoundEvents(ctx context.Context, subscriber message.Subscriber, handler *round.RoundEventHandler) error {
	var err error
	roundSubscriberOnce.Do(func() {
		roundSubscriber = subscriber

		// Subscribe to RoundStartedEvent
		roundStartedChan, err := subscriber.Subscribe(ctx, round.RoundStartedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundStartedEvent{}.Topic(), err)
			return
		}

		go handleRoundStartedEvents(ctx, roundStartedChan, handler)

		// Subscribe to RoundStartingOneHourEvent
		oneHourChan, err := subscriber.Subscribe(ctx, round.RoundStartingOneHourEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundStartingOneHourEvent{}.Topic(), err)
			return
		}

		go handleRoundStartingOneHourEvents(ctx, oneHourChan, handler)

		// Subscribe to RoundStartingThirtyMinutesEvent
		thirtyMinutesChan, err := subscriber.Subscribe(ctx, round.RoundStartingThirtyMinutesEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundStartingThirtyMinutesEvent{}.Topic(), err)
			return
		}

		go handleRoundStartingThirtyMinutesEvents(ctx, thirtyMinutesChan, handler)

		// Subscribe to RoundCreateEvent
		roundCreateChan, err := subscriber.Subscribe(ctx, round.RoundCreateEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundCreateEvent{}.Topic(), err)
			return
		}

		go handleRoundCreateEvents(ctx, roundCreateChan, handler)

		// Subscribe to RoundUpdatedEvent
		roundUpdatedChan, err := subscriber.Subscribe(ctx, round.RoundUpdatedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundUpdatedEvent{}.Topic(), err)
			return
		}

		go handleRoundUpdatedEvents(ctx, roundUpdatedChan, handler)

		// Subscribe to RoundDeletedEvent
		roundDeletedChan, err := subscriber.Subscribe(ctx, round.RoundDeletedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundDeletedEvent{}.Topic(), err)
			return
		}

		go handleRoundDeletedEvents(ctx, roundDeletedChan, handler)

		// Subscribe to RoundFinalizedEvent
		roundFinalizedChan, err := subscriber.Subscribe(ctx, round.RoundFinalizedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.RoundFinalizedEvent{}.Topic(), err)
			return
		}

		go handleRoundFinalizedEvents(ctx, roundFinalizedChan, handler)
	})
	return err
}

func handleRoundStartedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundStartedEvent
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

func handleRoundStartingOneHourEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundStartingOneHourEvent
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

func handleRoundStartingThirtyMinutesEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundStartingThirtyMinutesEvent
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

func handleRoundCreateEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan { // msg is correctly used here
		var evt round.RoundCreateEvent
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

func handleRoundUpdatedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundUpdatedEvent
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

func handleRoundDeletedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundDeletedEvent
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

func handleRoundFinalizedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan {
		var evt round.RoundFinalizedEvent
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
