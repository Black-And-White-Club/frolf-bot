package authnats

import (
	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
)

// UserJWTBuilder defines the interface for building NATS user JWTs.
type UserJWTBuilder interface {
	// BuildUserJWT creates a NATS user JWT with the specified permissions.
	BuildUserJWT(userNkey string, claims *authdomain.Claims, perms *permissions.Permissions) (string, error)

	// BuildAuthResponse creates a signed authorization response JWT for auth callout.
	// The audience should be the server's public key from the auth request.
	// The subject should be the user's public key (ClientNKey).
	BuildAuthResponse(audience string, subject string, userJWT string, errMsg string) (string, error)
}
