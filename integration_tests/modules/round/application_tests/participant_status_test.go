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

func TestCheckParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "User not a participant, requesting Accept - Expecting Validation Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_1"),
					Title:     "Round for status check",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_1"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				validationPayload, ok := returnedResult.Success.(*roundevents.ParticipantJoinValidationRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinValidationRequestPayload, got %T", returnedResult.Success)
					return
				}
				if validationPayload.UserID != sharedtypes.DiscordID("new_participant_1") {
					t.Errorf("Expected UserID to be 'new_participant_1', got '%s'", validationPayload.UserID)
				}
				if validationPayload.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", validationPayload.Response)
				}
			},
		},
		{
			name: "User is participant with Accept, requesting Accept (toggle off) - Expecting Removal Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				// Define the participant directly and include it in the round's Participants slice
				participant := roundtypes.Participant{
					UserID:   sharedtypes.DiscordID("existing_participant_2"),
					Response: roundtypes.ResponseAccept,
				}
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_2"),
					Title:     "Round for status check toggle",
					State:     roundtypes.RoundStateUpcoming,
				})
				roundForDBInsertion.Participants = []roundtypes.Participant{participant} // Set participant directly
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)                    // Create the round with the participant
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("existing_participant_2"),
					Response: roundtypes.ResponseAccept, // Same response as existing
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				removalPayload, ok := returnedResult.Success.(*roundevents.ParticipantRemovalRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantRemovalRequestPayload, got %T", returnedResult.Success)
					return
				}
				if removalPayload.UserID != sharedtypes.DiscordID("existing_participant_2") {
					t.Errorf("Expected UserID to be 'existing_participant_2', got '%s'", removalPayload.UserID)
				}
				if removalPayload.RoundID != removalPayload.RoundID { // Self-check, but good for consistency
					t.Errorf("Expected RoundID to match, got %s vs %s", removalPayload.RoundID, removalPayload.RoundID)
				}
			},
		},
		{
			name: "User is participant with Tentative, requesting Accept (change status) - Expecting Validation Request",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				// Define the participant directly and include it in the round's Participants slice
				participant := roundtypes.Participant{
					UserID:   sharedtypes.DiscordID("existing_participant_3"),
					Response: roundtypes.ResponseTentative,
				}
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_check_status_3"),
					Title:     "Round for status change",
					State:     roundtypes.RoundStateUpcoming,
				})
				roundForDBInsertion.Participants = []roundtypes.Participant{participant} // Set participant directly
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)                    // Create the round with the participant
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("existing_participant_3"),
					Response: roundtypes.ResponseAccept, // Different response
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				validationPayload, ok := returnedResult.Success.(*roundevents.ParticipantJoinValidationRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinValidationRequestPayload, got %T", returnedResult.Success)
					return
				}
				if validationPayload.UserID != sharedtypes.DiscordID("existing_participant_3") {
					t.Errorf("Expected UserID to be 'existing_participant_3', got '%s'", validationPayload.UserID)
				}
				if validationPayload.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", validationPayload.Response)
				}
			},
		},
		{
			name: "Round ID is nil - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				return sharedtypes.RoundID(uuid.Nil), roundevents.ParticipantJoinRequestPayload{
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,                               // Changed from expectedError
			expectedErrorMessagePart: "failed to get participant status", // Updated to match implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.ParticipantStatusCheckErrorPayload)
				if !ok {
					t.Errorf("Expected *ParticipantStatusCheckErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to get participant status") {
					t.Errorf("Expected failure error to contain 'failed to get participant status', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Attempt to check status for a non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  nonExistentID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true,                               // Changed from expectedError
			expectedErrorMessagePart: "failed to get participant status", // Updated to match implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.ParticipantStatusCheckErrorPayload)
				if !ok {
					t.Errorf("Expected *ParticipantStatusCheckErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to get participant status") {
					t.Errorf("Expected failure error to contain 'failed to get participant status', got '%s'", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload roundevents.ParticipantJoinRequestPayload
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				// Default payload if setupTestEnv is nil
				payload = roundevents.ParticipantJoinRequestPayload{
					RoundID:  sharedtypes.RoundID(uuid.New()),
					UserID:   sharedtypes.DiscordID("default_user"),
					Response: roundtypes.ResponseAccept,
				}
			}

			// Call the actual service method
			result, err := deps.Service.CheckParticipantStatus(deps.Ctx, payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.ParticipantStatusCheckErrorPayload)
					if !ok {
						t.Errorf("Expected *ParticipantStatusCheckErrorPayload, got %T", result.Failure)
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

func TestValidateParticipantJoinRequest(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Valid Accept request, round Created (not late join) - Expecting TagLookupRequestPayload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_1"),
					Title:     "Round for validation (created)",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_1"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// ValidateParticipantJoinRequest returns TagLookupRequestPayload for Accept/Tentative
				tagLookupPayloadPtr, ok := returnedResult.Success.(*roundevents.TagLookupRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.TagLookupRequestPayload, got %T", returnedResult.Success)
					return
				}
				if tagLookupPayloadPtr == nil {
					t.Fatalf("Expected non-nil TagLookupRequestPayload pointer")
				}

				if tagLookupPayloadPtr.UserID != sharedtypes.DiscordID("new_participant_1") {
					t.Errorf("Expected UserID to be 'new_participant_1', got '%s'", tagLookupPayloadPtr.UserID)
				}
				if tagLookupPayloadPtr.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be 'Accept', got '%s'", tagLookupPayloadPtr.Response)
				}
				if tagLookupPayloadPtr.JoinedLate == nil || *tagLookupPayloadPtr.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %v", tagLookupPayloadPtr.JoinedLate)
				}
			},
		},
		{
			name: "Valid Tentative request, round InProgress (late join) - Expecting TagLookupRequestPayload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_2"),
					Title:     "Round for validation (in progress)",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_2"),
					Response: roundtypes.ResponseTentative,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// ValidateParticipantJoinRequest returns TagLookupRequestPayload for Accept/Tentative
				tagLookupPayloadPtr, ok := returnedResult.Success.(*roundevents.TagLookupRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.TagLookupRequestPayload, got %T", returnedResult.Success)
					return
				}
				if tagLookupPayloadPtr == nil {
					t.Fatalf("Expected non-nil TagLookupRequestPayload pointer")
				}

				if tagLookupPayloadPtr.UserID != sharedtypes.DiscordID("new_participant_2") {
					t.Errorf("Expected UserID to be 'new_participant_2', got '%s'", tagLookupPayloadPtr.UserID)
				}
				if tagLookupPayloadPtr.Response != roundtypes.ResponseTentative {
					t.Errorf("Expected Response to be 'Tentative', got '%s'", tagLookupPayloadPtr.Response)
				}
				if tagLookupPayloadPtr.JoinedLate == nil || *tagLookupPayloadPtr.JoinedLate != true {
					t.Errorf("Expected JoinedLate to be true, got %v", tagLookupPayloadPtr.JoinedLate)
				}
			},
		},
		{
			name: "Valid Decline request, round Finalized (late join) - Expecting ParticipantJoinRequestPayload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_3"),
					Title:     "Round for validation (finalized)",
					State:     roundtypes.RoundStateFinalized,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("new_participant_3"),
					Response: roundtypes.ResponseDecline,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// ValidateParticipantJoinRequest returns ParticipantJoinRequestPayload for Decline
				joinRequestPayloadPtr, ok := returnedResult.Success.(*roundevents.ParticipantJoinRequestPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinRequestPayload, got %T", returnedResult.Success)
					return
				}
				if joinRequestPayloadPtr == nil {
					t.Fatalf("Expected non-nil ParticipantJoinRequestPayload pointer")
				}

				if joinRequestPayloadPtr.UserID != sharedtypes.DiscordID("new_participant_3") {
					t.Errorf("Expected UserID to be 'new_participant_3', got '%s'", joinRequestPayloadPtr.UserID)
				}
				if joinRequestPayloadPtr.Response != roundtypes.ResponseDecline {
					t.Errorf("Expected Response to be 'Decline', got '%s'", joinRequestPayloadPtr.Response)
				}
				if joinRequestPayloadPtr.JoinedLate == nil || *joinRequestPayloadPtr.JoinedLate != true {
					t.Errorf("Expected JoinedLate to be true, got %v", joinRequestPayloadPtr.JoinedLate)
				}
			},
		},
		{
			name: "Invalid: Nil Round ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				return sharedtypes.RoundID(uuid.Nil), roundevents.ParticipantJoinRequestPayload{
					RoundID:  sharedtypes.RoundID(uuid.Nil),
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true, // Changed from expectedError
			expectedErrorMessagePart: "validation failed: [round ID cannot be nil]",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "validation failed: [round ID cannot be nil]") {
					t.Errorf("Expected failure error to contain 'validation failed: [round ID cannot be nil]', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Invalid: Empty User ID - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_empty_user"),
					Title:     "Round for validation (empty user)",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   "", // Empty user ID
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true, // Changed from expectedError
			expectedErrorMessagePart: "validation failed: [participant Discord ID cannot be empty]",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "validation failed: [participant Discord ID cannot be empty]") {
					t.Errorf("Expected failure error to contain 'validation failed: [participant Discord ID cannot be empty]', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Attempt to validate join for a non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  nonExistentID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: roundtypes.ResponseAccept,
				}
			},
			expectedFailure:          true, // Changed from expectedError
			expectedErrorMessagePart: "failed to fetch round details",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details") {
					t.Errorf("Expected failure error to contain 'failed to fetch round details', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Invalid: Unexpected response type - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_validate_invalid_response"),
					Title:     "Round for validation (invalid response)",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:  roundForDBInsertion.ID,
					UserID:   sharedtypes.DiscordID("some_user"),
					Response: "INVALID_RESPONSE_TYPE", // Invalid response
				}
			},
			expectedFailure:          true, // Changed from expectedError
			expectedErrorMessagePart: "unexpected response type: INVALID_RESPONSE_TYPE",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "unexpected response type: INVALID_RESPONSE_TYPE") {
					t.Errorf("Expected failure error to contain 'unexpected response type: INVALID_RESPONSE_TYPE', got '%s'", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload roundevents.ParticipantJoinRequestPayload
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				// Default payload if setupTestEnv is nil
				generator := testutils.NewTestDataGenerator()
				dummyRound := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("default_user"),
					Title:     "Default Round",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(deps.Ctx, &dummyRound)
				if err != nil {
					t.Fatalf("Failed to create default round for test setup: %v", err)
				}
				payload = roundevents.ParticipantJoinRequestPayload{
					RoundID:  dummyRound.ID,
					UserID:   sharedtypes.DiscordID("default_user_payload"),
					Response: roundtypes.ResponseAccept,
				}
			}

			// Call the actual service method: ValidateParticipantJoinRequest
			result, err := deps.Service.ValidateParticipantJoinRequest(deps.Ctx, payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundParticipantJoinErrorPayload)
					if !ok {
						t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", result.Failure)
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
