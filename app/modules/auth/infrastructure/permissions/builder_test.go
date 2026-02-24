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
		{
			name: "admin permissions include scoped scorecard upload",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				Role:           authdomain.RoleAdmin,
				Clubs: []authdomain.ClubRole{
					{ClubUUID: clubUUID, Role: authdomain.RoleAdmin},
				},
			},
			verify: func(t *testing.T, p *Permissions) {
				expectedPub := fmt.Sprintf("round.scorecard.admin.upload.requested.v1.%s", clubUUID)
				if !contains(p.Publish.Allow, expectedPub) {
					t.Errorf("expected admin publish allow for %s, got %v", expectedPub, p.Publish.Allow)
				}
			},
		},
		{
			name: "admin permissions scoped correctly",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				GuildID:        "123456789",
				Role:           authdomain.RoleAdmin,
				Clubs: []authdomain.ClubRole{
					{ClubUUID: clubUUID, Role: authdomain.RoleAdmin},
				},
			},
			verify: func(t *testing.T, p *Permissions) {
				// Verify scoped admin actions for both club UUID and Guild ID
				for _, id := range []string{clubUUID.String(), "123456789"} {
					expected := fmt.Sprintf("leaderboard.manual.point.adjustment.v1.%s", id)
					if !contains(p.Publish.Allow, expected) {
						t.Errorf("expected scoped admin publish allow for %s, got %v", expected, p.Publish.Allow)
					}
				}

				// Ensure unscoped subjects are NOT present (regression check)
				unscoped := "leaderboard.manual.point.adjustment.v1"
				if contains(p.Publish.Allow, unscoped) {
					t.Errorf("unscoped subject %s should NOT be allowed", unscoped)
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
