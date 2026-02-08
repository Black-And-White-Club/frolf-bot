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
				Clubs: []authdomain.ClubRole{
					{ClubUUID: clubUUID, Role: authdomain.RolePlayer},
				},
			},
			verify: func(t *testing.T, p *Permissions) {
				// Subscribe patterns now use versioned format: round.*.v1.{id}
				expectedSub := fmt.Sprintf("round.*.v1.%s", clubUUID)
				if !contains(p.Subscribe.Allow, expectedSub) {
					t.Errorf("expected subscription allow for %s, got %v", expectedSub, p.Subscribe.Allow)
				}
				// Publish patterns use versioned format: round.participant.join.v1.{id}
				expectedPub := fmt.Sprintf("round.participant.join.v1.%s", clubUUID)
				if !contains(p.Publish.Allow, expectedPub) {
					t.Errorf("expected publish allow for %s, got %v", expectedPub, p.Publish.Allow)
				}
			},
		},
		{
			name: "editor permissions",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				Role:           authdomain.RoleEditor,
				Clubs: []authdomain.ClubRole{
					{ClubUUID: clubUUID, Role: authdomain.RoleEditor},
				},
			},
			verify: func(t *testing.T, p *Permissions) {
				// Editor publish pattern: round.create.v1.{id}
				expectedPub := fmt.Sprintf("round.create.v1.%s", clubUUID)
				if !contains(p.Publish.Allow, expectedPub) {
					t.Errorf("expected publish allow for %s, got %v", expectedPub, p.Publish.Allow)
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
