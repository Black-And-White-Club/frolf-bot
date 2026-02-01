package authnats

import "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"

// UserJWTBuilder defines the interface for building NATS user JWTs.
type UserJWTBuilder interface {
	// BuildUserJWT creates a NATS user JWT with the specified permissions.
	BuildUserJWT(userID, guildID string, perms *permissions.Permissions) (string, error)
}
