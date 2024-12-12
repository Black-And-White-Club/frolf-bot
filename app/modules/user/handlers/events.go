package userhandlers

import userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"

type CreateUserRequest struct {
	DiscordID string `json:"discord_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	TagNumber int    `json:"tag_number"`
}

// UserCreatedEvent represents an event triggered when a user is created.
type UserCreatedEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// UserUpdatedEvent represents an event triggered when a user is updated.
type UserUpdatedEvent struct {
	DiscordID string `json:"discord_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
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
	DiscordID string `json:"discord_id"`
	Role      string `json:"role"`
}

// GetUserRequest represents the request to get a user.
type GetUserRequest struct {
	DiscordID string `json:"discord_id"`
}

// GetUserResponse represents the response to a GetUserRequest.
type GetUserResponse struct {
	User userdb.User `json:"user"`
}

// UpdateUserRequest represents the request to update a user.
type UpdateUserRequest struct {
	DiscordID string                 `json:"discord_id"`
	Updates   map[string]interface{} `json:"updates"`
}

// UserRole represents the role of a user.
type UserRole string

// Constants for user roles
const (
	UserRoleRattler UserRole = "Rattler"
	UserRoleEditor  UserRole = "Editor"
	UserRoleAdmin   UserRole = "Admin"
)
