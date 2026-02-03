package authnats

import (
	"testing"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/Black-And-White-Club/frolf-bot/app/modules/auth/infrastructure/permissions"
	"github.com/google/uuid"
	"github.com/nats-io/nkeys"
)

func TestUserJWTBuilder_BuildUserJWT(t *testing.T) {
	// Setup keys
	accountKP, _ := nkeys.CreateAccount()
	accountPubKey, _ := accountKP.PublicKey()

	signingKP, _ := nkeys.CreateUser()

	builder := NewUserJWTBuilder(signingKP, accountPubKey)

	userUUID := uuid.New()
	clubUUID := uuid.New()

	tests := []struct {
		name   string
		claims *authdomain.Claims
		perms  *permissions.Permissions
		verify func(t *testing.T, token string, err error)
	}{
		{
			name: "success",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
			},
			perms: &permissions.Permissions{
				Publish:   permissions.PermissionSet{Allow: []string{"pub1"}},
				Subscribe: permissions.PermissionSet{Allow: []string{"sub1"}},
			},
			verify: func(t *testing.T, token string, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if token == "" {
					t.Fatal("expected token to be non-empty")
				}
				// We could decode the JWT here to verify contents if we wanted to be more thorough,
				// but just getting a signed JWT from uc.Encode(b.signingKey) is a good start.
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := builder.BuildUserJWT(tt.claims, tt.perms)
			tt.verify(t, token, err)
		})
	}
}
