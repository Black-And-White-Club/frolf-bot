package eventbus

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/adapters"
	"github.com/Black-And-White-Club/tcr-bot/app/events"
	"github.com/Black-And-White-Club/tcr-bot/app/types"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	watermillMsg "github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
)

// JetStreamEventBus implements the EventBus interface using NATS JetStream.
type JetStreamEventBus struct {
	logger          watermill.LoggerAdapter
	natsURL         string
	publisher       *nats.Publisher
	subscriber      *nats.Subscriber
	streamCreator   *StreamCreator
	notFoundHandler func(eventType events.EventType) error
	middleware      []events.MiddlewareFunc
}

// ensure JetStreamEventBus adheres to the EventBus interface
var _ events.EventBus = (*JetStreamEventBus)(nil)

// NewJetStreamEventBus creates a new JetStreamEventBus.
func NewJetStreamEventBus(natsURL string, logger watermill.LoggerAdapter) (*JetStreamEventBus, error) {
	streamCreator, err := NewStreamCreator(natsURL, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream creator: %w", err)
	}

	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.ErrorHandler(func(_ *nc.Conn, s *nc.Subscription, err error) {
			if s != nil {
				logger.Error("Error in subscription", err, watermill.LogFields{
					"subject": s.Subject,
					"queue":   s.Queue,
				})
			} else {
				logger.Error("Error in connection", err, nil)
			}
		}),
	}

	js, err := nc.Connect(natsURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	_, err = js.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         natsURL,
			NatsOptions: options,
			Marshaler:   &nats.NATSMarshaler{},
			JetStream: nats.JetStreamConfig{
				Disabled:      false,
				AutoProvision: false,
			},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill NATS publisher: %w", err)
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:         natsURL,
			NatsOptions: options,
			Unmarshaler: &nats.NATSMarshaler{},
			JetStream: nats.JetStreamConfig{
				Disabled:      false,
				AutoProvision: false,
			},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill NATS subscriber: %w", err)
	}

	return &JetStreamEventBus{
		logger:        logger,
		natsURL:       natsURL,
		publisher:     publisher,
		subscriber:    subscriber,
		streamCreator: streamCreator,
	}, nil
}

// Publish publishes an event to the specified topic.
func (b *JetStreamEventBus) Publish(ctx context.Context, eventType events.EventType, msg types.Message) error {
	return b.PublishWithMetadata(ctx, eventType, msg, nil)
}

func (b *JetStreamEventBus) PublishWithMetadata(ctx context.Context, eventType events.EventType, msg types.Message, metadata map[string]string) error {
	streamName := eventType.String() // Use the updated EventType.String() method
	if !isValidStreamName(streamName) {
		streamName = "default"
	}

	// Use StreamCreator to ensure stream exists
	err := b.streamCreator.CreateStream(streamName)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	for _, m := range b.middleware {
		if err := m(ctx, eventType, msg); err != nil {
			return err
		}
	}

	// Convert types.Message to watermill.Message
	watermillMsg := watermillMsg.NewMessage(msg.UUID(), msg.Payload())
	watermillMsg.Metadata = msg.Metadata()
	for k, v := range metadata {
		watermillMsg.Metadata[k] = v
	}

	err = b.publisher.Publish(eventType.String(), watermillMsg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Subscribe subscribes to a topic and registers a handler function for received messages.
func (b *JetStreamEventBus) Subscribe(ctx context.Context, topic string, handler func(ctx context.Context, msg types.Message) error) error {
	streamName := topic // Use the updated EventType.String() method
	if !isValidStreamName(streamName) {
		streamName = "default"
	}

	// Use StreamCreator to ensure stream and consumer exist
	err := b.streamCreator.CreateStream(streamName)
	if err != nil {
		return fmt.Errorf("failed to create stream: %w", err)
	}

	consumerName := fmt.Sprintf("%s-consumer", topic)
	err = b.streamCreator.CreateConsumer(streamName, consumerName, topic)
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	watermillHandler := func(msg *watermillMsg.Message) error {
		wrappedMsg := adapters.NewWatermillMessageAdapter(msg.UUID, msg.Payload)
		wrappedMsg.SetContext(ctx)

		return handler(ctx, wrappedMsg)
	}

	messages, err := b.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	go func() {
		for msg := range messages {
			if err := watermillHandler(msg); err != nil {
				b.logger.Error("Error handling message", err, watermill.LogFields{
					"message_uuid": msg.UUID,
					"topic":        topic,
				})
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}()

	return nil
}

func (b *JetStreamEventBus) RegisterNotFoundHandler(handler func(eventType events.EventType) error) {
	b.notFoundHandler = handler
}

func (b *JetStreamEventBus) RegisterMiddleware(middleware events.MiddlewareFunc) {
	b.middleware = append(b.middleware, middleware)
}

// Start starts the event bus.
func (b *JetStreamEventBus) Start(ctx context.Context) error {
	b.logger.Info("Starting JetStream event bus", nil)
	return nil
}

// Stop stops the event bus.
func (b *JetStreamEventBus) Stop(ctx context.Context) error {
	b.logger.Info("Stopping JetStream event bus", nil)
	if err := b.publisher.Close(); err != nil {
		return fmt.Errorf("failed to close publisher: %w", err)
	}
	if err := b.subscriber.Close(); err != nil {
		return fmt.Errorf("failed to close subscriber: %w", err)
	}
	return nil
}

// StreamCreator handles the creation of JetStream streams.
type StreamCreator struct {
	Publisher *nats.Publisher
	Logger    watermill.LoggerAdapter
	NatsURL   string
	nc        *nc.Conn
	js        nc.JetStreamContext
}

// NewStreamCreator creates a new StreamCreator.
func NewStreamCreator(natsURL string, logger watermill.LoggerAdapter) (*StreamCreator, error) {
	// Set up NATS options.
	options := []nc.Option{
		nc.RetryOnFailedConnect(true),
		nc.Timeout(30 * time.Second),
		nc.ReconnectWait(1 * time.Second),
		nc.ErrorHandler(func(_ *nc.Conn, s *nc.Subscription, err error) {
			if s != nil {
				logger.Error("Error in subscription", err, watermill.LogFields{
					"subject": s.Subject,
					"queue":   s.Queue,
				})
			} else {
				logger.Error("Error in connection", err, nil)
			}
		}),
	}

	// Connect to NATS
	conn, err := nc.Connect(natsURL, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         natsURL,
			NatsOptions: options,
			Marshaler:   &nats.NATSMarshaler{},
			JetStream: nats.JetStreamConfig{
				Disabled:      false,
				AutoProvision: false,
			},
		},
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Watermill NATS publisher: %w", err)
	}

	return &StreamCreator{
		Publisher: publisher,
		Logger:    logger,
		NatsURL:   natsURL,
		nc:        conn,
		js:        js,
	}, nil
}

// CreateStream creates a new JetStream stream if it doesn't exist.
func (sc *StreamCreator) CreateStream(streamName string) error {
	sc.Logger.Info("Creating stream", watermill.LogFields{"stream": streamName})

	// Validate stream name
	if !isValidStreamName(streamName) {
		return fmt.Errorf("invalid stream name: %s", streamName)
	}
	// Check if the stream already exists
	streamInfo, err := sc.js.StreamInfo(streamName)
	if err != nil && err != nc.ErrStreamNotFound {
		return fmt.Errorf("failed to get stream info: %w", err)
	}
	if streamInfo != nil {
		sc.Logger.Info("Stream already exists", watermill.LogFields{"stream": streamName})
		return nil
	}

	// Create the stream with subjects based on the module and event name
	_, err = sc.js.AddStream(&nc.StreamConfig{
		Name:     streamName,
		Subjects: []string{fmt.Sprintf("%s.>", streamName)}, // Use the full stream name for subjects
	})
	if err != nil {
		return fmt.Errorf("failed to add stream: %w", err)
	}

	sc.Logger.Info("Stream created", watermill.LogFields{"stream": streamName})
	return nil
}

// CreateConsumer creates a new JetStream consumer for the given stream and subject.
func (sc *StreamCreator) CreateConsumer(streamName, consumerName, subject string) error {
	sc.Logger.Info("Creating consumer", watermill.LogFields{
		"stream":   streamName,
		"consumer": consumerName,
		"subject":  subject,
	})

	_, err := sc.js.AddConsumer(streamName, &nc.ConsumerConfig{
		Durable:       consumerName,        // Set the durable name for the consumer
		DeliverPolicy: nc.DeliverAllPolicy, // Or another suitable policy
		AckPolicy:     nc.AckExplicitPolicy,
		FilterSubject: subject, // Filter subject for the consumer
	})
	if err != nil {
		return fmt.Errorf("failed to create consumer: %w", err)
	}

	sc.Logger.Info("Consumer created", watermill.LogFields{
		"stream":   streamName,
		"consumer": consumerName,
		"subject":  subject,
	})

	return nil
}

// Close closes the publisher
func (sc *StreamCreator) Close() {
	if err := sc.Publisher.Close(); err != nil {
		sc.Logger.Error("Failed to close publisher", err, nil)
	}
	sc.nc.Close()
}

// isValidStreamName checks if a stream name is valid according to NATS rules.
func isValidStreamName(name string) bool {
	// NATS stream names cannot be empty, contain whitespace, periods, asterisks, greater-than signs, or start/end with a hyphen.
	// They can only contain alphanumeric characters, hyphens, or underscores.
	for _, r := range name {
		if !isValidRune(r) {
			return false
		}
	}
	return name != "" && name[0] != '-' && name[len(name)-1] != '-'
}

func isValidRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}
