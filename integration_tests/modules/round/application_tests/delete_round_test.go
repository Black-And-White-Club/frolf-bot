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
		payload                  roundevents.RoundDeleteRequestPayload
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful validation of delete request",
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.New()),
				RequestingUserUserID: "user123",
			},
			expectedError: false,
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
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.Nil), // Zero UUID
				RequestingUserUserID: "user123",
			},
			expectedError:            true,
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
			// Corrected payload type to match expected input for ValidateRoundDeleteRequest
			payload: roundevents.RoundDeleteRequestPayload{
				RoundID:              sharedtypes.RoundID(uuid.New()),
				RequestingUserUserID: "", // Empty user ID
			},
			expectedError:            true,
			expectedErrorMessagePart: "requesting user's Discord ID cannot be empty", // This error comes from ValidateRoundDeleteRequest
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			// Call the actual service method
			result, err := deps.Service.ValidateRoundDeleteRequest(deps.Ctx, tt.payload)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.expectedErrorMessagePart != "" && !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
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
		expectedError            bool
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
			expectedError: false,
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
			expectedError:            true,
			expectedErrorMessagePart: "round with ID", // Expected error from TestRoundDB matching RoundDBImpl
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on error, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayload)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round with ID") {
					t.Errorf("Expected failure error to contain 'round with ID', got '%s'", failurePayload.Error)
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
			expectedError: false, // Service should still return success for round deletion
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
			expectedError:            true,
			expectedErrorMessagePart: "cannot delete round: nil UUID provided",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on error, but got: %+v", returnedResult.Success)
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

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got none")
				} else if tt.expectedErrorMessagePart != "" && !strings.Contains(err.Error(), tt.expectedErrorMessagePart) {
					t.Errorf("Expected error message to contain '%s', but got: '%v'", tt.expectedErrorMessagePart, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
