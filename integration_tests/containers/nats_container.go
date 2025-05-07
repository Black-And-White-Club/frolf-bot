package containers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/testcontainers/testcontainers-go"              // Import the base testcontainers package
	"github.com/testcontainers/testcontainers-go/modules/nats" // Import the NATS module
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupNatsContainer starts a NATS testcontainer with JetStream enabled
// and returns the container instance and the NATS connection URL.
func SetupNatsContainer(ctx context.Context) (*nats.NATSContainer, string, error) {
	log.Println("Starting NATS container...")

	// Use the nats.Run function from the dedicated module
	// JetStream is enabled by default in the nats.Run command.
	natsContainer, err := nats.Run(ctx,
		// Use a specific NATS version known to support JetStream
		"nats:2.9.22-alpine", // Changed from "nats:latest"
		// Removed nats.WithArgument("js", "") as it's included by default in nats.Run
		// Use a combined waiting strategy: wait for log and the client port to be listening
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("Server is ready"),
				wait.ForListeningPort("4222/tcp"), // Wait for the client port to be listening
			).WithDeadline(45*time.Second), // Increase timeout slightly for combined strategy
		),
	)
	if err != nil {
		// nats.Run handles termination if startup fails
		return nil, "", fmt.Errorf("failed to start NATS container: %w", err)
	}

	// The NATS module provides a helper to get the connection URL
	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		// Terminate the container if getting conn string fails
		terminateErr := natsContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate NATS container after getting connection string failed: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get NATS connection string: %w", err)
	}

	log.Printf("NATS container started and ready. URL: %s", natsURL)

	// Return the container and the connection URL
	// The caller is responsible for terminating the container when done.
	return natsContainer, natsURL, nil
}
