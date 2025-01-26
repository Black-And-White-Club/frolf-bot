package userevents

import (
	"time"

	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// Stream names
const (
	UserStreamName = "user"
	// Note: Changed to use the request/response pattern consistently
	LeaderboardTagAvailabilityCheckRequest = "leaderboard.tag.availability.check.request"
	UserCreatedDLQ                         = "user.created.dlq"            // DLQ for UserCreated
	UserCreationFailedDLQ                  = "user.creation.failed.dlq"    // DLQ for UserCreationFailed
	UserRoleUpdatedDLQ                     = "user.role.updated.dlq"       // DLQ for UserRoleUpdated
	UserRoleUpdateFailedDLQ                = "user.role.update.failed.dlq" // DLQ for UserRoleUpdateFailed
	GetUserRoleResponseDLQ                 = "user.get.role.response.dlq"  // DLQ for GetUserRoleResponse
	GetUserResponseDLQ                     = "user.get.response.dlq"       // DLQ for GetUserResponse
	GetUserRoleFailedDLQ                   = "user.get.role.failed.dlq"    // DLQ for GetUserRoleFailed
	GetUserFailedDLQ                       = "user.get.failed.dlq"         // DLQ for GetUserFailed
)

// Event names
const (
	CreateUserRequested           = "user.create.requested"
	UserSignupRequest             = "user.signup.request"
	UserSignupFailed              = "user.signup.failed"
	UserCreated                   = "user.created"
	UserCreationFailed            = "user.creation.failed"
	UserRoleUpdateRequest         = "user.role.update.request"
	UpdateUserRoleRequested       = "user.role.update.requested"
	UserRoleUpdated               = "user.role.updated"
	UserRoleUpdateFailed          = "user.role.update.failed"
	GetUserRoleRequest            = "user.get.role.request"
	GetUserRoleResponse           = "user.get.role.response"
	GetUserRoleFailed             = "user.get.role.failed"
	GetUserRequest                = "user.get.request"
	GetUserResponse               = "user.get.response"
	GetUserFailed                 = "user.get.failed"
	UserPermissionsCheckRequest   = "user.permissions.check.request"
	UserPermissionsCheckResponse  = "user.permissions.check.response"
	UserPermissionsCheckFailed    = "user.permissions.check.failed"
	TagAvailable                  = "user.tag.available"   // ADDED
	TagUnavailable                = "user.tag.unavailable" // ADDED
	TagAvailabilityCheckRequested = "user.tag.availability.check.requested"
)

// BaseEventPayload is a struct that can be embedded in other event structs to provide common fields.
type BaseEventPayload struct {
	EventID   string    `json:"event_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Payload types

type CreateUserRequestedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber *int                `json:"tag_number"`
}

type UserSignupRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber *int                `json:"tag_number,omitempty"`
}

type UserSignupFailedPayload struct {
	Reason string `json:"reason"`
}

type UserCreatedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber *int                `json:"tag_number,omitempty"`
}

type UserCreationFailedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	Reason    string              `json:"reason"`
	TagNumber *int                `json:"tag_number,omitempty"`
}

type UserRoleUpdateRequestPayload struct {
	DiscordID   usertypes.DiscordID    `json:"discord_id"`
	Role        usertypes.UserRoleEnum `json:"role"`
	RequesterID string                 `json:"requester_id"`
}

type UpdateUserRoleRequestedPayload struct {
	DiscordID   usertypes.DiscordID    `json:"discord_id"`
	Role        usertypes.UserRoleEnum `json:"role"`
	RequesterID string                 `json:"requester_id"`
}

type UserRoleUpdatedPayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	Role      usertypes.UserRoleEnum `json:"role"`
}

type UserRoleUpdateFailedPayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	Role      usertypes.UserRoleEnum `json:"role"`
	Reason    string                 `json:"reason"`
}

type GetUserRoleRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
}

type GetUserRoleResponsePayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	Role      usertypes.UserRoleEnum `json:"role"`
}

type GetUserRoleFailedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	Reason    string              `json:"reason"`
}

type GetUserRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
}

type GetUserResponsePayload struct {
	User *usertypes.UserData `json:"user"`
}

type GetUserFailedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	Reason    string              `json:"reason"`
}

type UserPermissionsCheckRequestPayload struct {
	DiscordID   usertypes.DiscordID    `json:"discord_id"`
	Role        usertypes.UserRoleEnum `json:"role"`
	RequesterID string                 `json:"requester_id"`
}

type UserPermissionsCheckResponsePayload struct {
	HasPermission bool                   `json:"has_permission"`
	DiscordID     usertypes.DiscordID    `json:"discord_id"`
	Role          usertypes.UserRoleEnum `json:"role"`
	RequesterID   string                 `json:"requester_id"`
}

type UserPermissionsCheckFailedPayload struct {
	Reason      string                 `json:"reason"`
	DiscordID   usertypes.DiscordID    `json:"discord_id"`
	Role        usertypes.UserRoleEnum `json:"role"`
	RequesterID string                 `json:"requester_id"`
}

// TagAvailabilityCheckRequestedPayload is the payload for the TagAvailabilityCheckRequested event.
type TagAvailabilityCheckRequestedPayload struct {
	TagNumber int                 `json:"tag_number"`
	DiscordID usertypes.DiscordID `json:"discord_id"`
}

// TagAvailablePayload is the payload for the TagAvailable event.
type TagAvailablePayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number"`
}

// TagUnavailablePayload is the payload for the TagUnavailable event.
type TagUnavailablePayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number"`
	Reason    string              `json:"reason"`
}
