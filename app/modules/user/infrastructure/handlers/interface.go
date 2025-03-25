package userhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface defines the contract for user-related message handlers.
type Handlers interface {
	HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error)
	HandleUserRoleUpdateRequest(msg *message.Message) ([]*message.Message, error)
	HandleGetUserRequest(msg *message.Message) ([]*message.Message, error)
	HandleGetUserRoleRequest(msg *message.Message) ([]*message.Message, error)
	HandleTagUnavailable(msg *message.Message) ([]*message.Message, error)
	HandleTagAvailable(msg *message.Message) ([]*message.Message, error)
}
