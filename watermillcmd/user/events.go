package userhandlers

// UserCreatedEvent represents an event triggered when a user is created.
type UserCreatedEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// UserUpdatedEvent represents an event triggered when a user is updated.
type UserUpdatedEvent struct {
	DiscordID string `json:"discord_id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
}

// CheckTagAvailabilityEvent represents an event to check if a tag number is available.
type CheckTagAvailabilityEvent struct {
	TagNumber int `json:"tag_number"`
}

// TagAvailabilityResponseEvent represents an event indicating whether a tag is available.
type TagAvailabilityResponseEvent struct {
	IsAvailable bool `json:"is_available"`
}

// GetUserRoleEvent represents an event to retrieve the role of a user.
type GetUserRoleEvent struct {
	DiscordID string `json:"discord_id"`
}

// UserRoleResponseEvent represents an event containing the role of a user.
type UserRoleResponseEvent struct {
	Role string `json:"role"`
}
