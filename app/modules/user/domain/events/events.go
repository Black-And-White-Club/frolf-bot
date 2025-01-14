package events

import (
	usertypes "github.com/Black-And-White-Club/tcr-bot/app/modules/user/domain/types"
)

// Stream names
const (
	UserStreamName        = "user"
	LeaderboardStreamName = "leaderboard"
)

// User-related events (published to the user stream)
const (
	UserSignupRequest      = "user.signup.request"
	UserSignupResponse     = "user.signup.response"
	UserCreated            = "user.created"
	UserRoleUpdateRequest  = "user.role.update.request"
	UserRoleUpdateResponse = "user.role.update.response"
	UserRoleUpdated        = "user.role.updated"
	GetUserRoleRequest     = "user.get.role.request"
	GetUserRoleResponse    = "user.get.role.response"
	GetUserRequest         = "user.get.request"
	GetUserResponse        = "user.get.response"
)

// Leaderboard-related events (used by user module, published to the leaderboard stream)
const (
	CheckTagAvailabilityRequest  = "leaderboard.check.tag.availability.request"
	CheckTagAvailabilityResponse = "leaderboard.check.tag.availability.response"
	TagAssignedRequest           = "leaderboard.tag.assigned.request"
	TagAssignedResponse          = "leaderboard.tag.assigned.response"
)

// User Events Payloads
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

// Leaderboard Request Payloads (initiated by the user module)
type CheckTagAvailabilityRequestPayload struct {
	TagNumber int `json:"tag_number"`
}

type TagAssignedRequestPayload struct {
	DiscordID usertypes.DiscordID `json:"discord_id"`
	TagNumber int                 `json:"tag_number"`
}

// Leaderboard Response Payloads (received by the user module from the leaderboard module)
type CheckTagAvailabilityResponsePayload struct {
	IsAvailable bool   `json:"is_available"`
	Error       string `json:"error,omitempty"`
}

type TagAssignedResponsePayload struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
