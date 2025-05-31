package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

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

	tests := []struct {
		name            string
		setupData       func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagSwapRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag swap between two users",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Clean database state for this subtest
				if err := testutils.TruncateTables(context.Background(), db, "leaderboards"); err != nil {
					return nil, fmt.Errorf("failed to truncate tables: %w", err)
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "swap_success_user_a", TagNumber: 10},
						{UserID: "swap_success_user_b", TagNumber: 20},
						{UserID: "swap_success_other", TagNumber: 30},
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
				RequestorID: "swap_success_user_a",
				TargetID:    "swap_success_user_b",
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

				if successPayload.RequestorID != "swap_success_user_a" {
					t.Errorf("Expected RequestorID 'swap_success_user_a', got '%s'", successPayload.RequestorID)
				}
				if successPayload.TargetID != "swap_success_user_b" {
					t.Errorf("Expected TargetID 'swap_success_user_b', got '%s'", successPayload.TargetID)
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
					if entry.UserID == "swap_success_user_a" && entry.TagNumber != 0 && entry.TagNumber == 20 {
						foundSwapA = true
					}
					if entry.UserID == "swap_success_user_b" && entry.TagNumber != 0 && entry.TagNumber == 10 {
						foundSwapB = true
					}
					if entry.UserID == "swap_success_other" && entry.TagNumber != 0 && entry.TagNumber == 30 {
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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Clean database state for this subtest
				if err := testutils.TruncateTables(context.Background(), db, "leaderboards"); err != nil {
					return nil, fmt.Errorf("failed to truncate tables: %w", err)
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "swap_missing_requestor_target", TagNumber: 20},
						{UserID: "swap_missing_requestor_other", TagNumber: 30},
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
				RequestorID: "swap_missing_requestor_missing", // Does not exist or has no tag
				TargetID:    "swap_missing_requestor_target",
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

				if failurePayload.RequestorID != "swap_missing_requestor_missing" {
					t.Errorf("Expected RequestorID 'swap_missing_requestor_missing', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "swap_missing_requestor_target" {
					t.Errorf("Expected TargetID 'swap_missing_requestor_target', got '%s'", failurePayload.TargetID)
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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Clean database state for this subtest
				if err := testutils.TruncateTables(context.Background(), db, "leaderboards"); err != nil {
					return nil, fmt.Errorf("failed to truncate tables: %w", err)
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "swap_missing_target_requestor", TagNumber: 10},
						{UserID: "swap_missing_target_other", TagNumber: 30},
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
				RequestorID: "swap_missing_target_requestor",
				TargetID:    "swap_missing_target_missing", // Does not exist or has no tag
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

				if failurePayload.RequestorID != "swap_missing_target_requestor" {
					t.Errorf("Expected RequestorID 'swap_missing_target_requestor', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "swap_missing_target_missing" {
					t.Errorf("Expected TargetID 'swap_missing_target_missing', got '%s'", failurePayload.TargetID)
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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Clean database state for this subtest
				if err := testutils.TruncateTables(context.Background(), db, "leaderboards"); err != nil {
					return nil, fmt.Errorf("failed to truncate tables: %w", err)
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "swap_neither_exists_other", TagNumber: 30},
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
				RequestorID: "swap_neither_exists_requestor", // Does not exist or has no tag
				TargetID:    "swap_neither_exists_target",    // Does not exist or has no tag
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

				if failurePayload.RequestorID != "swap_neither_exists_requestor" {
					t.Errorf("Expected RequestorID 'swap_neither_exists_requestor', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "swap_neither_exists_target" {
					t.Errorf("Expected TargetID 'swap_neither_exists_target', got '%s'", failurePayload.TargetID)
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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Clean database state for this subtest
				if err := testutils.TruncateTables(context.Background(), db, "leaderboards"); err != nil {
					return nil, fmt.Errorf("failed to truncate tables: %w", err)
				}

				// Ensure no active leaderboard by simply not creating one
				// (truncate already cleared everything)
				return nil, nil // No initial active leaderboard
			},
			payload: leaderboardevents.TagSwapRequestedPayload{
				RequestorID: "swap_no_leaderboard_requestor",
				TargetID:    "swap_no_leaderboard_target",
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

				if failurePayload.RequestorID != "swap_no_leaderboard_requestor" {
					t.Errorf("Expected RequestorID 'swap_no_leaderboard_requestor', got '%s'", failurePayload.RequestorID)
				}
				if failurePayload.TargetID != "swap_no_leaderboard_target" {
					t.Errorf("Expected TargetID 'swap_no_leaderboard_target', got '%s'", failurePayload.TargetID)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "no active leaderboard found") {
					t.Errorf("Expected failure reason to indicate no active leaderboard, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should remain unchanged (empty)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(t, deps.BunDB)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			ctx := context.Background()
			result, err := deps.Service.TagSwapRequested(ctx, tt.payload)

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
