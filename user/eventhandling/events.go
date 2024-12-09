// user/eventhandling/events.go
package usereventhandling

// UserRegisteredEvent is triggered when a new user registers with the bot.
type UserRegisteredEvent struct {
	DiscordID string `json:"discord_id"`
}

// Topic returns the topic name for the UserRegisteredEvent.
func (e UserRegisteredEvent) Topic() string {
	return "user.registered"
}

// UserRoleUpdatedEvent is triggered when a user's role is updated.
type UserRoleUpdatedEvent struct {
	DiscordID string `json:"discord_id"`
	NewRole   string `json:"new_role"`
}

// Topic returns the topic name for the UserRoleUpdatedEvent.
func (e UserRoleUpdatedEvent) Topic() string {
	return "user.role.updated"
}
