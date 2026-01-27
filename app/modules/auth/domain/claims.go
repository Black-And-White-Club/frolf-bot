package authdomain

import "time"

// Claims represents the domain model for authentication claims.
type Claims struct {
	UserID    string
	GuildID   string
	Role      Role
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// IsExpired checks if the claims have expired.
func (c *Claims) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}
