package roundhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// ConsumeScheduledTasks sets up the consumer for scheduled tasks.
func ConsumeScheduledTasks(ctx context.Context, jsCtx nats.JetStreamContext, pubsub watermillutil.PubSuber, handlers map[string]func(ctx context.Context, msg *message.Message) error) error {
	scheduledMessageHandler := func(msg *nats.Msg) error {
		watermillMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		// Convert nats.Header to Watermill Metadata
		metadata := message.Metadata{}
		for k, v := range msg.Header {
			if len(v) > 0 {
				metadata.Set(k, v[0])
			}
		}
		watermillMsg.Metadata = metadata

		var taskData TaskData
		if err := json.Unmarshal(msg.Data, &taskData); err != nil {
			return fmt.Errorf("failed to unmarshal task data: %w", err)
		}

		if err := taskData.Validate(); err != nil {
			return fmt.Errorf("invalid task data: %w", err)
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, RoundIDKey, taskData.RoundID)
		ctx = context.WithValue(ctx, HandlerKey, taskData.HandlerName)
		ctx = context.WithValue(ctx, ScheduledAtKey, watermillMsg.Metadata.Get("scheduled_at"))

		handler, ok := handlers[taskData.HandlerName]
		if !ok {
			return fmt.Errorf("unknown handler: %s", taskData.HandlerName)
		}

		return handler(ctx, watermillMsg)
	}

	go func() {
		if err := jetstream.ConsumeScheduledMessages(ctx, jsCtx, "scheduled_tasks", scheduledMessageHandler); err != nil {
			log.Fatalf("Error consuming messages: %v", err)
		}
	}()

	return nil
}
