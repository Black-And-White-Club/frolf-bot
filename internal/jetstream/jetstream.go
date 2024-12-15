package jetstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// PublishScheduledMessage publishes a message to JetStream with a scheduled execution time.
func PublishScheduledMessage(ctx context.Context, js nats.JetStreamContext, streamName string, roundID int64, handler string, executeAt time.Time) error {
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

// ConsumeScheduledMessages consumes scheduled messages and calls the provided handler function.
func ConsumeScheduledMessages(ctx context.Context, js nats.JetStreamContext, streamName string, handlerFunc func(msg *nats.Msg) error) error {
	// Ensure the stream exists
	_, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{fmt.Sprintf("%s.>", streamName)},
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return fmt.Errorf("failed to add stream: %w", err)
	}

	// Subscribe with a queue group and durable consumer
	sub, err := js.QueueSubscribe(fmt.Sprintf("%s.reminders", streamName), "scheduled_tasks_queue", func(msg *nats.Msg) {
		// Convert nats.Header to message.Metadata
		metadata := message.Metadata{}
		for k, v := range msg.Header {
			if len(v) > 0 {
				metadata.Set(k, v[0]) // Only take the first value if multiple are present
			}
		}

		scheduledAt, err := parseScheduledAt(metadata)
		if err != nil {
			log.Printf("Error parsing scheduled time: %v", err)
			msg.Ack() // Acknowledge invalid messages to prevent redelivery
			return
		}

		if time.Now().Before(scheduledAt) {
			msg.NakWithDelay(time.Until(scheduledAt))
			return
		}

		if err := handlerFunc(msg); err != nil {
			log.Printf("Error handling message: %v", err)
			msg.NakWithDelay(time.Second * 5) // NAK with delay on error
			return
		}
		msg.Ack()
	}, nats.Durable("scheduled_tasks_durable"), nats.AckExplicit())

	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}
	defer sub.Unsubscribe()

	<-ctx.Done() // Block until context is cancelled
	return nil
}

func parseScheduledAt(metadata message.Metadata) (time.Time, error) {
	scheduledAtStr := metadata.Get("scheduled_at")
	if scheduledAtStr == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, scheduledAtStr)
}

// FetchMessagesForRound fetches messages for a specific round from JetStream.
func FetchMessagesForRound(js nats.JetStreamContext, roundID int64) ([]*nats.Msg, error) {
	streamName := "scheduled_tasks"

	cons, err := js.ConsumerInfo(streamName, "scheduled_tasks_durable")
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	sub, err := js.PullSubscribe(fmt.Sprintf("%s.reminders", streamName), "delete_consumer", nats.Bind(streamName, cons.Config.Durable))
	if err != nil {
		return nil, fmt.Errorf("failed to create pull subscription: %w", err)
	}
	defer sub.Unsubscribe()

	var fetchedMessages []*nats.Msg
	for {
		msgs, err := sub.Fetch(10)
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				break
			}
			return nil, fmt.Errorf("failed to fetch messages: %w", err)
		}

		for _, msg := range msgs {
			var payload map[string]interface{}
			if err := json.Unmarshal(msg.Data, &payload); err != nil {
				log.Printf("Error unmarshalling message payload: %v", err)
				continue
			}

			// Correct comparison: Convert payload value to int64
			if idFloat, ok := payload["round_id"].(float64); ok {
				if int64(idFloat) == roundID { // Compare int64 to int64
					fetchedMessages = append(fetchedMessages, msg)
				}
			} else if idString, ok := payload["round_id"].(string); ok {
				idInt, err := strconv.ParseInt(idString, 10, 64)
				if err != nil {
					log.Printf("Error parsing round_id to int64: %v", err)
					continue
				}
				if idInt == roundID {
					fetchedMessages = append(fetchedMessages, msg)
				}
			}
		}
	}

	return fetchedMessages, nil
}
