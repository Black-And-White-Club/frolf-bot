package authdomain

import (
	"time"

	"github.com/google/uuid"
)

// ClubRole represents a user's role within a specific club.
type ClubRole struct {
	ClubUUID uuid.UUID
	Role     Role
}

// Claims represents the domain model for authentication claims.
type Claims struct {
	UserID         string // Legacy Discord User ID (kept for compatibility)
	UserUUID       uuid.UUID
	ActiveClubUUID uuid.UUID
	Clubs          []ClubRole
	GuildID        string // Legacy Discord Guild ID (kept for compatibility)
	Role           Role   // Legacy Role (kept for compatibility, refers to ActiveClubUUID)
	ExpiresAt      time.Time
	IssuedAt       time.Time
}

// IsExpired checks if the claims have expired.
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
