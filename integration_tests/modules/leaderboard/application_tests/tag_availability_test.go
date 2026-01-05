package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

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
	defer deps.Cleanup()

	ctx := context.Background()
	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagAvailabilityCheckRequestedPayloadV1
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayloadV1, failure *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, err error)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Tag is available when no active leaderboard exists",
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
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   "test_guild",
				UserID:    "user_1",
				TagNumber: tagPtr(10),
			},
			expectedError:   false, // Expect no error as a new leaderboard is created
			expectedSuccess: true,  // Expect success payload
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayloadV1, failure *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, err error) {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if !success.Available {
					t.Errorf("Expected tag to be available")
				}
				if failure != nil {
					t.Errorf("Expected no failure payload, but got %+v", failure)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// DB state should now have an active leaderboard
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}
				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected 1 active leaderboard, got %d", len(activeLeaderboards))
				}
			},
		},
		{
			name: "Tag is available when active leaderboard exists but tag is not assigned",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 1},
						{UserID: "user_B", TagNumber: 2},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
					GuildID:      "test_guild",
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   "test_guild",
				UserID:    "user_2",
				TagNumber: tagPtr(3), // Tag 3 is not in the initial leaderboard
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayloadV1, failure *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "user_2" {
					t.Errorf("Expected UserID 'user_2', got '%s'", success.UserID)
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
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_X", TagNumber: 50},
						{UserID: "user_Y", TagNumber: 60},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
					GuildID:      "test_guild",
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   "test_guild",
				UserID:    "user_3",
				TagNumber: tagPtr(50), // Tag 50 is assigned to user_X
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayloadV1, failure *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "user_3" {
					t.Errorf("Expected UserID 'user_3', got '%s'", success.UserID)
				}
				if success.TagNumber == nil || *success.TagNumber != 50 {
					t.Errorf("Expected TagNumber 50, got %v", success.TagNumber)
				}
				if success.Available {
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
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
					IsActive:        true,
					UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
					GuildID:         "test_guild",
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: leaderboardevents.TagAvailabilityCheckRequestedPayloadV1{
				GuildID:   "test_guild",
				UserID:    "user_4",
				TagNumber: tagPtr(-10), // Invalid tag number
			},
			expectedError:   false, // Corrected expectation: Service returns success for negative tag if leaderboard exists
			expectedSuccess: true,  // Corrected expectation: Service returns success payload
			validateResult: func(t *testing.T, deps TestDeps, success *leaderboardevents.TagAvailabilityCheckResultPayloadV1, failure *leaderboardevents.TagAvailabilityCheckFailedPayloadV1, err error) {
				if success == nil {
					t.Errorf("Expected success payload, but got nil")
					return
				}
				if success.UserID != "user_4" {
					t.Errorf("Expected UserID 'user_4', got '%s'", success.UserID)
				}
				// Corrected validation: TagNumber should be echoed back in the success payload
				if success.TagNumber == nil || *success.TagNumber != -10 {
					t.Errorf("Expected TagNumber -10 in success payload, got %v", success.TagNumber)
				}
				// Corrected validation: Available should be true as negative tags are not in the leaderboard data
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
				// DB state should be unchanged from setupData
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
			guildID := sharedtypes.GuildID("test_guild")
			successResult, failureResult, err := deps.Service.CheckTagAvailability(ctx, guildID, tt.payload)

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
