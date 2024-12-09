// user/eventhandling/events.go
package usereventhandling

import (
	"context"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/nats"
)

// PublishUserRegistered publishes a UserRegisteredEvent.
func PublishUserRegistered(ctx context.Context, natsConnectionPool *nats.NatsConnectionPool, event UserRegisteredEvent) error {
	if err := natsConnectionPool.Publish("user.registered", event); err != nil {
		return fmt.Errorf("failed to publish user.registered event: %w", err)
	}
	return nil
}

// PublishUserRoleUpdated publishes a UserRoleUpdatedEvent.
func PublishUserRoleUpdated(ctx context.Context, natsConnectionPool *nats.NatsConnectionPool, event UserRoleUpdatedEvent) error {
	if err := natsConnectionPool.Publish("user.role.updated", event); err != nil {
		return fmt.Errorf("failed to publish user.role.updated event: %w", err)
	}
	return nil
}
