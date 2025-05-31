package testutils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

// PurgeJetStreamStreams purges all messages from given streams.
func (env *TestEnvironment) PurgeJetStreamStreams(ctx context.Context, streamNames ...string) error {
	if env.JetStream == nil {
		return errors.New("JetStream not initialized")
	}
	for _, name := range streamNames {
		stream, err := env.JetStream.Stream(ctx, name)
		if err != nil {
			if strings.Contains(err.Error(), "stream not found") {
				log.Printf("Stream %q not found, skipping purge.", name)
				continue
			}
			log.Printf("Error accessing stream %q: %v", name, err)
			continue
		}
		if err := stream.Purge(ctx); err != nil {
			log.Printf("Failed to purge stream %q: %v", name, err)
		}
	}
	return nil
}

// DeleteJetStreamConsumers deletes all consumers for each specified stream.
func (env *TestEnvironment) DeleteJetStreamConsumers(ctx context.Context, streamNames ...string) error {
	if env.JetStream == nil {
		return errors.New("JetStream not initialized")
	}
	for _, name := range streamNames {
		stream, err := env.JetStream.Stream(ctx, name)
		if err != nil {
			log.Printf("Stream %q not accessible: %v", name, err)
			continue
		}
		consumers := stream.ListConsumers(ctx)
		for ci := range consumers.Info() {
			if ci == nil {
				continue
			}
			if err := env.JetStream.DeleteConsumer(ctx, name, ci.Name); err != nil {
				log.Printf("Failed to delete consumer %q from stream %q: %v", ci.Name, name, err)
			}
		}
		if err := consumers.Err(); err != nil {
			log.Printf("Error listing consumers for stream %q: %v", name, err)
		}
	}
	return nil
}

// ResetJetStreamState clears all JetStream streams and consumers
func (env *TestEnvironment) ResetJetStreamState(ctx context.Context, streamNames ...string) error {
	if env.JetStream == nil {
		return fmt.Errorf("JetStream context is nil")
	}

	for _, streamName := range streamNames {
		// Delete the stream if it exists (this also deletes all consumers)
		if err := env.JetStream.DeleteStream(ctx, streamName); err != nil {
			// Ignore "stream not found" errors
			if !isStreamNotFoundError(err) {
				log.Printf("Warning: failed to delete stream %s: %v", streamName, err)
			}
		}
	}

	return nil
}

// isStreamNotFoundError checks if the error indicates a stream was not found
func isStreamNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for JetStream API error codes
	if jsErr, ok := err.(jetstream.JetStreamError); ok {
		return jsErr.APIError().ErrorCode == 10059 // Stream not found error code
	}

	// Check for common error messages
	errMsg := err.Error()
	return errMsg == "stream not found" ||
		errMsg == "nats: stream not found" ||
		errMsg == "stream does not exist"
}
