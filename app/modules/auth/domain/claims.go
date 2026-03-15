package authdomain

import (
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/google/uuid"
)

// ClubRole represents a user's role within a specific club.
type ClubRole struct {
	ClubUUID    uuid.UUID `json:"club_uuid"`
	Role        Role      `json:"role"`
	DisplayName string    `json:"display_name,omitempty"` // Resolved display name (club nickname or fallback)
}

// Claims represents the domain model for authentication claims.
type Claims struct {
	UserID                 string // Legacy Discord User ID (kept for compatibility)
	UserUUID               uuid.UUID
	ActiveClubUUID         uuid.UUID
	ActiveClubEntitlements guildtypes.ResolvedClubEntitlements
	Clubs                  []ClubRole
	GuildID                string   // Legacy Discord Guild ID (kept for compatibility)
	Role                   Role     // Legacy Role (kept for compatibility, refers to ActiveClubUUID)
	LinkedProviders        []string // OAuth providers linked to this account (e.g. ["discord", "google"])
	RefreshTokenHash       string   // Hash of the refresh token used to mint this ticket
	ExpiresAt              time.Time
	IssuedAt               time.Time
}

// IsExpired checks if the claims have expired.
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
