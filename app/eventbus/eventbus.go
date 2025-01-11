package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// eventBus implements the shared.EventBus interface.
type eventBus struct {
	publisher      message.Publisher
	subscriber     message.Subscriber
	js             jetstream.JetStream
	natsConn       *nc.Conn
	logger         *slog.Logger
	createdStreams map[string]bool
	streamMutex    sync.Mutex
}

// NewEventBus creates and returns an EventBus with a connection to NATS JetStream.
func NewEventBus(ctx context.Context, natsURL string, logger *slog.Logger) (shared.EventBus, error) {
	// Connect to NATS
	natsConn, err := nc.Connect(natsURL)
	if err != nil {
		logger.Error("Failed to connect to NATS", slog.Any("error", err))
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Initialize JetStream
	js, err := jetstream.New(natsConn)
	if err != nil {
		natsConn.Close()
		logger.Error("Failed to initialize JetStream", slog.Any("error", err))
		return nil, fmt.Errorf("failed to initialize JetStream: %w", err)
	}

	// Create a Watermill logger that wraps slog
	watermillLogger := watermill.NewSlogLogger(logger)

	// Create a Marshaller for the publisher
	marshaller := &nats.NATSMarshaler{}

	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:       natsURL,
			Marshaler: marshaller,
			NatsOptions: []nc.Option{
				nc.RetryOnFailedConnect(true),
			},
		},
		watermillLogger, // Use the Watermill logger
	)
	if err != nil {
		natsConn.Close()
		logger.Error("Failed to create Watermill publisher", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:         natsURL,
			Unmarshaler: marshaller,
			NatsOptions: []nc.Option{
				nc.RetryOnFailedConnect(true),
			},
		},
		watermillLogger, // Use the Watermill logger
	)
	if err != nil {
		natsConn.Close()
		publisher.Close()
		logger.Error("Failed to create Watermill subscriber", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create Watermill subscriber: %w", err)
	}

	return &eventBus{
		publisher:      publisher,
		subscriber:     subscriber,
		js:             js,
		natsConn:       natsConn,
		logger:         logger,
		createdStreams: make(map[string]bool),
	}, nil
}

func (eb *eventBus) Publish(ctx context.Context, streamName string, msg *message.Message) error {
	// Ensure the message has a unique UUID
	if msg.UUID == "" {
		msg.UUID = watermill.NewUUID()
	}

	// Log details before publishing
	eb.logger.Debug("Publishing message",
		slog.String("stream_name", streamName),
		slog.String("subject", msg.Metadata.Get("subject")),
		slog.String("payload", string(msg.Payload)),
	)

	// Get JetStream context directly from the NATS connection
	js, err := eb.natsConn.JetStream()
	if err != nil {
		eb.logger.Error("Failed to get JetStream context", slog.Any("error", err))
		return fmt.Errorf("failed to get JetStream context: %w", err)
	}

	// Use the subject directly from the message's metadata
	subject := msg.Metadata.Get("subject")
	if subject == "" {
		return fmt.Errorf("message does not have a subject set in metadata")
	}

	// Publish the message
	ack, err := js.Publish(subject, msg.Payload)
	if err != nil {
		eb.logger.Error("Failed to publish message",
			slog.String("subject", subject),
			slog.String("stream_name", streamName),
			slog.Any("error", err),
		)
		return fmt.Errorf("failed to publish message to JetStream: %w", err)
	}

	// Log successful publishing
	eb.logger.Info("Message published successfully",
		slog.String("stream_name", streamName),
		slog.String("subject", subject),
		slog.Uint64("sequence", ack.Sequence),
	)

	return nil
}

func (eb *eventBus) Subscribe(ctx context.Context, streamName string, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
	eb.logger.Info("Subscribing to subject", slog.String("subject", subject))

	messages, err := eb.subscriber.Subscribe(ctx, subject)
	if err != nil {
		return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	eb.logger.Info("Subscription started", slog.String("subject", subject))

	go func() {
		for msg := range messages {
			// Log the message when received
			eb.logger.Info(
				"Message received by subscriber",
				slog.String("subject", subject),
				slog.String("payload", string(msg.Payload)),
			)

			if err := handler(ctx, msg); err != nil {
				eb.logger.Error("Handler error", slog.String("subject", subject), "error", err)
				msg.Nack()
				continue
			}
			msg.Ack()
		}
	}()

	return nil
}

func (eb *eventBus) CreateStream(ctx context.Context, streamName string, subject string) error {
	eb.logger.Info("Creating stream", "stream_name", streamName, "subject", subject)

	eb.streamMutex.Lock()
	defer eb.streamMutex.Unlock()

	// Check if the stream was already created in this process
	if eb.createdStreams[streamName] {
		eb.logger.Info("Stream already created in this process", "stream_name", streamName)
		return nil
	}

	// Attempt to retrieve the stream
	stream, err := eb.js.Stream(ctx, streamName)
	if err != nil && err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("failed to check if stream exists: %w", err)
	}

	if err == jetstream.ErrStreamNotFound {
		// Stream doesn't exist, create it
		_, err = eb.js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     streamName,
			Subjects: []string{subject},
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		eb.logger.Info("Stream created", "stream_name", streamName, "subject", subject)
	} else {
		// Stream exists, check if the subject needs to be added
		streamInfo, err := stream.Info(ctx)
		if err != nil {
			return fmt.Errorf("failed to get stream info: %w", err)
		}

		// Check if the subject already exists
		found := false
		for _, existingSubject := range streamInfo.Config.Subjects {
			if existingSubject == subject {
				found = true
				break
			}
		}

		if !found {
			// Subject not found, update the stream configuration
			streamInfo.Config.Subjects = append(streamInfo.Config.Subjects, subject)
			_, err = eb.js.UpdateStream(ctx, streamInfo.Config)
			if err != nil {
				return fmt.Errorf("failed to update stream with new subject: %w", err)
			}
			eb.logger.Info("Stream updated with new subject", "stream_name", streamName, "subject", subject)
		} else {
			eb.logger.Info("Stream already exists with subject", "stream_name", streamName, "subject", subject)
		}
	}

	// Wait for stream creation confirmation
	retries := 5
	retryInterval := 100 * time.Millisecond
	for i := 0; i < retries; i++ {
		_, err = eb.js.Stream(ctx, streamName)
		if err == nil {
			eb.logger.Info("Stream creation confirmed", "stream_name", streamName)
			break // Stream is ready
		}
		if err != jetstream.ErrStreamNotFound {
			eb.logger.Error("Failed to check if stream exists", "error", err, "stream_name", streamName)
			return fmt.Errorf("failed to check if stream exists: %w", err)
		}
		eb.logger.Warn("Stream not yet available, retrying...", "stream_name", streamName, "attempt", i+1)
		time.Sleep(retryInterval)
	}

	if err != nil {
		eb.logger.Error("Failed to confirm stream creation after retries", "error", err, "stream_name", streamName)
		return fmt.Errorf("failed to confirm stream creation after retries: %w", err)
	}

	// Mark the stream as created
	eb.createdStreams[streamName] = true

	return nil
}

// Close closes all NATS and Watermill resources.
func (eb *eventBus) Close() error {
	if eb.publisher != nil {
		if err := eb.publisher.Close(); err != nil {
			eb.logger.Error("Error closing NATS publisher", "error", err)
		}
	}
	if eb.subscriber != nil {
		if err := eb.subscriber.Close(); err != nil {
			eb.logger.Error("Error closing NATS subscriber", "error", err)
		}
	}

	// Close the NATS connection
	if eb.natsConn != nil {
		eb.natsConn.Close()
	}

	return nil
}
