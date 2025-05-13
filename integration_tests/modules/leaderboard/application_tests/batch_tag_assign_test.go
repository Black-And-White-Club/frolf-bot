package leaderboardintegrationtests

import (
	"context"
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

func TestBatchTagAssignmentRequested(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.BatchTagAssignmentRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful batch assignment",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(3)
				users[0].UserID = "user_1"
				users[1].UserID = "user_2"
				users[2].UserID = "user_3"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData:     leaderboardtypes.LeaderboardData{},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "initial_setup",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          uuid.New().String(),
				RequestingUserID: "test_admin_user",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user_1", TagNumber: 1},
					{UserID: "user_2", TagNumber: 2},
					{UserID: "user_3", TagNumber: 3},
				},
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.BatchTagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.BatchTagAssignedPayload, but got %T", result.Success)
					return
				}
				if successPayload.AssignmentCount != 3 {
					t.Errorf("Expected 3 assignments in success payload, got %d", successPayload.AssignmentCount)
				}
				expectedAssignments := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_1": 1,
					"user_2": 2,
					"user_3": 3,
				}
				if len(successPayload.Assignments) != len(expectedAssignments) {
					t.Errorf("Expected %d assignments in success payload, got %d", len(expectedAssignments), len(successPayload.Assignments))
					return
				}
				for _, assignment := range successPayload.Assignments {
					expectedTag, ok := expectedAssignments[assignment.UserID]
					if !ok {
						t.Errorf("Unexpected user_id in success payload assignments: %s", assignment.UserID)
						continue
					}
					if assignment.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s, but got %d", expectedTag, assignment.UserID, assignment.TagNumber)
					}
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Query the active leaderboard record
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				// Validate the entries within the LeaderboardData JSONB field
				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 3 {
					t.Errorf("Expected 3 leaderboard entries in active leaderboard data, got %d", len(leaderboardEntries))
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_1": 1,
					"user_2": 2,
					"user_3": 3,
				}

				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == nil || *entry.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries. Expected %d, found %d", len(expectedDBState), len(foundEntries))
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
					t.Errorf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Batch assignment with some invalid tag numbers and non-existent users",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_a"
				users[1].UserID = "user_c"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_initial", TagNumber: tagPtr(99)},
					},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "initial_setup",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          uuid.New().String(),
				RequestingUserID: "test_admin_user",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user_a", TagNumber: 10},
					{UserID: "user_b", TagNumber: -5},
					{UserID: "user_c", TagNumber: 11},
					{UserID: "user_d", TagNumber: 12},
				},
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.BatchTagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.BatchTagAssignedPayload, but got %T", result.Success)
					return
				}
				expectedProcessedCount := 3
				if successPayload.AssignmentCount != expectedProcessedCount {
					t.Errorf("Expected %d assignments in success payload, got %d", expectedProcessedCount, successPayload.AssignmentCount)
				}
				expectedAssignments := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_a": 10,
					"user_c": 11,
					"user_d": 12,
				}
				if len(successPayload.Assignments) != len(expectedAssignments) {
					t.Errorf("Expected %d assignments in success payload, got %d", len(expectedAssignments), len(successPayload.Assignments))
					return
				}
				for _, assignment := range successPayload.Assignments {
					expectedTag, ok := expectedAssignments[assignment.UserID]
					if !ok {
						t.Errorf("Unexpected user_id in success payload assignments: %s", assignment.UserID)
						continue
					}
					if assignment.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s, but got %d", expectedTag, assignment.UserID, assignment.TagNumber)
					}
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				expectedDBEntries := 4
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d", expectedDBEntries, len(leaderboardEntries))
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_a":       10,
					"user_c":       11,
					"user_d":       12,
					"user_initial": 99,
				}

				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == nil || *entry.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries for users: %v", expectedDBState)
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
					t.Errorf("Failed to find old leaderboard: %v", err)
				}
				if oldLeaderboard.IsActive {
					t.Errorf("Expected old leaderboard to be inactive, but it is still active")
				}
			},
		},
		{
			name: "Empty assignments list",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				users := generator.GenerateUsers(1)
				users[0].UserID = "existing_user"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "existing_user", TagNumber: tagPtr(42)},
					},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "initial_setup",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          uuid.New().String(),
				RequestingUserID: "test_admin_user",
				Assignments:      []leaderboardevents.TagAssignmentInfo{},
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.BatchTagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.BatchTagAssignedPayload, but got %T", result.Success)
					return
				}
				if successPayload.AssignmentCount != 0 {
					t.Errorf("Expected 0 assignments in success payload for empty input, got %d", successPayload.AssignmentCount)
				}
				if len(successPayload.Assignments) != 0 {
					t.Errorf("Expected empty assignments list in success payload, got %d entries", len(successPayload.Assignments))
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Query the active leaderboard record
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				// For empty assignments, the service does NOT call the repository's BatchAssignTags.
				// Thus, the initial leaderboard should remain active, and no new one should be created.
				if activeLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboard.ID)
				}

				// Verify no new leaderboard record was created
				var allLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&allLeaderboards).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query all leaderboards: %v", err)
				}
				// Assuming only one leaderboard existed initially
				if len(allLeaderboards) != 1 {
					t.Errorf("Expected only one leaderboard record after empty assignments, got %d", len(allLeaderboards))
				}

				// Optionally, verify the content of the single active leaderboard is unchanged
				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 1 || *leaderboardEntries[0].TagNumber != 42 || leaderboardEntries[0].UserID != "existing_user" {
					t.Errorf("Expected 1 leaderboard entry for existing_user with tag 42, got %d entries or wrong data %v", len(leaderboardEntries), leaderboardEntries)
				}
			},
		},
		{
			name: "Assignment to user_error (Simulated Success)",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				user := generator.GenerateUsers(1)
				user[0].UserID = "user_error"
				_, err := db.NewInsert().Model(&user).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_initial", TagNumber: tagPtr(99)},
					},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "initial_setup",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return user, initialLeaderboard, nil
			},
			payload: leaderboardevents.BatchTagAssignmentRequestedPayload{
				BatchID:          uuid.New().String(),
				RequestingUserID: "test_admin_user",
				Assignments: []leaderboardevents.TagAssignmentInfo{
					{UserID: "user_error", TagNumber: 50},
				},
			},
			expectedError:   false, // Expect success as DB error is not triggered
			expectedSuccess: true,  // Expect success as DB error is not triggered
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.BatchTagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.BatchTagAssignedPayload, but got %T", result.Success)
					return
				}
				if successPayload.AssignmentCount != 1 {
					t.Errorf("Expected 1 assignment in success payload, got %d", successPayload.AssignmentCount)
				}
				expectedAssignments := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_error": 50,
				}
				if len(successPayload.Assignments) != len(expectedAssignments) {
					t.Errorf("Expected %d assignments in success payload, got %d", len(expectedAssignments), len(successPayload.Assignments))
					return
				}
				for _, assignment := range successPayload.Assignments {
					expectedTag, ok := expectedAssignments[assignment.UserID]
					if !ok {
						t.Errorf("Unexpected user_id in success payload assignments: %s", assignment.UserID)
						continue
					}
					if assignment.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s, but got %d", expectedTag, assignment.UserID, assignment.TagNumber)
					}
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				expectedDBEntries := 2 // user_error, user_initial
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d", expectedDBEntries, len(leaderboardEntries))
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_error":   50,
					"user_initial": 99,
				}

				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == nil || *entry.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries for users: %v", expectedDBState)
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

			result, err := deps.Service.BatchTagAssignmentRequested(sharedCtx, tt.payload)

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
				tt.validateDB(t, deps, initialUsers, initialLeaderboard)
			}
		})
	}
}
