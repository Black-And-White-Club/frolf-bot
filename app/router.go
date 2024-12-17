package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	roundhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/round/handlers"
	"github.com/Black-And-White-Club/tcr-bot/internal/jetstream"
	watermillutil "github.com/Black-And-White-Club/tcr-bot/internal/watermill"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"
)

// Module defines the interface for application modules.
type Module interface {
	RegisterHandlers(router *message.Router, pubsub watermillutil.PubSuber) error
	GetHandlers() map[string]struct {
		topic         string
		handler       message.HandlerFunc
		responseTopic string
	}
}

// RegisterHandlers registers handlers for all provided modules and sets up scheduled task handling.
func RegisterHandlers(router *message.Router, natsURL string, logger watermill.LoggerAdapter, modules ...Module) error {
	log.Println("Starting RegisterHandlers function")

	pubsub, err := watermillutil.NewPubSub(natsURL, logger)
	if err != nil {
		return fmt.Errorf("failed to create pubsub instance: %w", err)
	}

	for i, module := range modules {
		log.Printf("Registering handlers for module %d: %T", i+1, module)

		if err := module.RegisterHandlers(router, pubsub); err != nil {
			log.Printf("Failed to register handlers for module %T: %v", module, err)
			return fmt.Errorf("failed to register module handlers: %w", err)
		}

		// Access and log the registered handlers
		for handlerName, h := range module.GetHandlers() {
			log.Printf("  - Registered handler: %s (topic: %s)", handlerName, h.topic)
		}
	}

	jsCtx, err := watermillutil.NewScheduledTaskSubscriber(natsURL, logger)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task subscriber: %w", err)
	}

	if jsCtx == nil {
		return fmt.Errorf("JetStream context not initialized")
	}

	handlerMap := map[string]func(context.Context, *message.Message) error{
		"ReminderOneHourHandler":       roundhandlers.NewReminderHandler(nil, nil).Handle,
		"ReminderThirtyMinutesHandler": roundhandlers.NewReminderHandler(nil, nil).Handle,
		"StartRoundEventHandler":       roundhandlers.NewStartRoundHandler(nil, nil).Handle,
		"ScheduledTaskHandler":         roundhandlers.NewScheduledTaskHandler(nil, nil).Handle,
	}

	scheduledMessageHandler := func(msg *nats.Msg) error {
		watermillMsg := message.NewMessage(watermill.NewUUID(), msg.Data)

		metadata := message.Metadata{}
		for k, v := range msg.Header {
			if len(v) > 0 {
				metadata.Set(k, v[0])
			}
		}
		watermillMsg.Metadata = metadata

		var taskData roundhandlers.TaskData
		if err := json.Unmarshal(msg.Data, &taskData); err != nil {
			return fmt.Errorf("failed to unmarshal task data: %w", err)
		}

		if err := taskData.Validate(); err != nil {
			return fmt.Errorf("invalid task data: %w", err)
		}

		ctx := context.Background()
		ctx = context.WithValue(ctx, roundhandlers.RoundIDKey, taskData.RoundID)
		ctx = context.WithValue(ctx, roundhandlers.HandlerKey, taskData.HandlerName)
		ctx = context.WithValue(ctx, roundhandlers.ScheduledAtKey, watermillMsg.Metadata.Get("scheduled_at"))

		handler, ok := handlerMap[taskData.HandlerName]
		if !ok {
			return fmt.Errorf("unknown handler: %s", taskData.HandlerName)
		}

		return handler(ctx, watermillMsg)
	}

	go func() {
		if err := jetstream.ConsumeScheduledMessages(context.Background(), jsCtx, "scheduled_tasks", scheduledMessageHandler); err != nil {
			log.Fatalf("Error consuming messages: %v", err)
		}
	}()

	return nil
}
