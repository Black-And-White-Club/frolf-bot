package userevents

import userdb "github.com/Black-And-White-Club/tcr-bot/app/modules/user/db"

// User Events
const (
	UserStream = "user" // Stream for all user-related events

	// User Signup Events
	UserSignupRequestSubject  = UserStream + ".signup.request"  // Subject for requesting a new user signup
	UserSignupResponseSubject = UserStream + ".signup.response" // Subject for the response to a signup request
	UserCreatedSubject        = UserStream + ".created"         // Subject for indicating a new user was created

	// User Role Update Events
	UserRoleUpdateRequestSubject  = UserStream + ".role.update.request"  // Subject for requesting a user role update
	UserRoleUpdateResponseSubject = UserStream + ".role.update.response" // Subject for the response to a role update request
	UserRoleUpdatedSubject        = UserStream + ".role.updated"         // Subject for indicating a user's role was updated

	// User Data Retrieval Events
	GetUserRoleRequestSubject  = UserStream + ".role.get.request"  // Subject for requesting a user's role
	GetUserRoleResponseSubject = UserStream + ".role.get.response" // Subject for the response to a role retrieval request
	GetUserRequestSubject      = UserStream + ".get.request"       // Subject for requesting user data
	GetUserResponseSubject     = UserStream + ".get.response"      // Subject for the response to a user data retrieval request

	// Leaderboard Interaction Events
	LeaderboardStream                   = "leaderboard"                             // Stream for leaderboard-related events
	CheckTagAvailabilityRequestSubject  = LeaderboardStream + ".tag.check.request"  // Subject for checking tag availability
	CheckTagAvailabilityResponseSubject = LeaderboardStream + ".tag.check.response" // Subject for the response to tag availability check
	TagAssignedSubject                  = LeaderboardStream + ".tag.assigned"       // Subject for indicating a tag was assigned
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
