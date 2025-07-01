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

// TestProcessTagAssignments is an integration test for the ProcessTagAssignments service method.
// It tests the service's logic and its interaction with the database.
func TestProcessTagAssignments(t *testing.T) {
	// Setup the test environment dependencies. sharedCtx and sharedDB are initialized in TestMain.
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	tests := []struct {
		name string
		// setupData prepares the database for the test case.
		// It returns any generated users and the initial active leaderboard record if one is created.
		setupData func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error)
		// serviceParams contains the parameters to pass to ProcessTagAssignments
		serviceParams struct {
			guildID          sharedtypes.GuildID
			source           sharedtypes.ServiceUpdateSource
			requests         []sharedtypes.TagAssignmentRequest
			requestingUserID *sharedtypes.DiscordID
			operationID      uuid.UUID
			batchID          uuid.UUID
		}
		// expectedError indicates if the service call is expected to return a Go error.
		expectedError bool
		// expectedSuccess indicates if the service call is expected to return a success payload.
		expectedSuccess bool
		// validateResult asserts the content of the LeaderboardOperationResult returned by the service.
		validateResult func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		// validateDB asserts the state of the database after the service call.
		validateDB func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful batch assignment",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				// Generate and insert initial users.
				users := generator.GenerateUsers(3)
				users[0].UserID = "user_1"
				users[1].UserID = "user_2"
				users[2].UserID = "user_3"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				// Insert an initial active leaderboard record with correct GuildID.
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{}, // Start with empty data
					IsActive:        true,
					UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
					GuildID:         "test_guild",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			serviceParams: struct {
				guildID          sharedtypes.GuildID
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				guildID: "test_guild",
				source:  sharedtypes.ServiceUpdateSourceAdminBatch,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "user_1", TagNumber: 1},
					{UserID: "user_2", TagNumber: 2},
					{UserID: "user_3", TagNumber: 3},
				},
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
			},
			expectedError:   false, // Expect no Go error
			expectedSuccess: true,  // Expect a success payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				// Assert the type of the success payload
				successPayload, ok := result.Success.(*leaderboardevents.BatchTagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.BatchTagAssignedPayload, but got %T", result.Success)
					return
				}
				// Validate fields in the success payload
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
				// Validate each assignment in the success payload
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
				// Expecting 3 entries for the 3 assigned users
				if len(leaderboardEntries) != 3 {
					t.Errorf("Expected 3 leaderboard entries in active leaderboard data, got %d", len(leaderboardEntries))
					return
				}

				// Define the expected state of the leaderboard data in the DB
				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_1": 1,
					"user_2": 2,
					"user_3": 3,
				}

				// Verify each entry in the DB matches the expected state
				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == 0 || entry.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				// Verify all expected entries were found
				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries. Expected %d, found %d", len(expectedDBState), len(foundEntries))
				}

				// Verify a new active leaderboard was created and the old one is inactive
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
				// Generate and insert users that exist before the batch assignment.
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_a"
				users[1].UserID = "user_c"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				// Insert an initial active leaderboard with some existing data and correct GuildID.
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_initial", TagNumber: 99},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
					GuildID:      "test_guild",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			serviceParams: struct {
				guildID          sharedtypes.GuildID
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				guildID: "test_guild",
				source:  sharedtypes.ServiceUpdateSourceAdminBatch,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "user_a", TagNumber: 10}, // Existing user, valid tag
					{UserID: "user_b", TagNumber: -5}, // Non-existent user, invalid tag (should be skipped)
					{UserID: "user_c", TagNumber: 11}, // Existing user, valid tag
					{UserID: "user_d", TagNumber: 12}, // Non-existent user, valid tag (should be added)
				},
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
			},
			expectedError:   false, // Expect no Go error
			expectedSuccess: true,  // Expect a success payload
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

				// Debug: Print the actual assignments to understand what's being returned
				t.Logf("Actual assignment count: %d", successPayload.AssignmentCount)
				for i, assignment := range successPayload.Assignments {
					t.Logf("Assignment %d: UserID=%s, TagNumber=%d", i, assignment.UserID, assignment.TagNumber)
				}

				// The service returns the complete leaderboard state after the batch operation.
				// This includes: 3 new valid assignments + 1 existing user = 4 total entries.
				// The invalid assignment (TagNumber: -5) is filtered out during validation.
				expectedTotalCount := 4 // Complete leaderboard after updates
				if successPayload.AssignmentCount != expectedTotalCount {
					t.Errorf("Expected %d assignments in success payload, got %d", expectedTotalCount, successPayload.AssignmentCount)
				}
				// The assignments list should contain the complete leaderboard state after updates.
				expectedAssignments := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_a":       10, // New assignment
					"user_c":       11, // New assignment
					"user_d":       12, // New assignment
					"user_initial": 99, // Existing user carried over
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
				// Expected entries in DB: user_a (10), user_c (11), user_d (12), user_initial (99)
				// Invalid tag for user_b (-5) should be skipped by the service before passing to DB.
				expectedDBEntries := 4
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d", expectedDBEntries, len(leaderboardEntries))
					return
				}

				// Define the expected state of the leaderboard data in the DB
				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user_a":       10,
					"user_c":       11,
					"user_d":       12,
					"user_initial": 99, // Initial user should still be there
				}

				// Verify each entry in the DB matches the expected state
				foundEntries := make(map[sharedtypes.DiscordID]bool)
				for _, entry := range leaderboardEntries {
					expectedTag, ok := expectedDBState[entry.UserID]
					if !ok {
						t.Errorf("Unexpected user_id found in database: %s", entry.UserID)
						continue
					}
					if entry.TagNumber == 0 || entry.TagNumber != expectedTag {
						t.Errorf("Expected tag %d for user %s in DB, got %v", expectedTag, entry.UserID, entry.TagNumber)
					}
					foundEntries[entry.UserID] = true
				}

				// Verify all expected entries were found
				if len(foundEntries) != len(expectedDBState) {
					t.Errorf("Missing expected database entries for users: %v", expectedDBState)
				}

				// Verify a new active leaderboard was created and the old one is inactive
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
				// Insert a user and an initial active leaderboard with existing data.
				users := generator.GenerateUsers(1)
				users[0].UserID = "existing_user"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "existing_user", TagNumber: 42},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
					GuildID:      "test_guild",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			serviceParams: struct {
				guildID          sharedtypes.GuildID
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				guildID:          "test_guild",
				source:           sharedtypes.ServiceUpdateSourceAdminBatch,
				requests:         []sharedtypes.TagAssignmentRequest{}, // Empty list
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
			},
			expectedError:   false, // Expect no Go error
			expectedSuccess: true,  // Expect a success payload
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
				// For empty assignments, the service should return the current leaderboard state (1 existing user)
				if successPayload.AssignmentCount != 1 {
					t.Errorf("Expected 1 assignment in success payload for existing leaderboard state, got %d", successPayload.AssignmentCount)
				}
				if len(successPayload.Assignments) != 1 {
					t.Errorf("Expected 1 assignment in success payload, got %d entries", len(successPayload.Assignments))
				}
				// Verify the existing user is included
				if len(successPayload.Assignments) > 0 {
					assignment := successPayload.Assignments[0]
					if assignment.UserID != "existing_user" || assignment.TagNumber != 42 {
						t.Errorf("Expected existing_user with tag 42, got %s with tag %d", assignment.UserID, assignment.TagNumber)
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

				// For empty assignments, the service does NOT call the repository's UpdateLeaderboard.
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
				if len(leaderboardEntries) != 1 || leaderboardEntries[0].TagNumber != 42 || leaderboardEntries[0].UserID != "existing_user" {
					t.Errorf("Expected 1 leaderboard entry for existing_user with tag 42, got %d entries or wrong data %v", len(leaderboardEntries), leaderboardEntries)
				}
			},
		},
		{
			name: "Single assignment that requires tag swap",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) ([]testutils.User, *leaderboarddb.Leaderboard, error) {
				// Generate users where one already has a tag
				users := generator.GenerateUsers(2)
				users[0].UserID = "user_with_tag"
				users[1].UserID = "user_requesting_tag"
				_, err := db.NewInsert().Model(&users).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				// Insert an initial active leaderboard with user_with_tag having tag 1 and correct GuildID.
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_with_tag", TagNumber: 1},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
					GuildID:      "test_guild",
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, nil, err
				}

				return users, initialLeaderboard, nil
			},
			serviceParams: struct {
				guildID          sharedtypes.GuildID
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				guildID: "test_guild",
				source:  sharedtypes.ServiceUpdateSourceManual,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "user_requesting_tag", TagNumber: 1}, // This should trigger a failure result
				},
				requestingUserID: nil, // Individual assignment
				operationID:      uuid.New(),
				batchID:          uuid.New(),
			},
			expectedError:   false, // No Go error - service returns failure result instead
			expectedSuccess: false, // Don't expect a success payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				// Expect a failure result for tag conflict
				if result.Success != nil {
					t.Errorf("Expected no success result for tag conflict, but got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Errorf("Expected failure result for tag conflict, but got nil")
					return
				}
				// Validate it's the correct failure type
				failurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.BatchTagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}
				// Validate failure payload contents
				if failurePayload.RequestingUserID != "system" {
					t.Errorf("Expected requesting user ID 'system', got %s", failurePayload.RequestingUserID)
				}
				if failurePayload.Reason == "" {
					t.Errorf("Expected non-empty failure reason")
				}
				// Check that the reason mentions the tag conflict
				expectedReasonSubstring := "tag 1 is already assigned to user user_with_tag"
				if failurePayload.Reason != expectedReasonSubstring {
					t.Errorf("Expected failure reason to be '%s', got '%s'", expectedReasonSubstring, failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialUsers []testutils.User, initialLeaderboard *leaderboarddb.Leaderboard) {
				// For a tag conflict failure, no leaderboard changes should occur
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				// The initial leaderboard should still be active and unchanged
				if activeLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard to still be active for tag conflict, but a different one is active")
				}

				// Verify leaderboard data is unchanged
				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 1 || leaderboardEntries[0].UserID != "user_with_tag" || leaderboardEntries[0].TagNumber != 1 {
					t.Errorf("Expected unchanged leaderboard data for tag conflict, got %v", leaderboardEntries)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean database tables before each test case.
			ctx := context.Background()
			err := testutils.CleanUserIntegrationTables(ctx, deps.BunDB)
			if err != nil {
				t.Fatalf("Failed to clean user tables: %v", err)
			}
			err = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
			if err != nil {
				t.Fatalf("Failed to clean leaderboard tables: %v", err)
			}

			// Create a new data generator for each test case to avoid ID conflicts
			testDataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

			// Setup test-specific data.
			var initialUsers []testutils.User
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialUsers, initialLeaderboard, setupErr = tt.setupData(deps.BunDB, testDataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			// Call the service method with the new signature
			result, err := deps.Service.ProcessTagAssignments(
				context.Background(),
				tt.serviceParams.guildID,
				tt.serviceParams.source,
				tt.serviceParams.requests,
				tt.serviceParams.requestingUserID,
				tt.serviceParams.operationID,
				tt.serviceParams.batchID,
			)

			// Validate expected error.
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
				// Note: We are not checking for the exact error message here, just that an error occurred.
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			// Validate expected success payload presence.
			if tt.expectedSuccess {
				if result.Success == nil {
					t.Errorf("Expected a success result, but got nil")
				}
			} else {
				if result.Success != nil {
					t.Errorf("Expected no success result, but got: %+v", result.Success)
				}
			}

			// Run test-specific result validation.
			if tt.validateResult != nil {
				tt.validateResult(t, deps, result)
			}

			// Run test-specific database validation.
			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialUsers, initialLeaderboard)
			}
		})
	}
}
