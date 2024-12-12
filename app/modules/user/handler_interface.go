package user

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandler defines the handlers for user-related events.
type UserHandler interface {
	HandleCreateUser(msg *message.Message) ([]*message.Message, error)
	HandleGetUser(msg *message.Message) ([]*message.Message, error)
	HandleUpdateUser(msg *message.Message) ([]*message.Message, error)
	HandleGetUserRole(msg *message.Message) ([]*message.Message, error)
}
