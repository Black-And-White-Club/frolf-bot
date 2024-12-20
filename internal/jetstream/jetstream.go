package jetstream

import (
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	nc "github.com/nats-io/nats.go"
)

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

	// Create the stream with a wildcard subject
	_, err = sc.js.AddStream(&nc.StreamConfig{
		Name:     streamName,
		Subjects: []string{fmt.Sprintf("%s.>", streamName)},
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
