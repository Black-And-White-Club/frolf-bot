// user/event_interface.go
package user

import (
	"context"

	usereventhandler "github.com/Black-And-White-Club/tcr-bot/user/eventhandling"
)

// UserEventHandler defines an interface for handling user events.
type UserEventHandler interface {
	HandleUserRegistered(ctx context.Context, event usereventhandler.UserRegisteredEvent) error
	HandleUserRoleUpdated(ctx context.Context, event usereventhandler.UserRoleUpdatedEvent) error
}
