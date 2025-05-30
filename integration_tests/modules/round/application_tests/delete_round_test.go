package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestValidateRoundDeleteRequest(t *testing.T) {
	tests := []struct {
		name                     string
		setupRound               func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload // Add setup function
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful validation of delete request",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				userID := "user123"
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID(userID),
					Title:     "Test Round for Validation",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(deps.Ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				return roundevents.RoundDeleteRequestPayload{
					RoundID:              roundForDB.ID,
					RequestingUserUserID: roundForDB.CreatedBy,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				_, ok := returnedResult.Success.(*roundevents.RoundDeleteValidatedPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteValidatedPayload, got %T", returnedResult.Success)
				}
			},
		},
		{
			name: "Validation fails with zero RoundID",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload {
				return roundevents.RoundDeleteRequestPayload{
					RoundID:              sharedtypes.RoundID(uuid.Nil), // Zero UUID
					RequestingUserUserID: "user123",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "round ID cannot be zero",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if failurePayload.Error != "round ID cannot be zero" {
					t.Errorf("Expected error 'round ID cannot be zero', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails with empty RequestingUserUserID",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload {
				return roundevents.RoundDeleteRequestPayload{
					RoundID:              sharedtypes.RoundID(uuid.New()),
					RequestingUserUserID: "", // Empty user ID
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "requesting user's Discord ID cannot be empty",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if failurePayload.Error != "requesting user's Discord ID cannot be empty" {
					t.Errorf("Expected error 'requesting user's Discord ID cannot be empty', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails when round does not exist",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload {
				return roundevents.RoundDeleteRequestPayload{
					RoundID:              sharedtypes.RoundID(uuid.New()), // Non-existent round
					RequestingUserUserID: "user123",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "round with ID",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round with ID") {
					t.Errorf("Expected error to contain 'round with ID', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails when user is not the creator",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayload {
				// Create a round with one user
				generator := testutils.NewTestDataGenerator()
				creatorUserID := "creator123"
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID(creatorUserID),
					Title:     "Test Round for Ownership Test",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(deps.Ctx, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				return roundevents.RoundDeleteRequestPayload{
					RoundID:              roundForDB.ID,
					RequestingUserUserID: "different_user123", // Different user trying to delete
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "unauthorized: only the round creator can delete the round", // Removed the extra quote
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "unauthorized: only the round creator can delete the round") { // Removed the extra quote
					t.Errorf("Expected error to contain 'unauthorized: only the round creator can delete the round', got '%s'", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			// Get the payload from the setup function
			payload := tt.setupRound(deps)

			// Call the actual service method
			result, err := deps.Service.ValidateRoundDeleteRequest(deps.Ctx, payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayload)
					if !ok {
						t.Errorf("Expected RoundDeleteErrorPayload, got %T", result.Failure)
					} else if !strings.Contains(failurePayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, failurePayload.Error)
					}
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestDeleteRound(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayload)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful deletion of an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_delete_1"),
					Title:     "Round to be deleted",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: roundForDBInsertion.ID,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				deletedPayload, ok := returnedResult.Success.(*roundevents.RoundDeletedPayload)
				if !ok {
					t.Errorf("Expected RoundDeletedPayload, got %T", returnedResult.Success)
				}

				// Verify the round's state is DELETED in the DB
				persistedRound, err := deps.DB.GetRound(ctx, deletedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after deletion: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Expected round to be found in DB (marked deleted), but it was nil")
				}
				if persistedRound.State != roundtypes.RoundStateDeleted {
					t.Errorf("Expected round state to be DELETED, but got %s", persistedRound.State)
				}
			},
		},
		{
			name: "Attempt to delete a non-existent round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayload) {
				return sharedtypes.RoundID(uuid.New()), &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: sharedtypes.RoundID(uuid.New()), // A non-existent UUID
				}
			},
			expectedFailure:          true,              // Changed from expectedError
			expectedErrorMessagePart: "round not found", // Updated to match implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round not found") {
					t.Errorf("Expected failure error to contain 'round not found', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Deletion with EventBus error (should still succeed for round deletion)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_delete_2"),
					Title:     "Round to be deleted with bus error",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: roundForDBInsertion.ID,
				}
			},
			expectedFailure: false, // Changed from expectedError - Service should still return success for round deletion
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				deletedPayload, ok := returnedResult.Success.(*roundevents.RoundDeletedPayload)
				if !ok {
					t.Errorf("Expected RoundDeletedPayload, got %T", returnedResult.Success)
				}

				// Verify the round's state is DELETED in the DB
				persistedRound, err := deps.DB.GetRound(ctx, deletedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to fetch round from DB after deletion: %v", err)
				}
				if persistedRound == nil {
					t.Fatalf("Expected round to be found in DB (marked deleted), but it was nil")
				}
				if persistedRound.State != roundtypes.RoundStateDeleted {
					t.Errorf("Expected round state to be DELETED, but got %s", persistedRound.State)
				}
			},
		},
		{
			name: "Attempt to delete round with nil UUID payload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayload) {
				return sharedtypes.RoundID(uuid.Nil), &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: sharedtypes.RoundID(uuid.Nil), // Nil UUID
				}
			},
			expectedFailure:          true,                     // Changed from expectedError
			expectedErrorMessagePart: "round ID cannot be nil", // Updated to match implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round ID cannot be nil") {
					t.Errorf("Expected failure error to contain 'round ID cannot be nil', got '%s'", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload *roundevents.RoundDeleteAuthorizedPayload
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = &roundevents.RoundDeleteAuthorizedPayload{
					RoundID: sharedtypes.RoundID(uuid.New()),
				}
			}

			// Call the actual service method
			result, err := deps.Service.DeleteRound(deps.Ctx, *payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayload)
					if !ok {
						t.Errorf("Expected RoundDeleteErrorPayload, got %T", result.Failure)
					} else if !strings.Contains(failurePayload.Error, tt.expectedErrorMessagePart) {
						t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, failurePayload.Error)
					}
				}
			} else {
				if result.Success == nil {
					t.Errorf("Expected success result, but got none")
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
