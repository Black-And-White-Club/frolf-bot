package userhandlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface to uncouple handlers from specific implementations.
type Handlers interface {
	HandleUserSignupRequest(ctx context.Context, msg *message.Message) error
	HandleUserRoleUpdateRequest(ctx context.Context, msg *message.Message) error
}
