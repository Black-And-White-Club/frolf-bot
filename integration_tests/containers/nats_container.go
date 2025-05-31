package containers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupNatsContainer starts a NATS testcontainer with JetStream enabled
// and returns the container instance and the NATS connection URL.
func SetupNatsContainer(ctx context.Context) (*nats.NATSContainer, string, error) {
	log.Println("Starting NATS container with resource limits...")

	// Create a generic container request with proper resource limits
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp", "8222/tcp", "6222/tcp"},
		Cmd: []string{
			"--jetstream",
			// Remove the invalid flags - NATS 2.10 doesn't support these command line options
			// These settings should be configured via config file if needed
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("Server is ready"),
			wait.ForListeningPort("4222/tcp"),
		).WithDeadline(30 * time.Second),
		// Use HostConfigModifier for resource limits
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Resources = container.Resources{
				Memory:   256 * 1024 * 1024, // 256MB limit
				NanoCPUs: 500000000,         // 0.5 CPU (50% of one core)
			}
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to start NATS container: %w", err)
	}

	// Wrap in NATSContainer type for consistency
	natsContainer := &nats.NATSContainer{Container: container}

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		terminateErr := natsContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate NATS container: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get NATS connection string: %w", err)
	}

	log.Printf("NATS container started with resource limits. URL: %s", natsURL)
	return natsContainer, natsURL, nil
}

// SetupNatsContainerWithoutLimits provides a container without resource restrictions
// Use this for development environments where you want maximum performance
func SetupNatsContainerWithoutLimits(ctx context.Context) (*nats.NATSContainer, string, error) {
	log.Println("Starting NATS container without resource limits...")

	// Use the nats module's Run function for simpler setup
	natsContainer, err := nats.Run(ctx,
		"nats:2.10-alpine",
		// Remove the invalid arguments - use JetStream option instead
		testcontainers.WithWaitStrategy(
			wait.ForAll(
				wait.ForLog("Server is ready"),
				wait.ForListeningPort("4222/tcp"),
			).WithDeadline(30*time.Second),
		),
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to start NATS container: %w", err)
	}

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		terminateErr := natsContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate NATS container: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get NATS connection string: %w", err)
	}

	log.Printf("NATS container started without limits. URL: %s", natsURL)
	return natsContainer, natsURL, nil
}

// SetupNatsContainerWithConfig creates NATS container with a configuration file for advanced settings
func SetupNatsContainerWithConfig(ctx context.Context) (*nats.NATSContainer, string, error) {
	log.Println("Starting NATS container with configuration file...")

	// NATS configuration content
	natsConfig := `port: 4222 http_port: 8222

	jetstream {
		store_dir: "/data"
		max_memory_store: 128MB
		max_file_store: 1GB
	}

	# Connection limits
	max_connections: 100
	max_subscriptions: 1000
	max_payload: 1MB
	max_control_line: 4096

	# Logging
	debug: false
	trace: false
	`

	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp", "8222/tcp", "6222/tcp"},
		Cmd:          []string{"--config", "/etc/nats/nats-server.conf"},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "", // Will be created from content
				ContainerFilePath: "/etc/nats/nats-server.conf",
				FileMode:          0o755,
				Reader:            strings.NewReader(natsConfig),
			},
		},
		WaitingFor: wait.ForAll(
			wait.ForLog("Server is ready"),
			wait.ForListeningPort("4222/tcp"),
		).WithDeadline(30 * time.Second),
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.Resources = container.Resources{
				Memory:   256 * 1024 * 1024, // 256MB limit
				NanoCPUs: 500000000,         // 0.5 CPU (50% of one core)
			}
		},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to start NATS container with config: %w", err)
	}

	natsContainer := &nats.NATSContainer{Container: container}

	natsURL, err := natsContainer.ConnectionString(ctx)
	if err != nil {
		terminateErr := natsContainer.Terminate(ctx)
		if terminateErr != nil {
			log.Printf("Failed to terminate NATS container: %v", terminateErr)
		}
		return nil, "", fmt.Errorf("failed to get NATS connection string: %w", err)
	}

	log.Printf("NATS container started with config file. URL: %s", natsURL)
	return natsContainer, natsURL, nil
}
