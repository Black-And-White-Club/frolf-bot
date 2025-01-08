package userinterfaces

import (
	"context"

	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// Handlers interface
type Handlers interface {
	HandleUserSignupRequest(ctx context.Context, msg shared.Message) error     // Updated to use *message.Message
	HandleUserRoleUpdateRequest(ctx context.Context, msg shared.Message) error // Updated to use *message.Message
}
