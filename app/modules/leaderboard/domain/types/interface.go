package leaderboardtypes

import "fmt"

// Tag represents a tag number.
type Tag int

// IsValid checks if the tag number is valid (e.g., within a certain range).
func (t Tag) IsValid() bool {
	// Implement your validation logic here
	return t > 0 && t <= 100
}

// String returns the string representation of the tag.
func (t Tag) String() string {
	return fmt.Sprintf("%d", t)
}
