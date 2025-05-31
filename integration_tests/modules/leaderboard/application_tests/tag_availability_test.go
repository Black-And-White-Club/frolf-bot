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
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestCheckTagAvailability(t *testing.T) {
	deps := SetupTestLeaderboardService(t)

	ctx := context.Background()

	tests := []struct {
		name            string
		setupData       func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagAvailabilityCheckRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayload, failure *leaderboardevents.TagAvailabilityCheckFailedPayload, err error)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Tag is available when no active leaderboard exists",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
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
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayload{
				UserID:    "tag_avail_no_board_user",
				TagNumber: tagPtr(10),
			},
			expectedError:   true, // Service returns error when no active leaderboard
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayload, failure *leaderboardevents.TagAvailabilityCheckFailedPayload, err error) {
				if failure == nil {
					t.Errorf("Expected failure payload, but got nil")
					return
				}
				if failure.UserID != "tag_avail_no_board_user" {
					t.Errorf("Expected UserID 'tag_avail_no_board_user', got '%s'", failure.UserID)
				}
				if failure.TagNumber == nil || *failure.TagNumber != 10 {
					t.Errorf("Expected TagNumber 10 in failure payload, got %v", failure.TagNumber)
				}
				if !strings.Contains(strings.ToLower(failure.Reason), "failed to check tag availability") {
					t.Errorf("Expected failure reason to contain 'failed to check tag availability', got '%s'", failure.Reason)
				}
				if success != nil {
					t.Errorf("Expected no success payload, but got %+v", success)
				}
				if err == nil {
					t.Error("Expected an error, but got none")
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should be unchanged from setupData
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}
				if len(activeLeaderboards) != 0 {
					t.Errorf("Expected no active leaderboards, got %d", len(activeLeaderboards))
				}
			},
		},
		{
			name: "Tag is available when active leaderboard exists but tag is not assigned",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Create users for leaderboard
				if err := testutils.InsertUser(t, db, "tag_avail_existing_user_a", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "tag_avail_existing_user_b", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "tag_avail_existing_user_a", TagNumber: 1},
						{UserID: "tag_avail_existing_user_b", TagNumber: 2},
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
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayload{
				UserID:    "tag_avail_checking_user",
				TagNumber: tagPtr(3), // Tag 3 is not in the initial leaderboard
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayload, failure *leaderboardevents.TagAvailabilityCheckFailedPayload, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "tag_avail_checking_user" {
					t.Errorf("Expected UserID 'tag_avail_checking_user', got '%s'", success.UserID)
				}
				if success.TagNumber == nil || *success.TagNumber != 3 {
					t.Errorf("Expected TagNumber 3, got %v", success.TagNumber)
				}
				if !success.Available {
					t.Error("Expected tag to be available, but it was not")
				}
				if failure != nil {
					t.Errorf("Expected no failure payload, but got %+v", failure)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should be unchanged from setupData
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboard: %v", err)
				}
				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
					return
				}
				if !leaderboards[0].IsActive {
					t.Error("Expected initial leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag is not available when active leaderboard exists and tag is assigned",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// First, deactivate any existing active leaderboards to ensure isolation
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users for leaderboard with unique IDs for this test
				if err := testutils.InsertUser(t, db, "tag_unavail_user_x", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "tag_unavail_user_y", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "tag_unavail_user_x", TagNumber: 50},
						{UserID: "tag_unavail_user_y", TagNumber: 60},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}

				// Debug: Verify the leaderboard was created with the expected data
				var verifyLeaderboard leaderboarddb.Leaderboard
				err = db.NewSelect().
					Model(&verifyLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Logf("Warning: Could not verify leaderboard creation: %v", err)
				} else {
					t.Logf("Debug: Created leaderboard ID %d with %d entries, IsActive=%v",
						verifyLeaderboard.ID, len(verifyLeaderboard.LeaderboardData), verifyLeaderboard.IsActive)
					for i, entry := range verifyLeaderboard.LeaderboardData {
						t.Logf("Debug: Entry %d: UserID=%s, TagNumber=%d", i, entry.UserID, entry.TagNumber)
					}
				}

				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayload{
				UserID:    "tag_unavail_checking_user",
				TagNumber: tagPtr(50), // Tag 50 is assigned to tag_unavail_user_x
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayload, failure *leaderboardevents.TagAvailabilityCheckFailedPayload, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "tag_unavail_checking_user" {
					t.Errorf("Expected UserID 'tag_unavail_checking_user', got '%s'", success.UserID)
				}
				if success.TagNumber == nil || *success.TagNumber != 50 {
					t.Errorf("Expected TagNumber 50, got %v", success.TagNumber)
				}
				if success.Available {
					// Debug: Log what the service is actually finding
					var activeLeaderboard leaderboarddb.Leaderboard
					err := deps.BunDB.NewSelect().
						Model(&activeLeaderboard).
						Where("is_active = ?", true).
						Scan(context.Background())
					if err != nil {
						t.Logf("Debug: Error querying active leaderboard: %v", err)
					} else {
						t.Logf("Debug: Active leaderboard ID %d has %d entries",
							activeLeaderboard.ID, len(activeLeaderboard.LeaderboardData))
						for i, entry := range activeLeaderboard.LeaderboardData {
							t.Logf("Debug: Active Entry %d: UserID=%s, TagNumber=%d", i, entry.UserID, entry.TagNumber)
						}
					}
					t.Error("Expected tag to be unavailable, but it was available")
				}
				if failure != nil {
					t.Errorf("Expected no failure payload, but got %+v", failure)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should be unchanged from setupData
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboard: %v", err)
				}
				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
					return
				}
				if !leaderboards[0].IsActive {
					t.Error("Expected initial leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag availability check succeeds for invalid tag number (e.g., negative)",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
					IsActive:        true,
					UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayload{
				UserID:    "tag_negative_checking_user",
				TagNumber: tagPtr(-10), // Invalid tag number
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayload, failure *leaderboardevents.TagAvailabilityCheckFailedPayload, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "tag_negative_checking_user" {
					t.Errorf("Expected UserID 'tag_negative_checking_user', got '%s'", success.UserID)
				}
				if success.TagNumber == nil || *success.TagNumber != -10 {
					t.Errorf("Expected TagNumber -10 in success payload, got %v", success.TagNumber)
				}
				if !success.Available {
					t.Error("Expected tag to be available (not found in data), but it was not")
				}
				if failure != nil {
					t.Errorf("Expected no failure payload, but got %+v", failure)
				}
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should be unchanged from setupData - query only the specific leaderboard
				var leaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboard: %v", err)
				}

				if leaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
				}
				if !leaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// No manual truncation - let the test harness handle isolation
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(t, deps.BunDB)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			successResult, failureResult, err := deps.Service.CheckTagAvailability(ctx, tt.payload)

			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			if tt.expectedSuccess && successResult == nil {
				t.Errorf("Expected a success result, but got nil")
			}
			if !tt.expectedSuccess && successResult != nil {
				t.Errorf("Expected no success result, but got: %+v", successResult)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps, successResult, failureResult, err)
			}

			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}
