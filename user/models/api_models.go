// user/models/api_models.go
package userapimodels

// CreateUserCommand represents the request body for creating a user.
type CreateUserCommand struct {
	DiscordID string `json:"discord_id"`
	Role      string `json:"role"`
	TagNumber int    `json:"tag_number"`
}

// UpdateUserCommand represents the request body for updating a user.
type UpdateUserCommand struct {
	DiscordID string                 `json:"discord_id"`
	Role      string                 `json:"role"`
	TagNumber int                    `json:"tagNumber"`
	Updates   map[string]interface{} `json:"updates"`
}

// CreateUserRequest represents the request body for creating a user.
type CreateUserRequest struct {
	Name      string `json:"name"`
	DiscordID string `json:"discord_id"`
	Role      string `json:"role"`
}

// UserRole represents the role of a user.
type UserRole string

// Constants for user roles
const (
	UserRoleMember UserRole = "member"
	UserRoleEditor UserRole = "editor"
	UserRoleAdmin  UserRole = "admin"
)
