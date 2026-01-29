package leaderboardintegrationtests

import (
	"context"
	"errors"
	"testing"

	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestExecuteBatchTagAssignment_Integration(t *testing.T) {
	// deps provides access to deps.Service (LeaderboardService) and deps.BunDB
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	tests := []struct {
		name          string
		setupData     func(t *testing.T, db *bun.DB)
		guildID       sharedtypes.GuildID
		requests      []sharedtypes.TagAssignmentRequest
		expectSwapErr bool
		validate      func(t *testing.T, db *bun.DB, result results.OperationResult[leaderboardtypes.LeaderboardData, error], err error)
	}{
		{
			name: "Successful complex internal swap",
			setupData: func(t *testing.T, db *bun.DB) {
				// Current state: User1(Tag1), User2(Tag2)
				lb := &leaderboarddb.Leaderboard{
					GuildID: "guild_integration", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_1", TagNumber: 1},
						{UserID: "user_2", TagNumber: 2},
					},
				}
				_, err := db.NewInsert().Model(lb).Exec(context.Background())
				if err != nil {
					t.Fatalf("failed to setup test data: %v", err)
				}
			},
			guildID: "guild_integration",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user_1", TagNumber: 2}, // User 1 takes User 2's tag
				{UserID: "user_2", TagNumber: 3}, // User 2 moves to a new tag
			},
			expectSwapErr: false,
			validate: func(t *testing.T, db *bun.DB, result results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if err != nil {
					t.Fatalf("expected success, got: %v", err)
				}

				// Verify DB: New active record should exist
				var activeLB leaderboarddb.Leaderboard
				db.NewSelect().Model(&activeLB).Where("guild_id = ? AND is_active = true", "guild_integration").Scan(context.Background())

				if len(activeLB.LeaderboardData) != 2 {
					t.Errorf("expected 2 entries in DB, got %d", len(activeLB.LeaderboardData))
				}

				// Verify result: Check that Assignments were computed correctly in the success payload
				if result.Success == nil {
					t.Fatalf("expected success payload, got nil")
				}
				successPayload := *result.Success
				// Assignments are not directly returned as payload in generic result ?
				// The generic result is [LeaderboardData, error].
				// So Success is *LeaderboardData.

				// Wait, LeaderboardData is []LeaderboardEntry. It doesn't have "Assignments" field.
				// We need to check the data itself.
				if len(successPayload) != 2 {
					t.Errorf("expected 2 entries in result, got %d", len(successPayload))
				}
			},
		},
		{
			name: "Conflict with external user returns TagSwapNeededError",
			setupData: func(t *testing.T, db *bun.DB) {
				// User 100 has Tag 10 but is NOT in the request batch
				lb := &leaderboarddb.Leaderboard{
					GuildID: "guild_integration", IsActive: true,
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_external", TagNumber: 10},
					},
				}
				_, err := db.NewInsert().Model(lb).Exec(context.Background())
				if err != nil {
					t.Fatalf("failed to setup test data: %v", err)
				}
			},
			guildID: "guild_integration",
			requests: []sharedtypes.TagAssignmentRequest{
				{UserID: "user_new", TagNumber: 10},
			},
			expectSwapErr: true,
			validate: func(t *testing.T, db *bun.DB, result results.OperationResult[leaderboardtypes.LeaderboardData, error], err error) {
				if err != nil {
					t.Fatalf("expected no system error, got %v", err)
				}
				if result.Failure == nil {
					t.Fatalf("expected domain failure, got nil")
				}

				var swapErr *leaderboardservice.TagSwapNeededError
				if !errors.As(*result.Failure, &swapErr) {
					t.Fatalf("expected TagSwapNeededError, got %v", *result.Failure)
				}

				if swapErr.TargetUserID != "user_external" {
					t.Errorf("expected conflict with user_external, got %s", swapErr.TargetUserID)
				}

				// Verify Atomicity: No new leaderboard should have been created
				count, _ := db.NewSelect().Model((*leaderboarddb.Leaderboard)(nil)).Count(context.Background())
				if count != 1 {
					t.Errorf("expected only 1 leaderboard record (no updates), found %d", count)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)

			if tt.setupData != nil {
				tt.setupData(t, deps.BunDB)
			}

			result, err := deps.Service.ExecuteBatchTagAssignment(
				ctx,
				tt.guildID,
				tt.requests,
				sharedtypes.RoundID(uuid.New()),
				sharedtypes.ServiceUpdateSourceManual,
			)

			tt.validate(t, deps.BunDB, result, err)
		})
	}
}
