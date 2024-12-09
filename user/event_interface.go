// user/event_interface.go
package user

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/user/eventhandling"
)

// UserEventHandler defines an interface for handling user events.
type UserEventHandler interface {
	HandleUserCreated(ctx context.Context, event eventhandling.UserCreatedEvent) error
	HandleUserUpdated(ctx context.Context, event eventhandling.UserUpdatedEvent) error
	HandleCheckTagAvailability(ctx context.Context, event eventhandling.CheckTagAvailabilityEvent) error
	HandleGetUserRole(ctx context.Context, event eventhandling.GetUserRoleEvent) error
}
