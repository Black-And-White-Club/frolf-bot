package leaderboardintegrationtests

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardService "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestTagSwapRequested(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		userID          sharedtypes.DiscordID
		targetTag       sharedtypes.TagNumber
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag swap between two users",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					GuildID: "test_guild",
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_A", TagNumber: 10},
						{UserID: "user_swap_B", TagNumber: 20},
						{UserID: "other_user", TagNumber: 30},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				return initialLeaderboard, err
			},
			userID:          "user_swap_A",
			targetTag:       20, // Targets user_swap_B
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Leaderboard == nil {
					t.Errorf("Expected leaderboard data in result, got nil")
					return
				}
				// Verify changes slice contains the swap
				if len(result.TagChanges) != 2 {
					t.Errorf("Expected 2 tag changes, got %d", len(result.TagChanges))
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().Model(&leaderboards).Order("id ASC").Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				// The refactor uses s.LeaderboardDB.UpdateLeaderboard,
				// which typically creates a new record and deactivates the old one.
				if len(leaderboards) != 2 {
					t.Errorf("Expected 2 leaderboard records, got %d", len(leaderboards))
					return
				}

				newLB := leaderboards[1]
				foundA, foundB := false, false
				for _, entry := range newLB.LeaderboardData {
					if entry.UserID == "user_swap_A" && entry.TagNumber == 20 {
						foundA = true
					}
					if entry.UserID == "user_swap_B" && entry.TagNumber == 10 {
						foundB = true
					}
				}
				if !foundA || !foundB {
					t.Error("Tags were not correctly swapped in the database")
				}
			},
		},
		{
			name: "Fails if requestor is not on leaderboard",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					GuildID: "test_guild",
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_B", TagNumber: 20},
					},
					IsActive: true,
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				return initialLeaderboard, err
			},
			userID:        "stranger_danger",
			targetTag:     20,
			expectedError: false, // Business logic error returned in Result.Err
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Err == nil || !strings.Contains(result.Err.Error(), "requesting user not on leaderboard") {
					t.Errorf("Expected 'requesting user not on leaderboard' error, got: %v", result.Err)
				}
			},
		},
		{
			name: "Fails if target tag is unassigned",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					GuildID: "test_guild",
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_A", TagNumber: 10},
					},
					IsActive: true,
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				return initialLeaderboard, err
			},
			userID:    "user_swap_A",
			targetTag: 999, // Non-existent tag
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Err == nil || !strings.Contains(result.Err.Error(), "target tag not currently assigned") {
					t.Errorf("Expected 'target tag not currently assigned' error, got: %v", result.Err)
				}
			},
		},
		{
			name: "Fails if swapping with self",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					GuildID: "test_guild",
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_A", TagNumber: 10},
					},
					IsActive: true,
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				return initialLeaderboard, err
			},
			userID:    "user_swap_A",
			targetTag: 10,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Err == nil || !strings.Contains(result.Err.Error(), "cannot swap tag with self") {
					t.Errorf("Expected 'cannot swap tag with self' error, got: %v", result.Err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanCtx := context.Background()
			testutils.CleanLeaderboardIntegrationTables(cleanCtx, deps.BunDB)

			var initialLeaderboard *leaderboarddb.Leaderboard
			if tt.setupData != nil {
				var err error
				initialLeaderboard, err = tt.setupData(deps.BunDB, dataGen)
				if err != nil {
					t.Fatalf("Setup failed: %v", err)
				}
			}

			ctx := context.Background()
			guildID := sharedtypes.GuildID("test_guild")

			// Calling the refactored method signature
			result, err := deps.Service.TagSwapRequested(ctx, guildID, tt.userID, tt.targetTag)

			if tt.expectedError && err == nil {
				t.Errorf("Expected actual error (err), got nil")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Unexpected actual error (err): %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps, result)
			}
			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}
