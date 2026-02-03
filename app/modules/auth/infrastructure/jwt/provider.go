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
	UserUUID       string                `json:"user_uuid,omitempty"`
	ActiveClubUUID string                `json:"active_club_uuid,omitempty"`
	Clubs          []authdomain.ClubRole `json:"clubs,omitempty"`
	Guild          string                `json:"guild,omitempty"` // Legacy Discord Guild ID
	Role           string                `json:"role,omitempty"`  // Legacy Role
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

// GenerateToken creates a signed JWT token from the given claims.
func (p *provider) GenerateToken(domainClaims *authdomain.Claims, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := &pwaClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   domainClaims.UserID,
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		UserUUID:       domainClaims.UserUUID.String(),
		ActiveClubUUID: domainClaims.ActiveClubUUID.String(),
		Clubs:          domainClaims.Clubs,
		Guild:          domainClaims.GuildID,
		Role:           string(domainClaims.Role),
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
		Clubs:   claims.Clubs,
	}

	if claims.UserUUID != "" {
		domainClaims.UserUUID, _ = uuid.Parse(claims.UserUUID)
	}
	if claims.ActiveClubUUID != "" {
		domainClaims.ActiveClubUUID, _ = uuid.Parse(claims.ActiveClubUUID)
	}

	if claims.ExpiresAt != nil {
		domainClaims.ExpiresAt = claims.ExpiresAt.Time
	}
	if claims.IssuedAt != nil {
		domainClaims.IssuedAt = claims.IssuedAt.Time
	}

	return domainClaims, nil
}
