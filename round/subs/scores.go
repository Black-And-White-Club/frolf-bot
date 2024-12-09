package subscribers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	roundevents "github.com/Black-And-White-Club/tcr-bot/round/eventhandling"
	"github.com/ThreeDotsLabs/watermill/message"
)

var (
	scoreSubscriber     message.Subscriber
	scoreSubscriberOnce sync.Once
)

// SubscribeToScoreEvents subscribes to score-related events.
func SubscribeToScoreEvents(ctx context.Context, subscriber message.Subscriber, handler *roundevents.RoundEventHandlerImpl) error { // Changed handler type
	var err error
	scoreSubscriberOnce.Do(func() {
		scoreSubscriber = subscriber

		scoreSubmittedChan, err := subscriber.Subscribe(ctx, roundevents.ScoreSubmissionEvent{}.Topic())
		if err != nil {
			err = fmt.Errorf("failed to subscribe to %s: %w", roundevents.ScoreSubmissionEvent{}.Topic(), err)
			return
		}

		go handleScoreSubmittedEvents(ctx, scoreSubmittedChan, handler)

	})
	return err
}

func handleScoreSubmittedEvents(ctx context.Context, msgChan <-chan *message.Message, handler *roundevents.RoundEventHandlerImpl) {
	for msg := range msgChan {
		var evt roundevents.ScoreSubmissionEvent
		if err := json.Unmarshal(msg.Payload, &evt); err != nil {
			log.Printf("Failed to unmarshal ScoreSubmittedEvent: %v", err)
			msg.Nack()
			continue
		}

		if err := handler.HandleScoreSubmitted(ctx, evt); err != nil { // Pass evt directly
			log.Printf("Failed to handle ScoreSubmittedEvent: %v", err)
			msg.Nack()
			continue
		}

		msg.Ack()
	}
}
