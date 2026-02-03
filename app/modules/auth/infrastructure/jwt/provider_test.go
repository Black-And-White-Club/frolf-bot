package authjwt

import (
	"errors"
	"testing"
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
)

func TestProvider_GenerateAndValidateToken(t *testing.T) {
	secret := "test-secret"
	p := NewProvider(secret)
	userID := "user-123"
	guildID := "guild-456"
	role := authdomain.RolePlayer
	ttl := 1 * time.Hour

	claims := &authdomain.Claims{
		UserID:  userID,
		GuildID: guildID,
		Role:    role,
	}

	t.Run("success", func(t *testing.T) {
		token, err := p.GenerateToken(claims, ttl)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		if token == "" {
			t.Fatal("generated token is empty")
		}

		validatedClaims, err := p.ValidateToken(token)
		if err != nil {
			t.Fatalf("failed to validate token: %v", err)
		}

		if validatedClaims.UserID != userID {
			t.Errorf("expected userID %s, got %s", userID, validatedClaims.UserID)
		}
		if validatedClaims.GuildID != guildID {
			t.Errorf("expected guildID %s, got %s", guildID, validatedClaims.GuildID)
		}
		if validatedClaims.Role != role {
			t.Errorf("expected role %s, got %s", role, validatedClaims.Role)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Generate token with negative TTL to expire it immediately
		token, err := p.GenerateToken(claims, -1*time.Hour)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		_, err = p.ValidateToken(token)
		if !errors.Is(err, ErrExpiredToken) {
			t.Errorf("expected ErrExpiredToken, got %v", err)
		}
	})

	t.Run("invalid signature", func(t *testing.T) {
		token, err := p.GenerateToken(claims, ttl)
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}

		// Use a different provider with a different secret to validate
		p2 := NewProvider("wrong-secret")
		_, err = p2.ValidateToken(token)
		if !errors.Is(err, ErrInvalidSignature) {
			t.Errorf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := p.ValidateToken("not.a.jwt.token")
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("expected ErrInvalidToken, got %v", err)
		}
	})
}
