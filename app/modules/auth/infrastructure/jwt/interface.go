package authjwt

import (
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
)

// Provider defines the interface for JWT token operations.
type Provider interface {
	// GenerateToken creates a signed JWT token from the given claims.
	GenerateToken(claims *authdomain.Claims, ttl time.Duration) (string, error)

	// ValidateToken validates a JWT token and returns the claims if valid.
	ValidateToken(tokenString string) (*authdomain.Claims, error)
}
