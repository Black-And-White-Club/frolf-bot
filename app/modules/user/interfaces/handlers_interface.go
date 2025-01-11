package userinterfaces

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface
type Handlers interface {
	HandleUserSignupRequest(ctx context.Context, msg *message.Message) error
	HandleUserRoleUpdateRequest(ctx context.Context, msg *message.Message) error
}
