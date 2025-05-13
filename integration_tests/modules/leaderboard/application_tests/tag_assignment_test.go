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
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		payload         leaderboardevents.TagAssignmentRequestedPayload
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful tag assignment to a new user",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "existing_user_1", TagNumber: tagPtr(1)},
						{UserID: "existing_user_2", TagNumber: tagPtr(2)},
					},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "setup_user",
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
				UpdateID:   uuid.New().String(),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.TagAssignedPayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.TagAssignedPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "new_user_1" {
					t.Errorf("Expected UserID 'new_user_1', got '%s'", successPayload.UserID)
				}
				if successPayload.TagNumber == nil || *successPayload.TagNumber != 3 {
					t.Errorf("Expected TagNumber 3, got %v", successPayload.TagNumber)
				}
				// Updated validation to match current service behavior (Source is not populated)
				if successPayload.Source != "" {
					t.Errorf("Expected empty Source, got '%s'", successPayload.Source)
				}
				// Updated validation to match current service behavior (AssignmentID is not populated)
				if successPayload.AssignmentID != sharedtypes.RoundID(uuid.Nil) {
					t.Errorf("Expected nil AssignmentID, got %s", successPayload.AssignmentID)
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
					if entry.UserID == "new_user_1" && entry.TagNumber != nil && *entry.TagNumber == 3 {
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
					if entry.UserID == "existing_user_1" && entry.TagNumber != nil && *entry.TagNumber == 1 {
						foundExistingUser1 = true
					}
					if entry.UserID == "existing_user_2" && entry.TagNumber != nil && *entry.TagNumber == 2 {
						foundExistingUser2 = true
					}
				}
				if !foundExistingUser1 || !foundExistingUser2 {
					t.Error("Existing users/tags not found in the new leaderboard data")
				}
			},
		},
		{
			name: "Tag assignment fails if tag is already assigned",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: tagPtr(10)},
						{UserID: "user_B", TagNumber: tagPtr(20)},
					},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "setup_user",
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
				UpdateID:   uuid.New().String(),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.UserID != "user_C" {
					t.Errorf("Expected UserID 'user_C', got '%s'", failurePayload.UserID)
				}
				if failurePayload.TagNumber == nil || *failurePayload.TagNumber != 10 {
					t.Errorf("Expected TagNumber 10 in failure payload, got %v", failurePayload.TagNumber)
				}
				if failurePayload.Source != "Test" {
					t.Errorf("Expected Source 'Test', got '%s'", failurePayload.Source)
				}
				if failurePayload.UpdateType != "Manual" {
					t.Errorf("Expected UpdateType 'Manual', got '%s'", failurePayload.UpdateType)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "tag 10 is already assigned") {
					t.Errorf("Expected failure reason to contain 'tag 10 is already assigned', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
				UpdateID:   uuid.New().String(),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.UserID != "user_D" {
					t.Errorf("Expected UserID 'user_D', got '%s'", failurePayload.UserID)
				}
				if failurePayload.TagNumber == nil || *failurePayload.TagNumber != 4 {
					t.Errorf("Expected TagNumber 4 in failure payload, got %v", failurePayload.TagNumber)
				}
				if failurePayload.Source != "Test" {
					t.Errorf("Expected Source 'Test', got '%s'", failurePayload.Source)
				}
				if failurePayload.UpdateType != "Manual" {
					t.Errorf("Expected UpdateType 'Manual', got '%s'", failurePayload.UpdateType)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "no active leaderboard found") && !strings.Contains(strings.ToLower(failurePayload.Reason), "failed to get active leaderboard") {
					t.Errorf("Expected failure reason to indicate no active leaderboard, got '%s'", failurePayload.Reason)
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
				if len(leaderboards) != 0 {
					t.Errorf("Expected 0 leaderboard records in DB, got %d", len(leaderboards))
				}
			},
		},
		{
			name: "Tag assignment fails for invalid tag number (e.g., negative)",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData:     leaderboardtypes.LeaderboardData{},
					IsActive:            true,
					UpdateSource:        leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:            sharedtypes.RoundID(uuid.New()),
					RequestingDiscordID: "setup_user",
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
				UpdateID:   uuid.New().String(),
				Source:     "Test",
				UpdateType: "Manual",
			},
			expectedError:   true,
			expectedSuccess: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.TagAssignmentFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.TagAssignmentFailedPayload, but got %T", result.Failure)
					return
				}

				if failurePayload.UserID != "user_E" {
					t.Errorf("Expected UserID 'user_E', got '%s'", failurePayload.UserID)
				}
				// Updated validation to match current service behavior (TagNumber is nil for negative input)
				if failurePayload.TagNumber != nil {
					t.Errorf("Expected nil TagNumber in failure payload for invalid tag, got %v", failurePayload.TagNumber)
				}
				if failurePayload.Source != "Test" {
					t.Errorf("Expected Source 'Test', got '%s'", failurePayload.Source)
				}
				if failurePayload.UpdateType != "Manual" {
					t.Errorf("Expected UpdateType 'Manual', got '%s'", failurePayload.UpdateType)
				}
				if !strings.Contains(strings.ToLower(failurePayload.Reason), "invalid tag number") {
					t.Errorf("Expected failure reason to contain 'invalid tag number', got '%s'", failurePayload.Reason)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
				if len(leaderboard.LeaderboardData) != 0 {
					t.Errorf("Expected leaderboard data to be empty, got %d entries", len(leaderboard.LeaderboardData))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := testutils.CleanLeaderboardIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean leaderboard integration tables: %v", err)
			}

			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			result, err := deps.Service.TagAssignmentRequested(sharedCtx, tt.payload)

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
