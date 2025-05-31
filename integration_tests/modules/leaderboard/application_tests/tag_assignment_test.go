package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardService "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func TestTagAssignmentRequested(t *testing.T) {
	deps := SetupTestLeaderboardService(t)

	tests := []struct {
		name            string
		setupData       func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagAssignmentRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag assignment to a new user",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users for initial leaderboard
				if err := testutils.InsertUser(t, db, "tag_assign_existing_user_1", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "tag_assign_existing_user_2", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "tag_assign_existing_user_1", TagNumber: 1},
						{UserID: "tag_assign_existing_user_2", TagNumber: 2},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "tag_assign_new_user_1",
				TagNumber:  tagPtr(3),
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) {
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
					t.Errorf("Expected 1 assignment, got %d", successPayload.AssignmentCount)
					return
				}

				if len(successPayload.Assignments) != 1 {
					t.Errorf("Expected 1 assignment in payload, got %d", len(successPayload.Assignments))
					return
				}

				assignment := successPayload.Assignments[0]
				if assignment.UserID != payload.UserID {
					t.Errorf("Expected UserID '%s', got '%s'", payload.UserID, assignment.UserID)
				}
				if assignment.TagNumber != *payload.TagNumber {
					t.Errorf("Expected TagNumber %v, got %v", *payload.TagNumber, assignment.TagNumber)
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
					t.Errorf("Expected 3 leaderboard entries, got %d", len(leaderboardEntries))
					return
				}

				foundNewUserTag := false
				foundExistingUser1 := false
				foundExistingUser2 := false

				for _, entry := range leaderboardEntries {
					if entry.UserID == "tag_assign_new_user_1" && entry.TagNumber == 3 {
						foundNewUserTag = true
					}
					if entry.UserID == "tag_assign_existing_user_1" && entry.TagNumber == 1 {
						foundExistingUser1 = true
					}
					if entry.UserID == "tag_assign_existing_user_2" && entry.TagNumber == 2 {
						foundExistingUser2 = true
					}
				}

				if !foundNewUserTag {
					t.Error("New user with assigned tag not found in the new leaderboard data")
				}
				if !foundExistingUser1 || !foundExistingUser2 {
					t.Error("Existing users/tags not found in the new leaderboard data")
				}
			},
		},
		{
			name: "Tag assignment fails if tag is already assigned",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				// Create users for leaderboard
				if err := testutils.InsertUser(t, db, "tag_conflict_user_a", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}
				if err := testutils.InsertUser(t, db, "tag_conflict_user_b", sharedtypes.UserRoleRattler); err != nil {
					return nil, err
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "tag_conflict_user_a", TagNumber: 10},
						{UserID: "tag_conflict_user_b", TagNumber: 20},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "tag_conflict_user_c",
				TagNumber:  tagPtr(10), // Tag 10 is already assigned to tag_conflict_user_a
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.BatchTagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if !strings.Contains(strings.ToLower(failurePayload.Reason), "already assigned") {
					t.Errorf("Expected failure reason to contain 'already assigned', got '%s'", failurePayload.Reason)
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
					t.Errorf("Expected leaderboard to remain unchanged for tag conflict")
				}
				if !activeLeaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
			},
		},
		{
			name: "Tag assignment fails if no active leaderboard exists",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}
				return nil, nil
			},
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "tag_no_board_user_d",
				TagNumber:  tagPtr(4),
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) {
				// This case expects an error from the service call itself
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Where("is_active = ?", true).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}
				if len(leaderboards) != 0 {
					t.Errorf("Expected 0 active leaderboard records in DB, got %d", len(leaderboards))
				}
			},
		},
		{
			name: "Tag assignment fails for invalid tag number (e.g., negative)",
			setupData: func(t *testing.T, db *bun.DB) (*leaderboarddb.Leaderboard, error) {
				// Ensure clean state
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil && err != sql.ErrNoRows {
					return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
				}

				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "tag_negative_user_e",
				TagNumber:  tagPtr(-5), // Invalid tag number
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.BatchTagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.BatchTagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if !strings.Contains(strings.ToLower(failurePayload.Reason), "invalid tag number") {
					t.Errorf("Expected failure reason to contain 'invalid tag number', got '%s'", failurePayload.Reason)
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
					t.Errorf("Expected leaderboard to remain unchanged for invalid input")
				}
				if !activeLeaderboard.IsActive {
					t.Error("Expected leaderboard to remain active")
				}
				if len(activeLeaderboard.LeaderboardData) != 0 {
					t.Errorf("Expected leaderboard data to be empty, got %d entries", len(activeLeaderboard.LeaderboardData))
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
			tagAssignmentRequests := []sharedtypes.TagAssignmentRequest{
				{
					UserID:    tt.payload.UserID,
					TagNumber: *tt.payload.TagNumber,
				},
			}

			// Call the service
			result, err := deps.Service.ProcessTagAssignments(
				ctx,
				sharedtypes.ServiceUpdateSourceManual, // Correct source type
				tagAssignmentRequests,
				nil,                        // No requesting user for manual assignments
				tt.payload.UpdateID.UUID(), // Operation ID
				tt.payload.UpdateID.UUID(), // Batch ID
			)

			// Check for error return value from the service function
			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// Check for success result payload
			if tt.expectedSuccess && result.Success == nil {
				t.Errorf("Expected a success result payload, but got nil")
			}
			if !tt.expectedSuccess && result.Success != nil {
				t.Errorf("Expected no success result payload, but got: %+v", result.Success)
			}

			// Validate the content of the result payload (either success or failure)
			if tt.validateResult != nil {
				tt.validateResult(t, deps, result, tt.payload)
			}

			// Validate the database state
			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}
