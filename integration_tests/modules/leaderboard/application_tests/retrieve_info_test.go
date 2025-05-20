package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/uptrace/bun"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardService "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

// TestGetLeaderboard tests the GetLeaderboard service function.
func TestGetLeaderboard(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) // Returns the initial active leaderboard
		expectedError   bool                                                                                         // Direct error from serviceWrapper
		expectedSuccess bool                                                                                         // Success result from service
		expectedFailure bool                                                                                         // Failure result from service (business logic)
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, initialLeaderboard *leaderboarddb.Leaderboard)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) // Validates DB state remains unchanged
	}{
		{
			name: "Successful retrieval of active leaderboard with data",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				// Create an active leaderboard with some data
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 1},
						{UserID: "user_B", TagNumber: 2},
						{UserID: "user_C", TagNumber: 3},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				// Create an inactive leaderboard to ensure only the active one is returned
				inactiveLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_Z", TagNumber: 99},
					},
					IsActive:     false,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err = db.NewInsert().Model(inactiveLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			expectedError:   false,
			expectedSuccess: true,
			expectedFailure: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, initialLeaderboard *leaderboarddb.Leaderboard) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.GetLeaderboardResponsePayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.GetLeaderboardResponsePayload, but got %T", result.Success)
					return
				}

				// Compare the returned leaderboard data with the initial active leaderboard data
				expectedData := initialLeaderboard.LeaderboardData
				actualData := successPayload.Leaderboard

				// Sort both slices for consistent comparison
				sort.SliceStable(expectedData, func(i, j int) bool {
					if expectedData[i].TagNumber == 0 && expectedData[j].TagNumber == 0 {
						return false
					}
					if expectedData[i].TagNumber == 0 {
						return true
					}
					if expectedData[j].TagNumber == 0 {
						return false
					}
					return expectedData[i].TagNumber < expectedData[j].TagNumber
				})
				sort.SliceStable(actualData, func(i, j int) bool {
					if actualData[i].TagNumber == 0 && actualData[j].TagNumber == 0 {
						return false
					}
					if actualData[i].TagNumber == 0 {
						return true
					}
					if actualData[j].TagNumber == 0 {
						return false
					}
					return actualData[i].TagNumber < actualData[j].TagNumber
				})

				if len(actualData) != len(expectedData) {
					t.Errorf("Expected %d leaderboard entries, got %d. Actual data: %+v", len(expectedData), len(actualData), actualData)
					return
				}

				for i := range actualData {
					actual := actualData[i]
					expected := expectedData[i]
					// Compare UserID and TagNumber (handling nil pointers)
					if actual.UserID != expected.UserID || !((actual.TagNumber == 0 && expected.TagNumber == 0) || (actual.TagNumber != 0 && expected.TagNumber != 0 && actual.TagNumber == expected.TagNumber)) {
						t.Errorf("Mismatch at index %d: Expected %+v, got %+v", i, expected, actual)
					}
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Verify the database state remains unchanged
				var leaderboards []leaderboarddb.Leaderboard
				err := deps.BunDB.NewSelect().
					Model(&leaderboards).
					Order("id ASC"). // Order for consistent comparison
					Scan(context.Background())
				if err != nil {
					t.Fatalf("Failed to query leaderboards: %v", err)
				}

				// Expecting 2 leaderboards (the initial active and the initial inactive)
				if len(leaderboards) != 2 {
					t.Errorf("Expected 2 leaderboard records in DB, got %d", len(leaderboards))
					return
				}

				// Verify the active status remains correct
				activeCount := 0
				for _, lb := range leaderboards {
					if lb.IsActive {
						activeCount++
						if lb.ID != initialLeaderboard.ID {
							t.Errorf("Expected initial leaderboard (%d) to be active, but found active leaderboard with ID %d", initialLeaderboard.ID, lb.ID)
						}
					}
				}
				if activeCount != 1 {
					t.Errorf("Expected exactly 1 active leaderboard, found %d", activeCount)
				}
			},
		},
		{
			name: "Retrieval when no active leaderboard exists",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				// Ensure no active leaderboard exists
				_, err := db.NewUpdate().
					Model((*leaderboarddb.Leaderboard)(nil)).
					Set("is_active = ?", false).
					Where("is_active = ?", true).
					Exec(context.Background())
				if err != nil {
					// If the table is empty, this won't return an error, which is fine.
					// If there was an error other than no rows updated, we should fail.
					if !errors.Is(err, sql.ErrNoRows) {
						return nil, fmt.Errorf("failed to deactivate existing leaderboards: %w", err)
					}
				}
				return nil, nil // No initial active leaderboard
			},
			expectedError:   false, // Service returns nil error for this business case
			expectedSuccess: false,
			expectedFailure: true, // Expect failure result
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, initialLeaderboard *leaderboarddb.Leaderboard) {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got nil")
					return
				}
				failurePayload, ok := result.Failure.(*leaderboardevents.GetLeaderboardFailedPayload)
				if !ok {
					t.Errorf("Expected failure result of type *leaderboardevents.GetLeaderboardFailedPayload, but got %T", result.Failure)
					return
				}
				expectedReason := leaderboarddb.ErrNoActiveLeaderboard.Error()
				if failurePayload.Reason != expectedReason {
					t.Errorf("Expected failure reason '%s', got '%s'", expectedReason, failurePayload.Reason)
				}
				// In this scenario, the service returns nil for the direct error, so we don't check result.Error
				if result.Error != nil {
					t.Errorf("Expected nil error, but got: %v", result.Error)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Verify no active leaderboards exist
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
				// Note: This test setup might leave inactive leaderboards if they existed before cleanup.
				// We primarily care that no *active* one is found and the DB state isn't altered by the failed get.
			},
		},
		{
			name: "Successful retrieval of active leaderboard with empty data",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				// Create an active leaderboard with empty data
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{}, // Empty data
					IsActive:        true,
					UpdateSource:    leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:        sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			expectedError:   false, // Service should return success with empty data
			expectedSuccess: true,
			expectedFailure: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult, initialLeaderboard *leaderboarddb.Leaderboard) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*leaderboardevents.GetLeaderboardResponsePayload)
				if !ok {
					t.Errorf("Expected success result of type *leaderboardevents.GetLeaderboardResponsePayload, but got %T", result.Success)
					return
				}

				// Expect an empty leaderboard data slice
				if len(successPayload.Leaderboard) != 0 {
					t.Errorf("Expected 0 leaderboard entries, got %d. Actual data: %+v", len(successPayload.Leaderboard), successPayload.Leaderboard)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
				// Verify the database state remains unchanged
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
				} else {
					if activeLeaderboards[0].ID != initialLeaderboard.ID {
						t.Errorf("Expected the initial leaderboard (%d) to still be active, but a different one (%d) is active", initialLeaderboard.ID, activeLeaderboards[0].ID)
					}
					// Verify the data is still empty
					if len(activeLeaderboards[0].LeaderboardData) != 0 {
						t.Errorf("Expected active leaderboard data to be empty, got %d entries", len(activeLeaderboards[0].LeaderboardData))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean the database before each test case
			if err := testutils.CleanLeaderboardIntegrationTables(sharedCtx, deps.BunDB); err != nil {
				t.Fatalf("Failed to clean leaderboard integration tables: %v", err)
			}
			// No user tables cleanup needed for GetLeaderboard tests

			var initialLeaderboard *leaderboarddb.Leaderboard
			var setupErr error
			if tt.setupData != nil {
				initialLeaderboard, setupErr = tt.setupData(deps.BunDB, dataGen)
				if setupErr != nil {
					t.Fatalf("Failed to set up test data: %v", setupErr)
				}
			}

			result, err := deps.Service.GetLeaderboard(sharedCtx)

			// Check for direct error returned by the serviceWrapper
			if tt.expectedError && err == nil {
				t.Errorf("Expected an error, but got none")
			}
			if !tt.expectedError && err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			// Check for success result
			if tt.expectedSuccess && result.Success == nil {
				t.Errorf("Expected a success result, but got nil")
			}
			if !tt.expectedSuccess && result.Success != nil {
				t.Errorf("Expected no success result, but got: %+v", result.Success)
			}

			// Check for failure result
			if tt.expectedFailure && result.Failure == nil {
				t.Errorf("Expected a failure result, but got nil")
			}
			if !tt.expectedFailure && result.Failure != nil {
				t.Errorf("Expected no failure result, but got: %+v", result.Failure)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps, result, initialLeaderboard)
			}

			if tt.validateDB != nil {
				tt.validateDB(t, deps, initialLeaderboard)
			}
		})
	}
}

// TestRoundGetTagByUserID tests the RoundGetTagByUserID service function.
func TestRoundGetTagByUserID(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		payload         sharedevents.RoundTagLookupRequestPayload
		expectedError   bool
		expectedSuccess bool
		expectedFailure bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult)
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful retrieval of tag for user with tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_123", TagNumber: 42},
						{UserID: "user_456", TagNumber: 10},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: sharedevents.RoundTagLookupRequestPayload{
				UserID:     "user_123",
				RoundID:    sharedtypes.RoundID(uuid.New()),
				Response:   "Test Response",
				JoinedLate: boolPtr(false),
			},
			expectedError:   false,
			expectedSuccess: true,
			expectedFailure: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.RoundTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "user_123" {
					t.Errorf("Expected UserID 'user_123', got '%s'", successPayload.UserID)
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("RoundID not echoed correctly")
				}
				if successPayload.OriginalResponse != "Test Response" {
					t.Errorf("Expected OriginalResponse 'Test Response', got '%s'", successPayload.OriginalResponse)
				}
				if (successPayload.OriginalJoinedLate == nil && boolPtr(false) != nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(false) == nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(false) != nil && *successPayload.OriginalJoinedLate != *boolPtr(false)) {
					t.Errorf("Expected OriginalJoinedLate false, got %v", successPayload.OriginalJoinedLate)
				}

				if !successPayload.Found {
					t.Error("Expected Found to be true, but got false")
				}
				if successPayload.TagNumber == nil {
					t.Error("Expected TagNumber to be non-nil, but got nil")
					return
				}
				if *successPayload.TagNumber != 42 {
					t.Errorf("Expected TagNumber 42, got %d", *successPayload.TagNumber)
				}
				if successPayload.Error != "" {
					t.Errorf("Expected empty Error, got '%s'", successPayload.Error)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
			},
		},
		{
			name: "User exists but has no tag number",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_no_tag", TagNumber: 0},
						{UserID: "user_with_tag", TagNumber: 5},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: sharedevents.RoundTagLookupRequestPayload{
				UserID:     "user_no_tag",
				RoundID:    sharedtypes.RoundID(uuid.New()),
				Response:   "Another Response",
				JoinedLate: boolPtr(true),
			},
			expectedError:   false,
			expectedSuccess: true,
			expectedFailure: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.RoundTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "user_no_tag" {
					t.Errorf("Expected UserID 'user_no_tag', got '%s'", successPayload.UserID)
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("RoundID not echoed correctly")
				}
				if successPayload.OriginalResponse != "Another Response" {
					t.Errorf("Expected OriginalResponse 'Another Response', got '%s'", successPayload.OriginalResponse)
				}
				if (successPayload.OriginalJoinedLate == nil && boolPtr(true) != nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(true) == nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(true) != nil && *successPayload.OriginalJoinedLate != *boolPtr(true)) {
					t.Errorf("Expected OriginalJoinedLate true, got %v", successPayload.OriginalJoinedLate)
				}

				if successPayload.Found {
					t.Error("Expected Found to be false, but got true")
				}
				if successPayload.TagNumber != nil {
					t.Errorf("Expected TagNumber to be nil, but got %v", successPayload.TagNumber)
				}
				// Updated check to expect the specific error message
				expectedErrorMsg := sql.ErrNoRows.Error()
				if successPayload.Error != expectedErrorMsg {
					t.Errorf("Expected Error '%s', got '%s'", expectedErrorMsg, successPayload.Error)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
			},
		},
		{
			name: "User ID not found in database",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_abc", TagNumber: 11},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: sharedevents.RoundTagLookupRequestPayload{
				UserID:     "non_existent_user",
				RoundID:    sharedtypes.RoundID(uuid.New()),
				Response:   "Yet Another Response",
				JoinedLate: boolPtr(false),
			},
			expectedError:   false,
			expectedSuccess: true,
			expectedFailure: false,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.RoundTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.RoundTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "non_existent_user" {
					t.Errorf("Expected UserID 'non_existent_user', got '%s'", successPayload.UserID)
				}
				if successPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("RoundID not echoed correctly")
				}
				if successPayload.OriginalResponse != "Yet Another Response" {
					t.Errorf("Expected OriginalResponse 'Yet Another Response', got '%s'", successPayload.OriginalResponse)
				}
				if (successPayload.OriginalJoinedLate == nil && boolPtr(false) != nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(false) == nil) || (successPayload.OriginalJoinedLate != nil && boolPtr(false) != nil && *successPayload.OriginalJoinedLate != *boolPtr(false)) {
					t.Errorf("Expected OriginalJoinedLate false, got %v", successPayload.OriginalJoinedLate)
				}

				if successPayload.Found {
					t.Error("Expected Found to be false, but got true")
				}
				if successPayload.TagNumber != nil {
					t.Errorf("Expected TagNumber to be nil, but got %v", successPayload.TagNumber)
				}
				// Updated check to expect the specific error message
				expectedErrorMsg := sql.ErrNoRows.Error()
				if successPayload.Error != expectedErrorMsg {
					t.Errorf("Expected Error '%s', got '%s'", expectedErrorMsg, successPayload.Error)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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

			result, err := deps.Service.RoundGetTagByUserID(sharedCtx, tt.payload)

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

			if tt.expectedFailure && result.Failure == nil {
				t.Errorf("Expected a failure result, but got nil")
			}
			if !tt.expectedFailure && result.Failure != nil {
				t.Errorf("Expected no failure result, but got: %+v", result.Failure)
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

func TestGetTagByUserID(t *testing.T) {
	deps := SetupTestLeaderboardService(sharedCtx, sharedDB, t)
	defer deps.Cleanup()

	dataGen := testutils.NewTestDataGenerator(time.Now().UnixNano())

	tests := []struct {
		name            string
		setupData       func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error)
		payload         struct{ UserID sharedtypes.DiscordID } // Updated payload struct in test
		expectedError   bool
		expectedSuccess bool
		validateResult  func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) // Use the correct type
		validateDB      func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard)
	}{
		{
			name: "Successful retrieval of tag for user with tag",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_A", TagNumber: 99},
						{UserID: "user_B", TagNumber: 88},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: struct{ UserID sharedtypes.DiscordID }{ // Updated payload in test case
				UserID: "user_A",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.DiscordTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.DiscordTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "user_A" {
					t.Errorf("Expected UserID 'user_A', got '%s'", successPayload.UserID)
				}
				// RoundID validation removed

				if !successPayload.Found {
					t.Error("Expected Found to be true, but got false")
				}
				if successPayload.TagNumber == nil {
					t.Error("Expected TagNumber to be non-nil, but got nil")
					return
				}
				if *successPayload.TagNumber != 99 {
					t.Errorf("Expected TagNumber 99, got %d", *successPayload.TagNumber)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
			},
		},
		{
			name: "User exists but has no tag number",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_C", TagNumber: 0}, // User exists with nil tag
						{UserID: "user_D", TagNumber: 77},
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: struct{ UserID sharedtypes.DiscordID }{ // Updated payload in test case
				UserID: "user_C",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.DiscordTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.DiscordTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "user_C" {
					t.Errorf("Expected UserID 'user_C', got '%s'", successPayload.UserID)
				}
				// RoundID validation removed

				if successPayload.Found {
					t.Error("Expected Found to be false, but got true")
				}
				if successPayload.TagNumber != nil {
					t.Errorf("Expected TagNumber to be nil, but got %v", successPayload.TagNumber)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
			},
		},
		{
			name: "User ID not found in database",
			setupData: func(db *bun.DB, generator *testutils.TestDataGenerator) (*leaderboarddb.Leaderboard, error) {
				initialLeaderboard := &leaderboarddb.Leaderboard{
					LeaderboardData: leaderboardtypes.LeaderboardData{
						{UserID: "user_E", TagNumber: 66}, // User E exists, but not the target user
					},
					IsActive:     true,
					UpdateSource: leaderboarddb.ServiceUpdateSourceManual,
					UpdateID:     sharedtypes.RoundID(uuid.New()),
				}
				_, err := db.NewInsert().Model(initialLeaderboard).Exec(context.Background())
				if err != nil {
					return nil, err
				}
				return initialLeaderboard, nil
			},
			payload: struct{ UserID sharedtypes.DiscordID }{ // Updated payload in test case
				UserID: "non_existent_user_2",
			},
			expectedError:   false,
			expectedSuccess: true,
			validateResult: func(t *testing.T, deps TestDeps, result leaderboardService.LeaderboardOperationResult) {
				if result.Success == nil {
					t.Errorf("Expected success result, but got nil")
					return
				}
				successPayload, ok := result.Success.(*sharedevents.DiscordTagLookupResultPayload)
				if !ok {
					t.Errorf("Expected success result of type *sharedevents.DiscordTagLookupResultPayload, but got %T", result.Success)
					return
				}

				if successPayload.UserID != "non_existent_user_2" {
					t.Errorf("Expected UserID 'non_existent_user_2', got '%s'", successPayload.UserID)
				}
				// RoundID validation removed

				if successPayload.Found {
					t.Error("Expected Found to be false, but got true")
				}
				if successPayload.TagNumber != nil {
					t.Errorf("Expected TagNumber to be nil, but got %v", successPayload.TagNumber)
				}
			},
			validateDB: func(t *testing.T, deps TestDeps, initialLeaderboard *leaderboarddb.Leaderboard) {
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
			},
		},
		// Add more test cases for other scenarios handled by the service method
		// e.g., No active leaderboard, database errors, etc.
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

			// Call the service method with the UserID directly
			result, err := deps.Service.GetTagByUserID(sharedCtx, tt.payload.UserID)

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
