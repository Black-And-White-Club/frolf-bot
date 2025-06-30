package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/uptrace/bun"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
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
		payload         leaderboardevents.TagSwapRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag swap between two users",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
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
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user_swap_A",
				TargetID:    "user_swap_B",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.TagSwapProcessedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.TagSwapProcessedPayload, but got %T", result.Success)
					return
				}

				if successPayload.RequestorID != "user_swap_A" {
					t.Errorf("Expected RequestorID 'user_swap_A', got '%s'", successPayload.RequestorID)
				}
				if successPayload.TargetID != "user_swap_B" {
					t.Errorf("Expected TargetID 'user_swap_B', got '%s'", successPayload.TargetID)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Order("id ASC").
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				if len(leaderboards) != 2 {
					t.Errorf("Expected 2 leaderboard records (old inactive, new active), got %d", len(leaderboards))
					return
				}

				oldLeaderboard := leaderboards[0]
				newLeaderboard := leaderboards[1]

				if oldLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected old leaderboard ID %d, got %d", initialLeaderboard.ID, oldLeaderboard.ID)
				}
				if oldLeaderboard.IsActive {
					t.Error("Expected old leaderboard to be inactive")
				}

				if !newLeaderboard.IsActive {
					t.Error("Expected new leaderboard to be active")
				}

				// Verify tags are swapped in the new leaderboard
				foundSwapA := false
				foundSwapB := false
				foundOtherUser := false
				for _, entry := range newLeaderboard.LeaderboardData {
					if entry.UserID == "user_swap_A" && entry.TagNumber != 0 && entry.TagNumber == 20 {
						foundSwapA = true
					}
					if entry.UserID == "user_swap_B" && entry.TagNumber != 0 && entry.TagNumber == 10 {
						foundSwapB = true
					}
					if entry.UserID == "other_user" && entry.TagNumber != 0 && entry.TagNumber == 30 {
						foundOtherUser = true
					}
				}
				if !foundSwapA {
					t.Error("User A did not get tag 20 in the new leaderboard")
				}
				if !foundSwapB {
					t.Error("User B did not get tag 10 in the new leaderboard")
				}
				if !foundOtherUser {
					t.Error("Other user's tag was not preserved")
				}
			},
		},
		{
			name: "Tag swap fails if requestor does not have a tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_B", TagNumber: 20},
						{UserID: "other_user", TagNumber: 30},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user_swap_A", // Does not exist or has no tag
				TargetID:    "user_swap_B",
			},
			expectedError:   false, // Service returns failure payload, not an error
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagSwapFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.RequestorID != "user_swap_A" {
					t.Errorf("Expected RequestorID 'user_swap_A', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "user_swap_B" {
					t.Errorf("Expected TargetID 'user_swap_B', got '%s'", failurePayload.TargetID)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "one or both users do not have tags on the leaderboard") {
					t.Errorf("Expected failure reason to indicate missing users/tags, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should remain unchanged
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
					return
				}

				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag swap fails if target does not have a tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_swap_A", TagNumber: 10},
						{UserID: "other_user", TagNumber: 30},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user_swap_A",
				TargetID:    "user_swap_B", // Does not exist or has no tag
			},
			expectedError:   false, // Service returns failure payload, not an error
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagSwapFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.RequestorID != "user_swap_A" {
					t.Errorf("Expected RequestorID 'user_swap_A', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "user_swap_B" {
					t.Errorf("Expected TargetID 'user_swap_B', got '%s'", failurePayload.TargetID)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "one or both users do not have tags on the leaderboard") {
					t.Errorf("Expected failure reason to indicate missing users/tags, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should remain unchanged
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
					return
				}

				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag swap fails if neither user has a tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "other_user", TagNumber: 30},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user_swap_A", // Does not exist or has no tag
				TargetID:    "user_swap_B", // Does not exist or has no tag
			},
			expectedError:   false, // Service returns failure payload, not an error
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagSwapFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.RequestorID != "user_swap_A" {
					t.Errorf("Expected RequestorID 'user_swap_A', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "user_swap_B" {
					t.Errorf("Expected TargetID 'user_swap_B', got '%s'", failurePayload.TargetID)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "one or both users do not have tags on the leaderboard") {
					t.Errorf("Expected failure reason to indicate missing users/tags, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should remain unchanged
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
					return
				}

				leaderboard := leaderboards[0]
				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag swap fails if no active leaderboard exists",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				// Ensure no active leaderboard
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}
				return nil, nil // No initial active leaderboard
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "user_swap_A",
				TargetID:    "user_swap_B",
			},
			expectedError:   false, // Service returns failure payload, not an error
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagSwapFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagSwapFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.RequestorID != "user_swap_A" {
					t.Errorf("Expected RequestorID 'user_swap_A', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "user_swap_B" {
					t.Errorf("Expected TargetID 'user_swap_B', got '%s'", failurePayload.TargetID)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "no active leaderboard found") {
					t.Errorf("Expected failure reason to indicate no active leaderboard, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should remain unchanged
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}
				if len(leaderboards) != 0 {
					t.Errorf("Expected 0 leaderboard records in DB, got %d", len(leaderboards))
				}
			},
		},
		// Add a test case for DB error during SwapTags if your mock DB can simulate it
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean database tables before each test case.
			cleanCtx := context.Background()
			err := testutils.CleanUserIntegrationTables(cleanCtx, deps.BunDB)
			if err != nil {
				t.Fatalf("Failed to clean user tables: %v", err)
			}
			err = testutils.CleanLeaderboardIntegrationTables(cleanCtx, deps.BunDB)
			if err != nil {
				t.Fatalf("Failed to clean leaderboard tables: %v", err)
			}

			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			ctx := context.Background()
			guildID := sharedtypes.GuildID("test_guild")
			result, err := deps.Service.TagSwapRequested(ctx, guildID, tt.payload)

			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if tt.expectedSuccess && result.Success == nil {
				t.Errorf("Expected a success result, but got nil")
			}
			if !tt.expectedSuccess && result.Success != nil {
				t.Errorf("Expected no success result, but got: %+v", result.Success)
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
