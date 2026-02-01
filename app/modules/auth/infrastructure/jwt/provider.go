package authjwt

import (
	"errors"
	"fmt"
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// pwaClaims represents the JWT claims structure.
type pwaClaims struct {
	jwt.RegisteredClaims
	Guild string `json:"guild"`
	Role  string `json:"role"`
}

// provider implements the Provider interface.
type provider struct {
	secret []byte
}

// NewProvider creates a new JWT provider.
func NewProvider(secret string) Provider {
	return &provider{
		secret: []byte(secret),
	}
}

// GenerateToken creates a signed JWT token for the given user, guild, and role.
func (p *provider) GenerateToken(userID, guildID string, role authdomain.Role, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &pwaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Guild: guildID,
		Role:  string(role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(p.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the domain claims if valid.
func (p *provider) ValidateToken(tokenString string) (*authdomain.Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &pwaClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidSignature
		}
		return p.secret, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*pwaClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Convert to domain claims
	domainClaims := &authdomain.Claims{
		UserID:  claims.Subject,
		GuildID: claims.Guild,
		Role:    authdomain.Role(claims.Role),
	}

	if claims.ExpiresAt != nil {
		domainClaims.ExpiresAt = claims.ExpiresAt.Time
	}
	if claims.IssuedAt != nil {
		domainClaims.IssuedAt = claims.IssuedAt.Time
	}

	return domainClaims, nil
}
