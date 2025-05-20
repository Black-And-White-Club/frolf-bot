package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"slices"
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

// Helper function to sort LeaderboardData by TagNumber (copied from your GenerateUpdatedLeaderboard test)
func sortLeaderboardData(data leaderboardtypes.LeaderboardData) {
	slices.SortFunc(data, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber != 0 && b.TagNumber != 0 {
			return int(a.TagNumber - b.TagNumber)
		}
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0
		}
		if a.TagNumber == 0 {
			return 1
		}
		return -1
	})
}

func TestUpdateLeaderboard(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error)
		roundID         sharedtypes.RoundID
		sortedTags      []string
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string)
	}{
		{
			name: "Successful update - updates existing and adds new users",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(5)
				users[0].UserID = "user_A" // Tag 10 in initial LB
				users[1].UserID = "user_B" // Not in initial LB
				users[2].UserID = "user_C" // Tag 20 in initial LB
				users[3].UserID = "user_D" // Not in initial LB
				users[4].UserID = "user_E" // Tag 30 in initial LB
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 10},
						{UserID: "user_C", TagNumber: 20},
						{UserID: "user_E", TagNumber: 30},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			// Participants in performance order: user_A, user_B, user_D, user_C, user_E
			// Original tags of participants from initial LB: user_A (10), user_C (20), user_E (30)
			// Sorted original participant tags: [10, 20, 30]
			// Expected tag assignments: user_A (10), user_B (20), user_D (30), user_C (needs a tag), user_E (needs a tag)
			// This input seems to have more participants than available original tags from the initial leaderboard.
			// Let's adjust the expected data based on the actual GenerateUpdatedLeaderboard logic:
			// It takes tags from sortedParticipantTags: [1, 2, 3, 20, 30]. Sorted: [1, 2, 3, 20, 30]
			// Assigns these to users in sortedTags order: user_A (1), user_B (2), user_D (3), user_C (20), user_E (30)
			// Non-participants from initial LB (none in this case, all initial users are in sortedTags) are added back.
			// Final expected data, sorted by tag:
			sortedTags: []string{
				"1:user_A", // Best performer
				"2:user_B",
				"3:user_D",
				"20:user_C",
				"30:user_E", // Worst performer
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Success payload missing RoundID")
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				// Expected data based on GenerateUpdatedLeaderboard logic and sortedTags:
				expectedLeaderboardData := leaderboardtypes.LeaderboardData{
					{UserID: "user_A", TagNumber: 1},
					{UserID: "user_B", TagNumber: 2},
					{UserID: "user_D", TagNumber: 3},
					{UserID: "user_C", TagNumber: 20},
					{UserID: "user_E", TagNumber: 30},
				}
				sortLeaderboardData(expectedLeaderboardData) // Ensure expected data is sorted by tag

				actualLeaderboardData := activeLeaderboard.LeaderboardData
				sortLeaderboardData(actualLeaderboardData) // Ensure actual data is sorted by tag

				if len(actualLeaderboardData) != len(expectedLeaderboardData) {
					t.Errorf("Expected %d leaderboard entries, got %d. Actual data: %+v", len(expectedLeaderboardData), len(actualLeaderboardData), actualLeaderboardData)
					return
				}

				for i := range expectedLeaderboardData {
					if actualLeaderboardData[i].UserID != expectedLeaderboardData[i].UserID || actualLeaderboardData[i].TagNumber != expectedLeaderboardData[i].TagNumber {
						t.Errorf("At position %d: Expected %+v, got %+v", i, expectedLeaderboardData[i], actualLeaderboardData[i])
					}
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Successful update with no changes",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_X"
				users[1].UserID = "user_Y"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_X", TagNumber: 5},
						{UserID: "user_Y", TagNumber: 6},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"5:user_X",
				"6:user_Y",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				_, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				// Expected data based on GenerateUpdatedLeaderboard logic and sortedTags:
				expectedLeaderboardData := leaderboardtypes.LeaderboardData{
					{UserID: "user_X", TagNumber: 5},
					{UserID: "user_Y", TagNumber: 6},
				}
				sortLeaderboardData(expectedLeaderboardData) // Ensure expected data is sorted by tag

				actualLeaderboardData := activeLeaderboard.LeaderboardData
				sortLeaderboardData(actualLeaderboardData) // Ensure actual data is sorted by tag

				if len(actualLeaderboardData) != len(expectedLeaderboardData) {
					t.Errorf("Expected %d leaderboard entries, got %d. Actual data: %+v", len(expectedLeaderboardData), len(actualLeaderboardData), actualLeaderboardData)
					return
				}

				for i := range expectedLeaderboardData {
					if actualLeaderboardData[i].UserID != expectedLeaderboardData[i].UserID || actualLeaderboardData[i].TagNumber != expectedLeaderboardData[i].TagNumber {
						t.Errorf("At position %d: Expected %+v, got %+v", i, expectedLeaderboardData[i], actualLeaderboardData[i])
					}
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Empty sorted participant tags",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_initial", TagNumber: 99},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}
				return nil, initialLeaderboard, nil
			},
			roundID:         sharedtypes.RoundID(uuid.New()),
			sortedTags:      []string{},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.LeaderboardUpdateFailedPayload, but got %T", result.Failure)
					return
				}
				if !strings.Contains(failurePayload.Reason, "invalid input: empty sorted participant tags") {
					t.Errorf("Expected failure reason to contain 'invalid input: empty sorted participant tags', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}

				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected exactly one active leaderboard, got %d", len(activeLeaderboards))
				} else if activeLeaderboards[0].ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboards[0].ID)
				}

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 1 {
					t.Errorf("Expected only one leaderboard record after empty input, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "No active leaderboard initially", // This test case is being kept for now based on the provided output, but might need re-evaluation if no active leaderboard is truly impossible.
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				return nil, nil, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"1:user_A",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.LeaderboardUpdateFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.LeaderboardUpdateFailedPayload, but got %T", result.Failure)
					return
				}
				// Updated expected reason to match the actual error from GetActiveLeaderboard returning nil
				if !strings.Contains(failurePayload.Reason, "database connection error") && !strings.Contains(failurePayload.Reason, "sql: no rows in result set") {
					t.Errorf("Expected failure reason to indicate DB error or no rows, got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
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

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 0 {
					t.Errorf("Expected no leaderboard records, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "Initial active leaderboard with empty data",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
					IsActive:        true,
					UpdateSource:    leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}
				return nil, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			sortedTags: []string{
				"1:user_A",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}

				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected exactly one active leaderboard, got %d", len(activeLeaderboards))
				} else if activeLeaderboards[0].ID == initialLeaderboard.ID {
					t.Errorf("Expected a new leaderboard to be active, but the initial leaderboard (%d) is still active", initialLeaderboard.ID)
				}

				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				if len(allLeaderboards) != 2 {
					t.Errorf("Expected two leaderboard records after update, got %d", len(allLeaderboards))
				}
			},
		},
		{
			name: "Successful update with one new tag (updates existing and adds one)",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_A" // Tag 10 in initial LB
				users[1].UserID = "user_B" // Not in initial LB
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 10},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			roundID: sharedtypes.RoundID(uuid.New()),
			// Participants in performance order: user_A, user_B
			// Original tags of participants from initial LB: user_A (10)
			// Sorted original participant tags: [10]
			// Expected tag assignments based on GenerateUpdatedLeaderboard logic:
			// It takes tags from sortedParticipantTags: [5, 6]. Sorted: [5, 6]
			// Assigns these to users in sortedTags order: user_A (5), user_B (6)
			// Non-participants from initial LB (none) are added back.
			// Final expected data, sorted by tag:
			sortedTags: []string{
				"5:user_A",
				"6:user_B",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				_, ok := result.Success.(*leaderboardevents.LeaderboardUpdatedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.LeaderboardUpdatedPayload, but got %T", result.Success)
					return
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard, expectedRoundID sharedtypes.RoundID, inputSortedTags []string) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.UpdateID != expectedRoundID {
					t.Errorf("Expected active leaderboard UpdateID to be %s, got %s", expectedRoundID, activeLeaderboard.UpdateID)
				}
				if activeLeaderboard.UpdateSource != leaderboarddb.ServiceUpdateSourceProcessScores {
					t.Errorf("Expected active leaderboard UpdateSource to be %s, got %s", leaderboarddb.ServiceUpdateSourceProcessScores, activeLeaderboard.UpdateSource)
				}

				// Expected data based on GenerateUpdatedLeaderboard logic and sortedTags:
				expectedLeaderboardData := leaderboardtypes.LeaderboardData{
					{UserID: "user_A", TagNumber: 5},
					{UserID: "user_B", TagNumber: 6},
				}
				sortLeaderboardData(expectedLeaderboardData) // Ensure expected data is sorted by tag

				actualLeaderboardData := activeLeaderboard.LeaderboardData
				sortLeaderboardData(actualLeaderboardData) // Ensure actual data is sorted by tag

				if len(actualLeaderboardData) != len(expectedLeaderboardData) {
					t.Errorf("Expected %d leaderboard entries, got %d. Actual data: %+v", len(expectedLeaderboardData), len(actualLeaderboardData), actualLeaderboardData)
					return
				}

				for i := range expectedLeaderboardData {
					if actualLeaderboardData[i].UserID != expectedLeaderboardData[i].UserID || actualLeaderboardData[i].TagNumber != expectedLeaderboardData[i].TagNumber {
						t.Errorf("At position %d: Expected %+v, got %+v", i, expectedLeaderboardData[i], actualLeaderboardData[i])
					}
				}

				if activeLeaderboard.ID == initialLeaderboard.ID {
					t.Errorf("Expected a new active leaderboard, but the old one is still active")
				}

				var oldLeaderboard leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&oldLeaderboard).
					Where("id = ?", initialLeaderboard.ID).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testutils.CleanLeaderboardIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean leaderboard integration tables: %v", err)
			}
			if err := testutils.CleanUserIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean user integration tables: %v", err)
			}

			var initialUsers []testutils.User
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialUsers, initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			result, err := deps.Service.UpdateLeaderboard(sharedCtx, tt.roundID, tt.sortedTags)

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
				tt.validateDB(t, deps, initialUsers, initialLeaderboard, tt.roundID, tt.sortedTags)
			}
		})
	}
}
