package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	nc "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
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

	// Create JetStream context
	js, err := jetstream.New(natsConn)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	// Ensure the stream exists
	streamName := "user"
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{"user.>"}, // Subscribe to all subjects under user
	})
	if err != nil {
		log.Printf("Could not create stream: %v", err)
	}

	// Create a consumer (optional, for inspection)
	consumerName := "user-signup-request-consumer" // Use a fixed name for testing
	_, err = js.CreateOrUpdateConsumer(context.Background(), streamName, jetstream.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: "user.signup.request",
		AckPolicy:     jetstream.AckExplicitPolicy, // Or AckAllPolicy
	})
	if err != nil {
		log.Printf("Could not create consumer: %v", err)
	}

	// Create a NATS subscriber
	subscriber, err := nats.NewSubscriber(
		nats.SubscriberConfig{
			URL:         "nats://localhost:4222", // Your NATS URL
			Unmarshaler: &nats.NATSMarshaler{},
			// ... other config options ...
		},
		logger,
	)
	if err != nil {
		log.Fatalf("Failed to create subscriber: %v", err)
	}
	defer subscriber.Close()

	// Subscribe using the consumer
	messages, err := subscriber.Subscribe(context.Background(), "user_signup_request")
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	// Handle messages
	go func() {
		for msg := range messages {
			log.Printf("Received message: %s, payload: %s", msg.UUID, string(msg.Payload))
			msg.Ack()
		}
	}()

	fmt.Println("Subscriber running. Press Ctrl+C to exit.")

	// Wait for an interrupt signal to gracefully shut down
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down.")
}
