package nats

import (
	"github.com/Black-And-White-Club/tcr-bot/api/structs"
	"github.com/Black-And-White-Club/tcr-bot/models"
)

// CheckTagAvailabilityEvent is triggered to check if a tag number is available.
type CheckTagAvailabilityEvent struct {
	TagNumber int    `json:"tag_number"`
	ReplyTo   string `json:"reply_to"`
}

// TagAvailabilityResponse is sent back with the tag availability status.
type TagAvailabilityResponse struct {
	IsAvailable bool `json:"is_available"`
}

// ScoresProcessedEvent is triggered when the Score module finishes processing scores.
type ScoresProcessedEvent struct {
	RoundID         int64                `json:"round_id"`
	ProcessedScores []structs.ScoreInput `json:"processed_scores"`
}

// UserUpdatedEvent is sent when a user's details are updated.
type UserUpdatedEvent struct {
	DiscordID string          `json:"discord_id"`
	Name      string          `json:"name"`
	Role      models.UserRole `json:"role"`
}

// UserCreatedEvent is sent when a new user is created with a tag number.
type UserCreatedEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"`
}

// RoundFinalizedEvent is sent when a round is finalized.
type RoundFinalizedEvent struct {
	RoundID int64 `json:"round_id"`
}

// UserGetRoleEvent is sent to retrieve the role of a user.
type UserGetRoleEvent struct {
	DiscordID string `json:"discord_id"`
	ReplyTo   string `json:"reply_to"`
}

// UserGetRoleResponse is sent back with the user's role.
type UserGetRoleResponse struct {
	Role models.UserRole `json:"role"`
}

// LeaderboardGetTagNumberEvent is sent to retrieve the tag number of a user.
type LeaderboardGetTagNumberEvent struct {
	DiscordID string `json:"discord_id"`
	TagNumber int    `json:"tag_number"` // Added tagNumber field
	ReplyTo   string `json:"reply_to"`
}

// LeaderboardGetTagNumberResponse is sent back with the user's tag number and availability.
type LeaderboardGetTagNumberResponse struct {
	IsAvailable bool `json:"is_available"` // Added IsAvailable field
	TagNumber   *int `json:"tag_number"`
}
