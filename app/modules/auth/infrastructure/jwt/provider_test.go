package authjwt

import (
	"errors"
	"os"
	"testing"
	"time"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/google/uuid"
)

func TestProvider_GenerateAndValidateToken(t *testing.T) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "test-secret-at-least-32-chars-long!!"
	}
	p := NewProvider(secret)

	userUUID := uuid.New()
	clubUUID := uuid.New()

	claims := &authdomain.Claims{
		UserID:         "user-123",
		UserUUID:       userUUID,
		ActiveClubUUID: clubUUID,
		GuildID:        "guild-456",
		Role:           authdomain.RolePlayer,
	}

	tests := []struct {
		name        string
		setupClaims *authdomain.Claims
		ttl         time.Duration
		provider    Provider
		expectedErr error
		verify      func(t *testing.T, validated *authdomain.Claims)
	}{
		{
			name:        "success",
			setupClaims: claims,
			ttl:         1 * time.Hour,
			provider:    p,
			verify: func(t *testing.T, validated *authdomain.Claims) {
				if validated.UserID != claims.UserID {
					t.Errorf("expected userID %s, got %s", claims.UserID, validated.UserID)
				}
				if validated.UserUUID != claims.UserUUID {
					t.Errorf("expected userUUID %v, got %v", claims.UserUUID, validated.UserUUID)
				}
				if validated.ActiveClubUUID != claims.ActiveClubUUID {
					t.Errorf("expected ActiveClubUUID %v, got %v", claims.ActiveClubUUID, validated.ActiveClubUUID)
				}
			},
		},
		{
			name:        "expired token",
			setupClaims: claims,
			ttl:         -1 * time.Hour,
			provider:    p,
			expectedErr: ErrExpiredToken,
		},
		{
			name:        "invalid signature",
			setupClaims: claims,
			ttl:         1 * time.Hour,
			provider:    NewProvider("wrong-secret"),
			expectedErr: ErrInvalidSignature,
		},
		{
			name:        "malformed token",
			setupClaims: nil, // Special case for manual token
			expectedErr: ErrInvalidToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var token string
			var err error

			if tt.setupClaims != nil {
				token, err = p.GenerateToken(tt.setupClaims, tt.ttl)
				if err != nil {
					t.Fatalf("failed to generate token: %v", err)
				}
			} else if tt.name == "malformed token" {
				token = "not.a.jwt"
			}

			validateTarget := p
			if tt.provider != nil {
				validateTarget = tt.provider
			}

			validatedClaims, err := validateTarget.ValidateToken(token)

			if tt.expectedErr != nil {
				if !errors.Is(err, tt.expectedErr) {
					t.Errorf("expected error %v, got %v", tt.expectedErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.verify != nil {
				tt.verify(t, validatedClaims)
			}
		})
	}
}
