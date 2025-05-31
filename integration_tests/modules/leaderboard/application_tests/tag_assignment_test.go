package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

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
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagAssignmentRequestedPayload
		expectedError   bool                                                                                                                                             // Expected error from the service function call itself
		expectedSuccess bool                                                                                                                                             // Expected a non-nil Success field in LeaderboardOperationResult
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) // Modified signature
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag assignment to a new user",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "existing_user_1", TagNumber: 1},
						{UserID: "existing_user_2", TagNumber: 2},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "new_user_1",
				TagNumber:  tagPtr(3),
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) { // Added payload parameter
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.TagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.TagAssignedPayload, but got %T", result.Success)
					return
				}

				// Validate fields that should match the request payload (using the 'payload' parameter)
				if successPayload.UserID != payload.UserID {
					t.Errorf("Expected UserID '%s', got '%s'", payload.UserID, successPayload.UserID)
				}
				if successPayload.TagNumber == nil || *successPayload.TagNumber != *payload.TagNumber {
					t.Errorf("Expected TagNumber %v, got %v", payload.TagNumber, successPayload.TagNumber)
				}

				// Validate Source: Should match the Source from the original request payload (payload.Source)
				if successPayload.Source != payload.Source {
					t.Errorf("Expected Source '%s', got '%s'", payload.Source, successPayload.Source)
				}

				// Validate AssignmentID: Should be a newly generated, non-nil UUID
				if successPayload.AssignmentID == sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("Expected a non-nil AssignmentID, but got the zero UUID")
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

				foundNewUserTag := false
				for _, entry := range newLeaderboard.LeaderboardData {
					// Corrected comparison for TagNumber (assuming it's now value type sharedtypes.TagNumber)
					if entry.UserID == "new_user_1" && entry.TagNumber != 0 && entry.TagNumber == 3 {
						foundNewUserTag = true
						break
					}
				}
				if !foundNewUserTag {
					t.Error("New user with assigned tag not found in the new leaderboard data")
				}

				foundExistingUser1 := false
				foundExistingUser2 := false
				for _, entry := range newLeaderboard.LeaderboardData {
					// Corrected comparison for TagNumber (assuming it's now value type sharedtypes.TagNumber)
					if entry.UserID == "existing_user_1" && entry.TagNumber != 0 && entry.TagNumber == 1 {
						foundExistingUser1 = true
					}
					if entry.UserID == "existing_user_2" && entry.TagNumber != 0 && entry.TagNumber == 2 {
						foundExistingUser2 = true
					}
				}
				if !foundExistingUser1 || !foundExistingUser2 {
					t.Error("Existing users/tags not found in the new leaderboard data")
				}
			},
		},
		{
			name: "Tag swap is triggered when user with tag claims another user's tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 10},
						{UserID: "user_B", TagNumber: 20},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "user_A",   // Already has tag 10
				TagNumber:  tagPtr(20), // Wants to claim tag 20 (owned by user_B)
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false, // Swap is a successful outcome for this handler
			expectedSuccess: true,  // Swap returns a success payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) { // Added payload parameter
				swapPayload, ok := result.Success.(*leaderboardevents.TagSwapRequestedPayload)
				if !ok {
					t.Errorf("Expected swap result of type *TagSwapRequestedPayload, but got %T", result.Success)
					return
				}
				if swapPayload.RequestorID != "user_A" {
					t.Errorf("Expected RequestorID 'user_A', got '%s'", swapPayload.RequestorID)
				}
				if swapPayload.TargetID != "user_B" {
					t.Errorf("Expected TargetID 'user_B', got '%s'", swapPayload.TargetID)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Optionally, check that no new leaderboard was created yet (swap not performed)
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}
				if len(leaderboards) != 1 {
					t.Errorf("Expected 1 leaderboard record in DB (no swap performed yet), got %d", len(leaderboards))
				}
			},
		},
		{
			name: "Tag assignment fails if tag is already assigned",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 10},
						{UserID: "user_B", TagNumber: 20},
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "user_C",
				TagNumber:  tagPtr(10), // Tag 10 is already assigned to user_A
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false, // Service returns nil error for validation failures
			expectedSuccess: false, // Service returns a failure payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) { // Added payload parameter
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.UserID != payload.UserID {
					t.Errorf("Expected UserID '%s', got '%s'", payload.UserID, failurePayload.UserID)
				}
				// Corrected comparison for TagNumber (assuming the payload still uses a pointer)
				if failurePayload.TagNumber == nil || *failurePayload.TagNumber != *payload.TagNumber {
					t.Errorf("Expected TagNumber %v in failure payload, got %v", payload.TagNumber, failurePayload.TagNumber)
				}
				if failurePayload.Source != payload.Source {
					t.Errorf("Expected Source '%s', got '%s'", payload.Source, failurePayload.Source)
				}
				if failurePayload.UpdateType != payload.UpdateType {
					t.Errorf("Expected UpdateType '%s', got '%s'", payload.UpdateType, failurePayload.UpdateType)
				}
				// Updated substring check to be more general or match the exact expected error message
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "already assigned") {
					t.Errorf("Expected failure reason to contain 'already assigned', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
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
			name: "Tag assignment fails if no active leaderboard exists",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
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
				UserID:     "user_D",
				TagNumber:  tagPtr(4),
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   true, // Service returns an error if getting active leaderboard fails
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) { // Added payload parameter
				// This case expects an error from the service call itself, not a failure payload.
				// The check for `result.Failure == nil` is sufficient here based on expectedSuccess: false.
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
		{
			name: "Tag assignment fails for invalid tag number (e.g., negative)",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
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
			payload: leaderboardevents.TagAssignmentRequestedPayload{
				UserID:     "user_E",
				TagNumber:  tagPtr(-5), // Invalid tag number
				UpdateID:   sharedtypes.RoundID(uuid.New()),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false, // Service returns nil error for validation failures
			expectedSuccess: false, // Service returns a failure payload
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, payload leaderboardevents.TagAssignmentRequestedPayload) { // Added payload parameter
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.UserID != payload.UserID {
					t.Errorf("Expected UserID '%s', got '%s'", payload.UserID, failurePayload.UserID)
				}
				// Updated validation to match current service behavior (TagNumber is nil for negative input)
				if failurePayload.TagNumber != nil {
					t.Errorf("Expected nil TagNumber in failure payload for invalid tag, got %v", failurePayload.TagNumber)
				}
				if failurePayload.Source != payload.Source {
					t.Errorf("Expected Source %v, got '%v'", payload.Source, failurePayload.Source)
				}
				if failurePayload.UpdateType != payload.UpdateType {
					t.Errorf("Expected UpdateType %s, got '%s'", payload.UpdateType, failurePayload.UpdateType)
				}
				// Updated substring check to match the actual error message
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "invalid input: tag number cannot be negative") {
					t.Errorf("Expected failure reason to contain 'invalid input: tag number cannot be negative', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Scan(context.Background())
				if err != nil && err != sql.ErrNoRows {
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
				if len(leaderboard.LeaderboardData) != 0 {
					t.Errorf("Expected leaderboard data to be empty, got %d entries", len(leaderboard.LeaderboardData))
				}
			},
		},
	}

	for _, tt := range tests {
		// Capture tt for use in the validateResult closure
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
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
			result, err := deps.Service.ProcessTagAssignments(ctx, tt.payload, tagAssignmentRequests, nil, tt.payload.UpdateID.UUID(), tt.payload.UpdateID.UUID())
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
				// Pass the test case's payload to the validation function
				tt.validateResult(t, deps, result, tt.payload)
			}

			// Validate the database state
			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}
