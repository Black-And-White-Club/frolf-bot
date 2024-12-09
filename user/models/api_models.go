// user/models/api_models.go
package userapimodels

// CreateUserRequest represents the request body for creating a user.
type CreateUserRequest struct {
	Name      string `json:"name"`
	DiscordID string `json:"discordID"`
	Role      string `json:"role"`
}

// UpdateUserRequest represents the request body for updating a user.
type UpdateUserRequest struct {
	Name string `json:"name"`
	Role string `json:"role"`
}
