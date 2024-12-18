package userevents

import userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"

// UserSignupRequest represents an event requesting a new user signup.
type UserSignupRequest struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// UserSignupResponse represents an event responding to a user signup request.
type UserSignupResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UserCreated represents an event triggered when a new user is created.
type UserCreated struct {
	DiscordID string          `json:"discord_id"`
	TagNumber int             `json:"tag_number"`
	Role      userdb.UserRole `json:"role"`
}

// UserRoleUpdateRequest represents an event requesting a user's role to be updated.
type UserRoleUpdateRequest struct {
	DiscordID string          `json:"discord_id"`
	NewRole   userdb.UserRole `json:"new_role"`
}

// UserRoleUpdateResponse represents an event responding to a user role update request.
type UserRoleUpdateResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// UserUpdated represents an event triggered when a user's information is updated.
type UserUpdated struct {
	DiscordID   string          `json:"discord_id"`
	NewUsername string          `json:"new_username"`
	NewRole     userdb.UserRole `json:"new_role"`
	// ... other updated fields with their new values
}

// GetUserRoleRequest represents an event requesting the role of a user.
type GetUserRoleRequest struct {
	DiscordID string `json:"discord_id"`
}

// GetUserRoleResponse represents an event containing the role of a user.
type GetUserRoleResponse struct {
	DiscordID string          `json:"discord_id"`
	Role      userdb.UserRole `json:"role"`
	Error     string          `json:"error,omitempty"` // Include error information if needed
}

// GetUserRequest represents an event to request user data.
type GetUserRequest struct {
	DiscordID string `json:"discord_id"`
}

// GetUserResponse represents an event containing user data.
type GetUserResponse struct {
	User  userdb.User `json:"user"`
	Error string      `json:"error,omitempty"` // Include error information if needed
}

// Leaderboard Events (defined within the user module)

const (
	CheckTagAvailabilityRequestSubject  = "user.check_tag_availability_request"
	CheckTagAvailabilityResponseSubject = "user.check_tag_availability_response"
	TagAssignedSubject                  = "user.tag_assigned"
	UserRoleUpdatedSubject              = "user.user_role_updated"
	LeaderboardStream                   = "leaderboard"
	UserSignupResponseSubject           = "user.user_signup_response"
	UserRoleUpdateResponseSubject       = "user.user_role_update_response"
	UserStream                          = "user"
	UserSignupRequestSubject            = "user.user_signup_request"
	UserRoleUpdateRequestSubject        = "user.user_role_update_request"
)

// CheckTagAvailabilityRequest represents an event to check if a tag number is available.
type CheckTagAvailabilityRequest struct {
	TagNumber int `json:"tag_number"`
}

// CheckTagAvailabilityResponse represents an event indicating whether a tag is available.
type CheckTagAvailabilityResponse struct {
	TagNumber   int  `json:"tag_number"`
	IsAvailable bool `json:"is_available"`
}

// TagAssigned represents an event indicating that a tag has been assigned to a user.
type TagAssigned struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// UserRoleUpdated represents an event indicating that a user's role has been updated.
type UserRoleUpdated struct {
	DiscordID string          `json:"discord_id"`
	NewRole   userdb.UserRole `json:"new_role"`
}
