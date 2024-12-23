package userevents

import userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"

const (
	UserStream = "user" // Stream for all user-related events

	// User Signup Events
	UserSignupRequestSubject  = "user.signup.request"  // Subject for requesting a new user signup
	UserSignupResponseSubject = "user.signup.response" // Subject for the response to a signup request
	UserCreatedSubject        = "user.created"         // Subject for indicating a new user was created

	// User Role Update Events
	UserRoleUpdateRequestSubject  = "user.role.update.request"  // Subject for requesting a user role update
	UserRoleUpdateResponseSubject = "user.role.update.response" // Subject for the response to a role update request
	UserRoleUpdatedSubject        = "user.role.updated"         // Subject for indicating a user's role was updated

	// User Data Retrieval Events
	GetUserRoleRequestSubject  = "user.role.get.request"  // Subject for requesting a user's role
	GetUserRoleResponseSubject = "user.role.get.response" // Subject for the response to a role retrieval request
	GetUserRequestSubject      = "user.get.request"       // Subject for requesting user data
	GetUserResponseSubject     = "user.get.response"      // Subject for the response to a user data retrieval request

	// Leaderboard Interaction Events - Subjects used by the USER module
	LeaderboardStream                  = "leaderboard"                                // Stream for leaderboard-related events
	CheckTagAvailabilityRequestSubject = "leaderboard.check.tag.availability.request" // Subject for checking tag availability (published by user module)
	TagAssignedSubject                 = "leaderboard.tag.assigned"                   // Subject for indicating a tag was assigned
	CheckTagAvailabilityResponseTopic  = "leaderboard.check.tag.availability.response"
)

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

// CheckTagAvailabilityRequest represents an event to check if a tag number is available.
type CheckTagAvailabilityRequest struct {
	TagNumber int `json:"tag_number"`
}

// CheckTagAvailabilityResponse represents an event indicating whether a tag is available.
type CheckTagAvailabilityResponse struct {
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
