package userevents

import (
	"github.com/Black-And-White-Club/tcr-bot/app/events"
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// EventType variables for user events
var (
	UserSignupRequest  = events.EventType{Module: "user", Name: "signup.request"}
	UserSignupResponse = events.EventType{Module: "user", Name: "signup.response"}
	UserCreated        = events.EventType{Module: "user", Name: "created"}

	UserRoleUpdateRequest  = events.EventType{Module: "user", Name: "role.update.request"}
	UserRoleUpdateResponse = events.EventType{Module: "user", Name: "role.update.response"}
	UserRoleUpdated        = events.EventType{Module: "user", Name: "role.updated"}

	GetUserRoleRequest  = events.EventType{Module: "user", Name: "role.get.request"}
	GetUserRoleResponse = events.EventType{Module: "user", Name: "role.get.response"}
	GetUserRequest      = events.EventType{Module: "user", Name: "get.request"}
	GetUserResponse     = events.EventType{Module: "user", Name: "get.response"}
)

// EventType variables for leaderboard events (used by the user module)
var (
	CheckTagAvailabilityRequest  = events.EventType{Module: "leaderboard", Name: "check.tag.availability.request"}
	CheckTagAvailabilityResponse = events.EventType{Module: "leaderboard", Name: "check.tag.availability.response"}
	TagAssignedRequest           = events.EventType{Module: "leaderboard", Name: "tag.assigned.request"}
)

// User Events
type UserSignupRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number,omitempty"`
}

type UserSignupResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type UserCreatedPayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	TagNumber int                    `json:"tag_number,omitempty"`
	Role      usertypes.UserRoleEnum `json:"role"`
}

type UserRoleUpdateRequestPayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	NewRole   usertypes.UserRoleEnum `json:"new_role"`
}

type UserRoleUpdateResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

type UserRoleUpdatedPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	NewRole   string              `json:"new_role"`
}

type GetUserRoleRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
}

type GetUserRoleResponsePayload struct {
	DiscordID usertypes.DiscordID    `json:"discord_id"`
	Role      usertypes.UserRoleEnum `json:"role"`
	Error     string                 `json:"error,omitempty"`
}

type GetUserRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
}

type GetUserResponsePayload struct {
	User  usertypes.User `json:"user"`
	Error string         `json:"error,omitempty"`
}

// Leaderboard Events
type CheckTagAvailabilityRequestPayload struct {
	TagNumber int `json:"tag_number"`
}

type CheckTagAvailabilityResponsePayload struct {
	IsAvailable bool `json:"is_available"`
}

type TagAssignedRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number"`
}
