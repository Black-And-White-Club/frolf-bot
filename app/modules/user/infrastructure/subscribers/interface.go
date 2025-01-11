package usersubscribers

import (
	"context"
	"log/slog"

	user "github.com/Black-And-White-Club/tcr-bot/app/modules/user/interfaces"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// UserEventSubscriber defines the interface for subscribing to user events.
type UserEventSubscriber interface {
	SubscribeToUserEvents(ctx context.Context, eventBus shared.EventBus, handlers user.Handlers, logger *slog.Logger) error
}
