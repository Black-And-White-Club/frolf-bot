package authjwt

import (
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
)

// Provider defines the interface for JWT token operations.
type Provider interface {
	// GenerateToken creates a signed JWT token for the given user, guild, and role.
	GenerateToken(userID, guildID string, role authdomain.Role, ttl time.Duration) (string, error)

	// ValidateToken validates a JWT token and returns the claims if valid.
	ValidateToken(tokenString string) (*authdomain.Claims, error)
}
