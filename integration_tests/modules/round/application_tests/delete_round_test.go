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
		setupRound               func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful validation of delete request",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1 {
				// Create a round in the database first
				generator := testutils.NewTestDataGenerator()
				userID := "user123"
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID(userID),
					Title:     "Test Round for Validation",
					State:     roundtypes.RoundStateUpcoming,
				})
				guildID := sharedtypes.GuildID("test-guild")
				err := deps.DB.CreateRound(deps.Ctx, guildID, &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              guildID,
					RoundID:              roundForDB.ID,
					RequestingUserUserID: roundForDB.CreatedBy,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				_, ok := returnedResult.Success.(*roundevents.RoundDeleteValidatedPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteValidatedPayload, got %T", returnedResult.Success)
				}
			},
		},
		{
			name: "Validation fails with zero RoundID",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1 {
				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              sharedtypes.GuildID("test-guild"),
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
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round ID cannot be zero") {
					t.Errorf("Expected error to contain 'round ID cannot be zero', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails with empty RequestingUserUserID",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1 {
				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              sharedtypes.GuildID("test-guild"),
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
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "requesting user's Discord ID cannot be empty") {
					t.Errorf("Expected error to contain 'requesting user's Discord ID cannot be empty', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails when round does not exist",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1 {
				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              sharedtypes.GuildID("test-guild"),
					RoundID:              sharedtypes.RoundID(uuid.New()), // Non-existent round
					RequestingUserUserID: "user123",
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "round not found",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round not found") {
					t.Errorf("Expected error to contain 'round not found', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Validation fails for non-creator (authorization check is in ValidateRoundDeleteRequest)",
			setupRound: func(deps RoundTestDeps) roundevents.RoundDeleteRequestPayloadV1 {
				// Create a round with one user
				generator := testutils.NewTestDataGenerator()
				creatorUserID := "creator123"
				roundForDB := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID(creatorUserID),
					Title:     "Test Round for Ownership Test",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(deps.Ctx, sharedtypes.GuildID("test-guild"), &roundForDB)
				if err != nil {
					t.Fatalf("Failed to create round for test: %v", err)
				}

				return roundevents.RoundDeleteRequestPayloadV1{
					GuildID:              sharedtypes.GuildID("test-guild"),
					RoundID:              roundForDB.ID,
					RequestingUserUserID: "different_user123", // Different user trying to delete
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "unauthorized: only the round creator can delete the round",
			validateResult: func(t *testing.T, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "unauthorized: only the round creator can delete the round") {
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
					failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
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
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayloadV1)
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Successful deletion of an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayloadV1) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_delete_1"),
					Title:     "Round to be deleted",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, sharedtypes.GuildID("test-guild"), &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, &roundevents.RoundDeleteAuthorizedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundForDBInsertion.ID,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				deletedPayload, ok := returnedResult.Success.(*roundevents.RoundDeletedPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeletedPayload, got %T", returnedResult.Success)
				}

				// Verify the round was actually deleted (accept both soft and hard delete)
				round, err := deps.DB.GetRound(ctx, sharedtypes.GuildID("test-guild"), deletedPayload.RoundID)
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "not found") {
						// Hard delete: round is gone, this is acceptable
						return
					}
					t.Fatalf("Unexpected error getting round after deletion: %v", err)
				}
				t.Logf("DEBUG: Round after deletion - ID: %s, State: %s", round.ID, round.State)
				if round.State != roundtypes.RoundStateDeleted {
					t.Errorf("Expected round state to be DELETED, but got %s", round.State)
				}
			},
		},
		{
			name: "Attempt to delete a non-existent round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayloadV1) {
				return sharedtypes.RoundID(uuid.New()), &roundevents.RoundDeleteAuthorizedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: sharedtypes.RoundID(uuid.New()), // A non-existent UUID
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "round not found",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
				if !ok {
					t.Errorf("Expected RoundDeleteErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "round not found") {
					t.Errorf("Expected failure error to contain 'round not found', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Attempt to delete round with nil UUID payload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, *roundevents.RoundDeleteAuthorizedPayloadV1) {
				return sharedtypes.RoundID(uuid.Nil), &roundevents.RoundDeleteAuthorizedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: sharedtypes.RoundID(uuid.Nil), // Nil UUID
				}
			},
			expectedFailure:          true,
			expectedErrorMessagePart: "round ID cannot be nil",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
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

			var payload *roundevents.RoundDeleteAuthorizedPayloadV1
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				payload = &roundevents.RoundDeleteAuthorizedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
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
					failurePayload, ok := result.Failure.(*roundevents.RoundDeleteErrorPayloadV1)
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
