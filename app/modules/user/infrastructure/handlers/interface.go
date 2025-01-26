package userhandlers

import (
	"github.com/ThreeDotsLabs/watermill/message"
)

// Handlers interface defines the contract for user-related message handlers.
type Handlers interface {
	HandleUserSignupRequest(msg *message.Message) error
	HandleUserRoleUpdateRequest(msg *message.Message) error
	HandleUserPermissionsCheckResponse(msg *message.Message) error
	HandleUserPermissionsCheckRequest(msg *message.Message) error
	HandleUserPermissionsCheckFailed(msg *message.Message) error
	HandleGetUserRequest(msg *message.Message) error
	HandleGetUserRoleRequest(msg *message.Message) error
	HandleTagUnavailable(msg *message.Message) error
	HandleTagAvailable(msg *message.Message) error
}
