package leaderboardtypes

import (
	"fmt"
	"regexp"
)

// LeaderboardData represents the data of a leaderboard.
type LeaderboardData map[int]string

// Tag represents a tag number.
type Tag int

// IsValid checks if the tag number is valid (e.g., within a certain range).
func (t Tag) IsValid() bool {
	// Implement your validation logic here
	return t > 0 && t <= 200
}

// String returns the string representation of the tag.
func (t Tag) String() string {
	return fmt.Sprintf("%d", t)
}

// DiscordID defines a custom type for Discord IDs.
type DiscordID string

var discordIDRegex = regexp.MustCompile(`^[0-9]+$`) // Matches one or more digits

// IsValid checks if the DiscordID is valid (contains only numbers).
func (id DiscordID) IsValid() bool {
	return discordIDRegex.MatchString(string(id))
}

// User interface
type User interface {
	GetID() int64
	// GetName() string
	GetDiscordID() DiscordID
}
