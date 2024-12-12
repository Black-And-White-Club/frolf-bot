package userhandlers

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// CreateUserCommandHandler handles the CreateUserCommand.
type CreateUserCommandHandler interface {
	Handle(ctx context.Context, cmd *CreateUserRequest) error
	checkTagAvailability(ctx context.Context, tagNumber int) (bool, error)
}

// UpdateUserCommandHandler handles the UpdateUserCommand.
type UpdateUserCommandHandler interface {
	Handle(ctx context.Context, cmd *UpdateUserRequest) error
}

// GetUserRoleHandler handles requests for user roles.
type GetUserRoleQueryHandler interface {
	Handle(msg *message.Message) error
}
