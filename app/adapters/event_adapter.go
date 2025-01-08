package adapters

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// StreamNamingStrategy defines a function to determine the stream name
// for an event type.
type StreamNamingStrategy func(eventType shared.EventType) string

// EventAdapter facilitates publishing and subscribing to events using
// NATS JetStream via Watermill.
type EventAdapter struct {
	natsConn       *nc.Conn
	js             jetstream.JetStream
	natsPublisher  *nats.Publisher
	natsSubscriber *nats.Subscriber
	logger         watermill.LoggerAdapter
	streamNaming   StreamNamingStrategy
}

// NewEventAdapter creates and configures a new EventAdapter for NATS JetStream.
func NewEventAdapter(natsURL string, logger shared.LoggerAdapter, namingStrategy StreamNamingStrategy) (*EventAdapter, error) {
	// Connect to NATS
	natsConn, err := nc.Connect(natsURL,
		nc.RetryOnFailedConnect(true),
		nc.Timeout(10*time.Second),
		nc.ReconnectWait(1*time.Second),
		nc.MaxReconnects(5),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Initialize JetStream
	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	// Configure Watermill publisher
	watermillLogger := watermill.NewStdLogger(false, false) // Consider using your shared.LoggerAdapter

	// Create a Marshaller for the publisher
	marshaller := &nats.NATSMarshaler{}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:       natsURL,
			Marshaler: marshaller,
		},
		watermillLogger,
	)
	if err != nil {
		natsConn.Close()
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	// Create an Unmarshaler for the subscriber
	unmarshaler := &nats.NATSMarshaler{}

	// Configure Watermill subscriber
	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:            natsURL,
			AckWaitTimeout: 30 * time.Second,
			Unmarshaler:    unmarshaler,
		},
		watermillLogger,
	)
	if err != nil {
		natsConn.Close()
		publisher.Close()
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &EventAdapter{
		natsConn:       natsConn,
		js:             js,
		natsPublisher:  publisher,
		natsSubscriber: subscriber,
		logger:         watermillLogger,
		streamNaming:   namingStrategy,
	}, nil
}

// Improve CreateStream to make storage and retention configurable.
func (e *EventAdapter) CreateStream(ctx context.Context, eventType shared.EventType, subjects []string, storage jetstream.StorageType, retention jetstream.RetentionPolicy) error {
	streamName := e.streamNaming(eventType)

	_, err := e.js.Stream(ctx, streamName)
	if err == jetstream.ErrStreamNotFound {
		_, err = e.js.CreateStream(ctx, jetstream.StreamConfig{
			Name:      streamName,
			Subjects:  subjects,
			Storage:   storage,
			Retention: retention,
		})
		if err != nil {
			e.logger.Error("Failed to create stream", err, watermill.LogFields{"stream_name": streamName})
			return fmt.Errorf("failed to create stream %s: %w", streamName, err)
		}
	} else if err != nil {
		e.logger.Error("Failed to get stream", err, watermill.LogFields{"stream_name": streamName})
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	e.logger.Info("Stream created or already exists", watermill.LogFields{"stream_name": streamName})
	return nil
}

// Publish sends a message to the appropriate stream.
func (e *EventAdapter) Publish(ctx context.Context, eventType shared.EventType, payload []byte, metadata map[string]string) error {
	subject := fmt.Sprintf("%s.%s", eventType.Module, eventType.Name)

	// Convert metadata to Watermill's message.Metadata
	meta := message.Metadata{}
	for key, value := range metadata {
		meta.Set(key, value)
	}

	// Create a Watermill message
	msg := message.NewMessage(watermill.NewUUID(), payload)
	msg.Metadata = meta

	// Publish using the Watermill publisher
	if err := e.natsPublisher.Publish(subject, msg); err != nil {
		e.logger.Error("Failed to publish message", err, watermill.LogFields{"subject": subject})
		return fmt.Errorf("failed to publish message to subject %s: %w", subject, err)
	}
	return nil
}

// Subscribe listens to a subject and processes messages using the provided handler.
func (e *EventAdapter) Subscribe(ctx context.Context, eventType shared.EventType, queueGroup string, handler func(payload []byte, metadata map[string]string) error) error {
	subject := fmt.Sprintf("%s.%s", eventType.Module, eventType.Name)
	streamName := e.streamNaming(eventType)

	// Subscribe using Watermill
	messages, err := e.natsSubscriber.Subscribe(ctx, subject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	// Get the stream
	stream, err := e.js.Stream(ctx, streamName)
	if err != nil {
		return fmt.Errorf("failed to get stream %s: %w", streamName, err)
	}

	// Create a consumer with the appropriate options
	consumerName := fmt.Sprintf("%s-consumer", eventType.Name)

	// Consumer configuration
	consumerCfg := jetstream.ConsumerConfig{
		Name:          consumerName,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy, // Set AckPolicy directly in the config
	}

	if queueGroup != "" {
		consumerCfg.Durable = consumerName // Set the durable name for queue consumers
		// No need to set DeliverGroup. Durable name implies a queue group.
	}

	// Create or update the consumer
	_, err = stream.CreateOrUpdateConsumer(ctx, consumerCfg)
	if err != nil {
		return fmt.Errorf("failed to create or update consumer: %w", err)
	}

	// Process messages
	go func() {
		for msg := range messages {
			err := handler(msg.Payload, msg.Metadata)
			if err != nil {
				e.logger.Error("Message handling error", err, watermill.LogFields{"subject": subject})
				msg.Nack() // Nack the message on handler error
				continue
			}
			msg.Ack() // Ack the message after successful processing
		}
	}()

	return nil
}

// Close closes all NATS and Watermill resources.
func (e *EventAdapter) Close() error {
	if e.natsPublisher != nil {
		if err := e.natsPublisher.Close(); err != nil {
			e.logger.Error("Error closing NATS publisher", err, nil)
		}
	}
	if e.natsSubscriber != nil {
		if err := e.natsSubscriber.Close(); err != nil {
			e.logger.Error("Error closing NATS subscriber", err, nil)
		}
	}
	if e.natsConn != nil {
		e.natsConn.Drain()
		e.natsConn.Close()
	}
	return nil
}
