package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	nc "github.com/nats-io/nats.go"

	userevents "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/events"
)

func main() {
	logger := watermill.NewStdLogger(true, true) // Enable debug logs

	// Create a NATS connection with options
	natsConn, err := nc.Connect("nats://localhost:4222", // Your NATS URL
		nc.RetryOnFailedConnect(true),
		nc.Timeout(5*time.Second),
		nc.ReconnectWait(1*time.Second),
		nc.MaxReconnects(-1),
	)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsConn.Close()

	// Create a Watermill publisher
	marshaller := &nats.NATSMarshaler{}
	publisher, err := nats.NewPublisher(
		nats.PublisherConfig{
			URL:       "nats://localhost:4222", // Your NATS URL
			Marshaler: marshaller,
		},
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to create publisher: %v", err)
	}
	defer publisher.Close()

	// Generate a correlation ID and message UUID
	correlationID := watermill.NewUUID()
	messageUUID := watermill.NewUUID()
	tagNumber := 2
	// Create the payload
	payload := userevents.UserSignupRequestPayload{
		DiscordID: "13", // Replace with test Discord ID
		TagNumber: &tagNumber,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Failed to marshal payload: %v", err)
	}

	// Create the message
	msg := message.NewMessage(messageUUID, payloadBytes)

	// Set the correlation ID in the metadata
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

	// Remove the following line:
	// msg.Metadata.Set("Nats-Msg-Id", messageUUID)

	// Add a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Publish the message with retries
	err = publishWithRetry(ctx, publisher, userevents.UserSignupRequest, msg)
	if err != nil {
		log.Fatalf("Failed to publish message after retries: %v", err)
	}

	log.Printf("Published message with correlation ID: %s, message ID: %s", correlationID, messageUUID)
	fmt.Println("Test message published. Press Ctrl+C to exit.")

	// Wait for an interrupt signal to gracefully shut down
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down test publisher.")
}

// Helper function for publishing with retries
func publishWithRetry(ctx context.Context, publisher message.Publisher, topic string, msg *message.Message) error {
	var lastErr error
	for i := 0; i < 5; i++ {
		if err := publisher.Publish(topic, msg); err != nil {
			lastErr = err
			delay := time.Duration(i+1) * time.Second
			log.Printf("Error publishing message, retrying in %v: %v\n", delay, err)
			select {
			case <-time.After(delay):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		return nil
	}
	return lastErr
}
