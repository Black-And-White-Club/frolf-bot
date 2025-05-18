package testutils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
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

// ResetJetStreamState cleans both consumers and messages in one call.
func (env *TestEnvironment) ResetJetStreamState(ctx context.Context, streamNames ...string) error {
	if err := env.DeleteJetStreamConsumers(ctx, streamNames...); err != nil {
		log.Printf("Error cleaning consumers: %v", err)
	}
	if err := env.PurgeJetStreamStreams(ctx, streamNames...); err != nil {
		return fmt.Errorf("failed to purge streams: %w", err)
	}
	return nil
}
