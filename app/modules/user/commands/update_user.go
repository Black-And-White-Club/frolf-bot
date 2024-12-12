package usercommands

// UpdateUserRequest represents the request to update a user.
type UpdateUserRequest struct {
	DiscordID string                 `json:"discord_id"`
	Updates   map[string]interface{} `json:"updates"`
}

// CommandName returns the command name for UpdateUserRequest
func (cmd UpdateUserRequest) CommandName() string {
	return "update_user"
}

var _ Command = UpdateUserRequest{}
