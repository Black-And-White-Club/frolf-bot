package userevents

import (
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
	"github.com/Black-And-White-Club/tcr-bot/app/shared"
)

// EventType variables for user events
var (
	UserSignupRequest  = shared.EventType{Module: "user", Name: "signup.request"}
	UserSignupResponse = shared.EventType{Module: "user", Name: "signup.response"}
	UserCreated        = shared.EventType{Module: "user", Name: "created"}

	UserRoleUpdateRequest  = shared.EventType{Module: "user", Name: "role.update.request"}
	UserRoleUpdateResponse = shared.EventType{Module: "user", Name: "role.update.response"}
	UserRoleUpdated        = shared.EventType{Module: "user", Name: "role.updated"}

	GetUserRoleRequest  = shared.EventType{Module: "user", Name: "role.get.request"}
	GetUserRoleResponse = shared.EventType{Module: "user", Name: "role.get.response"}
	GetUserRequest      = shared.EventType{Module: "user", Name: "get.request"}
	GetUserResponse     = shared.EventType{Module: "user", Name: "get.response"}
)

// EventType variables for leaderboard events (used by the user module)
var (
	CheckTagAvailabilityRequest  = shared.EventType{Module: "leaderboard", Name: "check.tag.availability.request"}
	CheckTagAvailabilityResponse = shared.EventType{Module: "leaderboard", Name: "check.tag.availability.response"}
	TagAssignedRequest           = shared.EventType{Module: "leaderboard", Name: "tag.assigned.request"}
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
	IsAvailable bool   `json:"is_available"`
	Error       string `json:"error,omitempty"`
}

type TagAssignedRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number"`
}
