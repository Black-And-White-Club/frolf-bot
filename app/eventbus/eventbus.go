package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	roundevents "github.com/Black-And-White-Club/tcr-bot/app/modules/round/domain/events"
	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
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

	// Initialize the publisher
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:         natsURL,
			Marshaler:   marshaller,
			NatsOptions: []nc.Option{nc.RetryOnFailedConnect(true)},
			// JetStream's asynchronous publishing is enabled by default
			// You can customize it further using the JetStreamConfig field
			// if needed (e.g., setting AckAsync, timeouts, etc.)
		},
		watermillLogger,
	)
	if err != nil {
		natsConn.Close()
		logger.Error("Failed to create Watermill publisher", slog.Any("error", err))
		return nil, fmt.Errorf("failed to create Watermill publisher: %w", err)
	}

	return &eventBus{
		publisher:      publisher,
		js:             js,
		natsConn:       natsConn,
		logger:         logger,
		createdStreams: make(map[string]bool),
	}, nil
}

// Publish sends a message to a specific subject within a stream.
func (eb *eventBus) Publish(ctx context.Context, streamName string, msg *message.Message) error {
	// Ensure the message has a unique UUID
	if msg.UUID == "" {
		msg.UUID = watermill.NewUUID()
	}

	// Extract subject from metadata
	subject := msg.Metadata.Get("subject")
	if subject == "" {
		return fmt.Errorf("message metadata does not contain a valid 'subject'")
	}

	// Log the details before publishing
	eb.logger.Debug("Publishing message",
		slog.String("stream_name", streamName),
		slog.String("subject", subject),
		slog.String("message_id", msg.UUID),
		slog.String("payload", string(msg.Payload)),
	)

	// Asynchronous publish with acknowledgement
	type AsyncPublishResult struct {
		Msg *message.Message
		Err error
	}
	ackChan := make(chan AsyncPublishResult, 1)

	go func() {
		// Publish the message asynchronously
		err := eb.publisher.Publish(subject, msg)
		ackChan <- AsyncPublishResult{Msg: msg, Err: err}
		close(ackChan)
	}()

	// Set a timeout for the acknowledgement
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		eb.logger.Error("Timeout waiting for publish acknowledgement",
			slog.String("stream_name", streamName),
			slog.String("subject", subject),
			slog.String("message_id", msg.UUID),
		)
		return fmt.Errorf("timeout waiting for publish acknowledgement")

	case result := <-ackChan:
		if result.Err != nil {
			eb.logger.Error("Failed to publish message",
				slog.String("stream_name", streamName),
				slog.String("subject", subject),
				slog.String("message_id", msg.UUID),
				slog.Any("error", result.Err),
			)
			return fmt.Errorf("failed to publish message to JetStream: %w", result.Err)
		}

		eb.logger.Info("Message published successfully",
			slog.String("stream_name", streamName),
			slog.String("subject", subject),
			slog.String("message_id", msg.UUID),
		)

		return nil
	}
}

// Subscribe listens to a subject within a stream and processes messages.
func (eb *eventBus) Subscribe(ctx context.Context, streamName, subject string, handler func(ctx context.Context, msg *message.Message) error) error {
	eb.logger.Info("Subscribing",
		slog.String("stream_name", streamName),
		slog.String("subject", subject),
	)

	// Add logging before subscribing
	eb.logger.Info("Attempting to subscribe to stream", slog.String("stream_name", streamName))

	// Create a channel to receive messages
	messages := make(chan *nc.Msg)

	// Subscribe using nats.Subscribe with the channel
	_, err := eb.natsConn.Subscribe(subject, func(msg *nc.Msg) {
		messages <- msg
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to subject %s: %w", subject, err)
	}

	// Add logging after subscribing
	eb.logger.Info("Successfully subscribed to stream", slog.String("stream_name", streamName))

	go func() {
		for msg := range messages {
			// Convert nats.Msg to watermill.Message
			wmMsg := message.NewMessage(watermill.NewUUID(), msg.Data)
			wmMsg.Metadata.Set("subject", subject) // Set the subject in metadata

			eb.logger.Info("Message received",
				slog.String("subject", subject),
				slog.String("payload", string(msg.Data)),
			)

			if err := handler(ctx, wmMsg); err != nil {
				eb.logger.Error("Handler error", slog.String("subject", subject), slog.Any("error", err))
				// Handle the error (e.g., log, retry, etc.)
				continue
			}

			// Acknowledge the message manually
			msg.Ack()
		}
	}()

	return nil
}

func (eb *eventBus) CreateStream(ctx context.Context, streamName string) error {
	eb.logger.Info("Creating stream", "stream_name", streamName)

	eb.streamMutex.Lock()
	defer eb.streamMutex.Unlock()

	// Check if the stream was already created in this process
	if eb.createdStreams[streamName] {
		eb.logger.Info("Stream already created in this process", "stream_name", streamName)
		return nil
	}

	// Attempt to retrieve the stream
	_, err := eb.js.Stream(ctx, streamName)
	if err != nil && err != jetstream.ErrStreamNotFound {
		return fmt.Errorf("failed to check if stream exists: %w", err)
	}

	if err == jetstream.ErrStreamNotFound {
		// Stream doesn't exist, create it with specific subjects
		var subjects []string

		if streamName == userevents.UserStreamName {
			subjects = []string{
				userevents.UserSignupRequest,
				userevents.UserRoleUpdateRequest,
				userevents.UserSignupResponse,
			}
		} else if streamName == leaderboardevents.LeaderboardStreamName {
			subjects = []string{
				leaderboardevents.LeaderboardUpdatedSubject,
				leaderboardevents.TagAssignedSubject,
				leaderboardevents.TagSwapRequestedSubject,
				leaderboardevents.GetLeaderboardRequestSubject,
				leaderboardevents.GetTagByDiscordIDRequestSubject,
				leaderboardevents.CheckTagAvailabilityRequestSubject,
				leaderboardevents.CheckTagAvailabilityResponseSubject,
			}
		} else if streamName == roundevents.RoundStreamName {
			subjects = []string{
				roundevents.RoundCreateRequest,
				roundevents.RoundCreated,
				roundevents.RoundUpdateRequest,
				roundevents.RoundUpdated,
				roundevents.RoundDeleteRequest,
				roundevents.RoundDeleted,
				roundevents.ParticipantResponse,
				roundevents.ScoreUpdated,
				roundevents.RoundFinalized,
				roundevents.GetUserRoleRequest,
				roundevents.GetUserRoleResponse,
				roundevents.RoundReminder,
				roundevents.RoundStateUpdated,
				roundevents.RoundStarted,
				roundevents.GetTagNumberRequest,
				roundevents.GetTagNumberResponse,
				roundevents.ParticipantJoined,
				roundevents.ProcessRoundScoresRequest,
			}
		} else {
			return fmt.Errorf("unknown stream name: %s", streamName)
		}

		_, err = eb.js.CreateStream(ctx, jetstream.StreamConfig{
			Name:     streamName,
			Subjects: subjects, // Include specific subjects
		})
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		eb.logger.Info("Stream created with subjects", "stream_name", streamName, "subjects", subjects)
	} else {
		eb.logger.Info("Stream already exists", "stream_name", streamName)
	}

	// Mark the stream as created in this process
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

	// Close the NATS connection
	if eb.natsConn != nil {
		eb.natsConn.Close()
	}

	return nil
}
