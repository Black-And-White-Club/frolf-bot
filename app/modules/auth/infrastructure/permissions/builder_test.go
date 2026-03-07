package permissions

import (
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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
				for _, expectedSub := range []string{
					fmt.Sprintf("round.*.v1.%s", clubUUID),
					fmt.Sprintf("round.*.v2.%s", clubUUID),
					fmt.Sprintf("leaderboard.*.v1.%s", clubUUID),
					fmt.Sprintf("leaderboard.*.v2.%s", clubUUID),
				} {
					if !contains(p.Subscribe.Allow, expectedSub) {
						t.Errorf("expected subscription allow for %s, got %v", expectedSub, p.Subscribe.Allow)
					}
				}
				// Participant publish actions are unscoped
				for _, expectedPub := range []string{
					"round.participant.join.requested.v2",
					"round.participant.declined.v1",
					"round.participant.removal.requested.v2",
					"user.udisc.identity.update.requested.v1",
				} {
					if !contains(p.Publish.Allow, expectedPub) {
						t.Errorf("expected publish allow for %s, got %v", expectedPub, p.Publish.Allow)
					}
				}
				if contains(p.Publish.Allow, "round.score.update.requested.v2") {
					t.Errorf("player permissions must not allow score update writes, got %v", p.Publish.Allow)
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
				// Editor publish actions are unscoped
				for _, expectedPub := range []string{
					"round.creation.requested.v2",
					"round.update.requested.v2",
					"round.delete.requested.v2",
					"round.score.update.requested.v2",
					"user.udisc.identity.update.requested.v1",
				} {
					if !contains(p.Publish.Allow, expectedPub) {
						t.Errorf("expected publish allow for %s, got %v", expectedPub, p.Publish.Allow)
					}
				}
			},
		},
		{
			name: "admin permissions include unscoped admin publish subjects",
			claims: &authdomain.Claims{
				UserUUID:       userUUID,
				ActiveClubUUID: clubUUID,
				Role:           authdomain.RoleAdmin,
				Clubs: []authdomain.ClubRole{
					{ClubUUID: clubUUID, Role: authdomain.RoleAdmin},
				},
			},
			verify: func(t *testing.T, p *Permissions) {
				expectedPublishSubjects := []string{
					leaderboardevents.LeaderboardPointHistoryRequestedV1,
					leaderboardevents.LeaderboardManualPointAdjustmentV2,
					leaderboardevents.LeaderboardRecalculateRoundV1,
					leaderboardevents.LeaderboardStartNewSeasonV1,
					leaderboardevents.LeaderboardEndSeasonV1,
					leaderboardevents.LeaderboardGetSeasonStandingsV1,
					"leaderboard.batch.tag.assignment.requested.v2",
					"round.scorecard.admin.upload.requested.v2",
				}

				for _, expectedPub := range expectedPublishSubjects {
					if !contains(p.Publish.Allow, expectedPub) {
						t.Errorf("expected admin publish allow for %s, got %v", expectedPub, p.Publish.Allow)
					}
				}
			},
		},
		{
			name: "admin permissions are unscoped",
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
				// Admin actions are unscoped (guild scoping done via payload).
				// Regression guard: these write subjects must not regain a club/guild suffix.
				for _, subject := range []string{
					leaderboardevents.LeaderboardPointHistoryRequestedV1,
					leaderboardevents.LeaderboardManualPointAdjustmentV2,
					leaderboardevents.LeaderboardRecalculateRoundV1,
					leaderboardevents.LeaderboardStartNewSeasonV1,
					leaderboardevents.LeaderboardEndSeasonV1,
					leaderboardevents.LeaderboardGetSeasonStandingsV1,
				} {
					scoped := fmt.Sprintf("%s.%s", subject, clubUUID)
					if contains(p.Publish.Allow, scoped) {
						t.Errorf("scoped admin publish subject %s must not be present (write subjects are intentionally unscoped)", scoped)
					}
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
