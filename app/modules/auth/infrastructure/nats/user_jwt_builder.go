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
func (b *userJWTBuilder) BuildUserJWT(userNkey string, claims *authdomain.Claims, perms *permissions.Permissions) (string, error) {
	// Create user claims for NATS
	uc := NewUserClaims(userNkey)
	uc.Name = "frolf-pwa-user"

	// Ensure audience is a public key even if a seed was provided in config
	accountPubKey := b.issuerAccount
	if kp, err := nkeys.FromSeed([]byte(accountPubKey)); err == nil {
		if pub, err := kp.PublicKey(); err == nil {
			accountPubKey = pub
		}
	}
	uc.Audience = accountPubKey
	uc.Expires = time.Now().Add(24 * time.Hour).Unix()
	uc.RefreshTokenHash = claims.RefreshTokenHash

	// Set permissions
	uc.Nats.Pub.Allow = perms.Publish.Allow
	uc.Nats.Pub.Deny = perms.Publish.Deny
	uc.Nats.Sub.Allow = perms.Subscribe.Allow
	uc.Nats.Sub.Deny = perms.Subscribe.Deny

	// NOTE: Do NOT set uc.Nats.IssuerAccount in non-operator mode (simple config file mode).
	// The issuer_account claim is only valid in decentralized operator-based auth.

	// Encode and sign the JWT
	token, err := uc.Encode(b.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode user claims: %w", err)
	}

	return token, nil
}

// BuildAuthResponse creates a signed authorization response JWT for auth callout.
func (b *userJWTBuilder) BuildAuthResponse(audience string, subject string, userJWT string, errMsg string) (string, error) {
	// Ensure issuer account is a public key even if a seed was provided in config
	accountPubKey := b.issuerAccount
	if kp, err := nkeys.FromSeed([]byte(accountPubKey)); err == nil {
		if pub, err := kp.PublicKey(); err == nil {
			accountPubKey = pub
		}
	}

	claims := NewAuthorizationResponseClaims(audience, subject, accountPubKey, userJWT, errMsg)

	token, err := claims.Encode(b.signingKey)
	if err != nil {
		return "", fmt.Errorf("failed to encode auth response: %w", err)
	}

	return token, nil
}
