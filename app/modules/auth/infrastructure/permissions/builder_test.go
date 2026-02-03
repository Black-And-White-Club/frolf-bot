package permissions

import (
	"fmt"
	"testing"

	authdomain "github.com/Black-And-White-Club/frolf-bot/app/modules/auth/domain"
	"github.com/google/uuid"
)

func TestBuilder_ForRole(t *testing.T) {
	builder := NewBuilder()
	userUUID := uuid.New()
	clubUUID := uuid.New()

	tests := []struct {
		name   string
		claims *authdomain.Claims
		verify func(t *testing.T, p *Permissions)
	}{
		{
			name: "player permissions",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				Role:           authdomain.RolePlayer,
			},
			verify: func(t *testing.T, p *Permissions) {
				expectedSub := fmt.Sprintf("round.*.%s", clubUUID)
				if !contains(p.Subscribe.Allow, expectedSub) {
					t.Errorf("expected subscription allow for %s", expectedSub)
				}
				expectedPub := fmt.Sprintf("round.participant.join.%s", clubUUID)
				if !contains(p.Publish.Allow, expectedPub) {
					t.Errorf("expected publish allow for %s", expectedPub)
				}
			},
		},
		{
			name: "editor permissions",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				Role:           authdomain.RoleEditor,
			},
			verify: func(t *testing.T, p *Permissions) {
				expectedPub := fmt.Sprintf("round.create.%s", clubUUID)
				if !contains(p.Publish.Allow, expectedPub) {
					t.Errorf("expected publish allow for %s", expectedPub)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := builder.ForRole(tt.claims)
			tt.verify(t, p)
		})
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
