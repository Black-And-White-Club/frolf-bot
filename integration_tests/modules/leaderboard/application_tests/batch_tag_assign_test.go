package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
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

// TestProcessTagAssignments is an integration test for the ProcessTagAssignments service method.
// It tests the service's logic and its interaction with the database.
func TestProcessTagAssignments(t *testing.T) {
	// Setup the test environment dependencies
	deps := SetupTestLeaderboardService(t)

	tests := []struct {
		name          string
		setupData     func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error)
		serviceParams struct {
			source           sharedtypes.ServiceUpdateSource
			requests         []sharedtypes.TagAssignmentRequest
			requestingUserID *sharedtypes.DiscordID
			operationID      uuid.UUID
			batchID          uuid.UUID
		}
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful batch assignment",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state - deactivate any existing leaderboards
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users using testutils.InsertUser helper (proper way)
				if err := testutils.InsertUser(t, db, "batch_success_user_1", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "batch_success_user_2", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "batch_success_user_3", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				// Insert an initial active leaderboard record
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{}, // Start with empty data
					IsActive:        true,
					UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}

				return initialLeaderboard, nil
			},
			serviceParams: struct {
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				source: sharedtypes.ServiceUpdateSourceAdminBatch,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "batch_success_user_1", TagNumber: 1},
					{UserID: "batch_success_user_2", TagNumber: 2},
					{UserID: "batch_success_user_3", TagNumber: 3},
				},
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
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
					"batch_success_user_1": 1,
					"batch_success_user_2": 2,
					"batch_success_user_3": 3,
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
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 3 {
					t.Errorf("Expected 3 leaderboard entries in active leaderboard data, got %d", len(leaderboardEntries))
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"batch_success_user_1": 1,
					"batch_success_user_2": 2,
					"batch_success_user_3": 3,
				}

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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state - deactivate any existing leaderboards
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users using proper helper
				if err := testutils.InsertUser(t, db, "batch_mixed_user_a", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "batch_mixed_user_c", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				// Insert an initial active leaderboard with some existing data
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "batch_mixed_initial", TagNumber: 99},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}

				return initialLeaderboard, nil
			},
			serviceParams: struct {
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				source: sharedtypes.ServiceUpdateSourceAdminBatch,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "batch_mixed_user_a", TagNumber: 10}, // Existing user, valid tag
					{UserID: "batch_mixed_user_b", TagNumber: -5}, // Non-existent user, invalid tag (should be skipped)
					{UserID: "batch_mixed_user_c", TagNumber: 11}, // Existing user, valid tag
					{UserID: "batch_mixed_user_d", TagNumber: 12}, // Non-existent user, valid tag (should be added)
				},
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
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
				expectedProcessedCount := 3 // Number of assignments with TagNumber >= 0
				if successPayload.AssignmentCount != expectedProcessedCount {
					t.Errorf("Expected %d assignments in success payload, got %d", expectedProcessedCount, successPayload.AssignmentCount)
				}
				expectedAssignments := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"batch_mixed_user_a": 10,
					"batch_mixed_user_c": 11,
					"batch_mixed_user_d": 12,
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
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				expectedDBEntries := 4 // user_a (10), user_c (11), user_d (12), initial (99)
				if len(leaderboardEntries) != expectedDBEntries {
					t.Errorf("Expected %d leaderboard entries in active leaderboard data, got %d", expectedDBEntries, len(leaderboardEntries))
					return
				}

				expectedDBState := map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"batch_mixed_user_a":  10,
					"batch_mixed_user_c":  11,
					"batch_mixed_user_d":  12,
					"batch_mixed_initial": 99,
				}

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
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state - deactivate any existing leaderboards
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create user using proper helper
				if err := testutils.InsertUser(t, db, "batch_empty_existing_user", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "batch_empty_existing_user", TagNumber: 42},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}

				return initialLeaderboard, nil
			},
			serviceParams: struct {
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				source:           sharedtypes.ServiceUpdateSourceAdminBatch,
				requests:         []sharedtypes.TagAssignmentRequest{}, // Empty list
				requestingUserID: func() *sharedtypes.DiscordID { id := sharedtypes.DiscordID("test_admin_user"); return &id }(),
				operationID:      uuid.New(),
				batchID:          uuid.New(),
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
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboard.ID)
				}

				// For empty assignments, we should only have 1 active leaderboard (the initial one)
				var activeLeaderboards []leaderboarddb.Leaderboard
				err = deps.BunDB.NewSelect().
					Model(&activeLeaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboards: %v", err)
				}
				if len(activeLeaderboards) != 1 {
					t.Errorf("Expected only one active leaderboard record after empty assignments, got %d", len(activeLeaderboards))
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 1 || leaderboardEntries[0].TagNumber != 42 || leaderboardEntries[0].UserID != "batch_empty_existing_user" {
					t.Errorf("Expected 1 leaderboard entry for batch_empty_existing_user with tag 42, got %d entries or wrong data %v", len(leaderboardEntries), leaderboardEntries)
				}
			},
		},
		{
			name: "Single assignment that requires tag swap",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state - deactivate any existing leaderboards
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users using proper helper
				if err := testutils.InsertUser(t, db, "batch_conflict_with_tag", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "batch_conflict_requesting", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				// Insert an initial active leaderboard with one user having tag 1
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "batch_conflict_with_tag", TagNumber: 1},
					},
					IsActive:     true,
					UpdateSource: sharedtypes.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}

				return initialLeaderboard, nil
			},
			serviceParams: struct {
				source           sharedtypes.ServiceUpdateSource
				requests         []sharedtypes.TagAssignmentRequest
				requestingUserID *sharedtypes.DiscordID
				operationID      uuid.UUID
				batchID          uuid.UUID
			}{
				source: sharedtypes.ServiceUpdateSourceManual,
				requests: []sharedtypes.TagAssignmentRequest{
					{UserID: "batch_conflict_requesting", TagNumber: 1}, // This should trigger a failure result
				},
				requestingUserID: nil, // Individual assignment
				operationID:      uuid.New(),
				batchID:          uuid.New(),
			},
			expectedError:   false, // No Go error - service returns failure result instead
			expectedSuccess: false, // Don't expect a success payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success != nil {
					t.Errorf("Expected no success result for tag conflict, but got: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Errorf("Expected failure result for tag conflict, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.BatchTagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}
				if failurePayload.RequestingUserID != "system" {
					t.Errorf("Expected requesting user ID 'system', got %s", failurePayload.RequestingUserID)
				}
				if failurePayload.Reason == "" {
					t.Errorf("Expected non-empty failure reason")
				}
				expectedReasonSubstring := "tag 1 is already assigned to user batch_conflict_with_tag"
				if failurePayload.Reason != expectedReasonSubstring {
					t.Errorf("Expected failure reason to be '%s', got '%s'", expectedReasonSubstring, failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var activeLeaderboard leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&activeLeaderboard).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query active leaderboard: %v", err)
				}

				if activeLeaderboard.ID != initialLeaderboard.ID {
					t.Errorf("Expected the initial leaderboard to still be active for tag conflict, but a different one is active")
				}

				leaderboardEntries := activeLeaderboard.LeaderboardData
				if len(leaderboardEntries) != 1 || leaderboardEntries[0].UserID != "batch_conflict_with_tag" || leaderboardEntries[0].TagNumber != 1 {
					t.Errorf("Expected unchanged leaderboard data for tag conflict, got %v", leaderboardEntries)
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

			// Call the service method
			result, err := deps.Service.ProcessTagAssignments(
				context.Background(),
				tt.serviceParams.source,
				tt.serviceParams.requests,
				tt.serviceParams.requestingUserID,
				tt.serviceParams.operationID,
				tt.serviceParams.batchID,
			)

			// Validate expected error
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			// Validate expected success payload presence
			if tt.expectedSuccess {
				if result.Success == nil {
					t.Errorf("Expected a success result, but got nil")
				}
			} else {
				if result.Success != nil {
					t.Errorf("Expected no success result, but got: %+v", result.Success)
				}
			}

			// Run test-specific result validation
			if tt.validateResult != nil {
				tt.validateResult(t, deps, result)
			}

			// Run test-specific database validation
			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}
