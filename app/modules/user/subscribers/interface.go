package usersubscribers

import (
	"context"

	userhandlers "github.com/Black-And-White-Club/tcr-bot/app/modules/user/handlers"
	"github.com/nats-io/nats.go"
)

// UserEventSubscriber defines the interface for subscribing to user events.
type UserEventSubscriber interface {
	SubscribeToUserEvents(ctx context.Context, js nats.JetStreamContext, handlers userhandlers.UserHandlers) error
}
