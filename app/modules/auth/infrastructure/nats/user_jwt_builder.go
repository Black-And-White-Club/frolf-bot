package authnats

import (
	"fmt"
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	"github.com/nats-io/nkeys"
)

// userJWTBuilder implements the UserJWTBuilder interface.
type userJWTBuilder struct {
	signingKey    nkeys.KeyPair
	issuerAccount string
}

// NewUserJWTBuilder creates a new UserJWTBuilder.
func NewUserJWTBuilder(signingKey nkeys.KeyPair, issuerAccount string) UserJWTBuilder {
	return &userJWTBuilder{
		signingKey:    signingKey,
		issuerAccount: issuerAccount,
	}
}

// BuildUserJWT creates a NATS user JWT with the specified permissions.
func (b *userJWTBuilder) BuildUserJWT(claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
	// Get public key from signing key
	publicKey, err := b.signingKey.PublicKey()
	if err != nil {
		return "", fmt.Errorf("failed to get public key: %w", err)
	}

	// Create user claims for NATS
	uc := NewUserClaims(publicKey)
	uc.Name = fmt.Sprintf("%s@%s", claims.UserUUID.String(), claims.ActiveClubUUID.String())
	uc.Audience = b.issuerAccount
	uc.Expires = time.Now().Add(24 * time.Hour).Unix()
	uc.RefreshTokenHash = claims.RefreshTokenHash

	// Set permissions
	uc.Permissions.Pub.Allow = perms.Publish.Allow
	uc.Permissions.Pub.Deny = perms.Publish.Deny
	uc.Permissions.Sub.Allow = perms.Subscribe.Allow
	uc.Permissions.Sub.Deny = perms.Subscribe.Deny

	// Encode and sign the JWT
	token, err := uc.Encode(b.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode user claims: %w", err)
	}

	return token, nil
}
