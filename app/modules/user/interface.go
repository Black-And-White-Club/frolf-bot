package userhandlers

import "github.com/ThreeDotsLabs/watermill/message"

// In watermillcmd/user/create_user_handler.go

type CreateUserCommandHandler interface {
	Handle(msg *message.Message) error
	checkTagAvailability(tagNumber int) (bool, error) // Add this method
}

// In watermillcmd/user/update_user_handler.go

type UpdateUserCommandHandler interface {
	Handle(msg *message.Message) error
}
