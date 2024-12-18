package userhandlers

import "github.com/ThreeDotsLabs/watermill/message"

// UserHandlers handles user-related events.
type Handlers interface {
	HandleUserSignupRequest(msg *message.Message) error
	HandleUserRoleUpdateRequest(msg *message.Message) error
}
