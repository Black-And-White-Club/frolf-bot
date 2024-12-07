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
	scoreSubscriber     message.Subscriber
	scoreSubscriberOnce sync.Once
)

// SubscribeToScoreEvents subscribes to score-related events.
func SubscribeToScoreEvents(ctx context.Context, subscriber message.Subscriber, handler *round.RoundEventHandler) error {
	var err error
	scoreSubscriberOnce.Do(func() {
		scoreSubscriber = subscriber

		// Subscribe to ScoreSubmittedEvent
		scoreSubmittedChan, err := subscriber.Subscribe(ctx, round.ScoreSubmittedEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", round.ScoreSubmittedEvent{}.Topic(), err)
			return
		}

		go handleScoreSubmittedEvents(ctx, scoreSubmittedChan, handler)

		// ... add subscriptions for other score-related events in the future ...
	})
	return err
}

func handleScoreSubmittedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *round.RoundEventHandler) {
	for msg := range msgChan { // msg is correctly used here
		var evt round.ScoreSubmittedEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal ScoreSubmittedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleScoreSubmitted(ctx, &evt); err != nil {
			log.Printf("Failed to handle ScoreSubmittedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}
