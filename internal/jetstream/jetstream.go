package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/nats-io/nats.go"
)

// PublishScheduledMessage publishes a message to JetStream with a scheduled execution time.
func PublishScheduledMessage(ctx context.Context, ps watermillutil.PubSuber, streamName string, roundID int64, handler string, executeAt time.Time) error {
	js := ps.GetJetStreamContext() // Get the JetStream context from PubSuber

	payload, err := json.Marshal(map[string]interface{}{
		"round_id": roundID,
		"handler":  handler,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	msg := &nats.Msg{
		Subject: fmt.Sprintf("%s.reminders", streamName),
		Data:    payload,
		Header:  nats.Header{},
	}

	msg.Header.Set("scheduled_at", executeAt.Format(time.RFC3339))

	_, err = js.PublishMsg(msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// In jetstream.go

// FetchMessagesForRound fetches scheduled messages for a specific round from JetStream.
func FetchMessagesForRound(js nats.JetStreamContext, roundID int64) ([]*nats.Msg, error) {
	// Construct the subject filter based on your message subject pattern
	subjectFilter := fmt.Sprintf("scheduled_tasks.reminders.%d.*", roundID) // Assuming your subject includes the roundID

	// Create a pull-based subscription with the filter
	sub, err := js.PullSubscribe(subjectFilter, "scheduled-tasks-consumer") // Use a durable name
	if err != nil {
		return nil, fmt.Errorf("failed to create pull subscription: %w", err)
	}
	defer sub.Unsubscribe()

	// Fetch messages from the subscription
	msgs, err := sub.Fetch(10, nats.Context(context.Background())) // Adjust fetch options as needed
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return msgs, nil
}
