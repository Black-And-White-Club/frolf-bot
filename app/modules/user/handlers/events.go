package userhandlers

import userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"

// UserCreatedEvent represents an event triggered when a user is created.
type UserCreatedEvent struct {
	DiscordID string          `json:"discord_id"`
	TagNumber int             `json:"tag_number"`
	Role      userdb.UserRole `json:"role"`
}

// UserUpdatedEvent represents an event triggered when a user is updated.
type UserUpdatedEvent struct {
	DiscordID     string   `json:"discord_id"`
	UpdatedFields []string `json:"updated_fields"`
}

// CheckTagAvailabilityEvent represents an event to check if a tag number is available.
type CheckTagAvailabilityEvent struct {
	TagNumber int `json:"tag_number"`
}

// TagAvailabilityResponseEvent represents an event indicating whether a tag is available.
type TagAvailabilityResponseEvent struct {
	TagNumber   int  `json:"tag_number"`
	IsAvailable bool `json:"is_available"`
}

// GetUserRoleRequestEvent represents an event to request the role of a user.
type GetUserRoleRequestEvent struct {
	DiscordID string `json:"discord_id"`
}

// UserRoleResponseEvent represents an event containing the role of a user.
type UserRoleResponseEvent struct {
	DiscordID string          `json:"discord_id"`
	Role      userdb.UserRole `json:"role"`
}

// GetUserResponse represents the response to a GetUserRequest.
type GetUserResponse struct {
	User userdb.User `json:"user"`
}
