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

func TestParticipantRemoval(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Valid removal of existing participant - Expecting ParticipantRemovedPayload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_1"),
					Title:     "Round for participant removal",
					State:     roundtypes.RoundStateUpcoming,
				})

				// Add some participants to the round
				scoreZero := sharedtypes.Score(0)
				tagNum1 := sharedtypes.TagNumber(1)
				tagNum2 := sharedtypes.TagNumber(2)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("participant_to_remove"),
						TagNumber: &tagNum1,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
					{
						UserID:    sharedtypes.DiscordID("participant_staying_1"),
						TagNumber: &tagNum2,
						Response:  roundtypes.ResponseTentative,
						Score:     &scoreZero,
					},
					{
						UserID:    sharedtypes.DiscordID("participant_staying_2"),
						TagNumber: nil, // This participant doesn't have a tag number yet
						Response:  roundtypes.ResponseDecline,
						Score:     &scoreZero,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, roundevents.ParticipantRemovalRequestPayload{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("participant_to_remove"),
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				// Fixed: expecting pointer type
				removedPayloadPtr, ok := returnedResult.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantRemovedPayload, got %T", returnedResult.Success)
					return
				}

				if removedPayloadPtr.UserID != sharedtypes.DiscordID("participant_to_remove") {
					t.Errorf("Expected UserID to be 'participant_to_remove', got '%s'", removedPayloadPtr.UserID)
				}

				// Check that the remaining participants are categorized correctly
				if len(removedPayloadPtr.AcceptedParticipants) != 0 {
					t.Errorf("Expected 0 accepted participants after removal, got %d", len(removedPayloadPtr.AcceptedParticipants))
				}
				if len(removedPayloadPtr.TentativeParticipants) != 1 {
					t.Errorf("Expected 1 tentative participant after removal, got %d", len(removedPayloadPtr.TentativeParticipants))
				}
				if len(removedPayloadPtr.DeclinedParticipants) != 1 {
					t.Errorf("Expected 1 declined participant after removal, got %d", len(removedPayloadPtr.DeclinedParticipants))
				}

				// Verify the remaining participant details
				if len(removedPayloadPtr.TentativeParticipants) > 0 && removedPayloadPtr.TentativeParticipants[0].UserID != sharedtypes.DiscordID("participant_staying_1") {
					t.Errorf("Expected tentative participant to be 'participant_staying_1', got '%s'", removedPayloadPtr.TentativeParticipants[0].UserID)
				}
				if len(removedPayloadPtr.DeclinedParticipants) > 0 && removedPayloadPtr.DeclinedParticipants[0].UserID != sharedtypes.DiscordID("participant_staying_2") {
					t.Errorf("Expected declined participant to be 'participant_staying_2', got '%s'", removedPayloadPtr.DeclinedParticipants[0].UserID)
				}
			},
		},
		{
			name: "Attempt to remove non-existent participant - Should succeed with no changes",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_2"),
					Title:     "Round for non-existent participant removal",
					State:     roundtypes.RoundStateInProgress,
				})

				// Add one participant
				scoreZero := sharedtypes.Score(0)
				tagNum10 := sharedtypes.TagNumber(10)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("existing_participant"),
						TagNumber: &tagNum10,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, roundevents.ParticipantRemovalRequestPayload{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("non_existent_participant"),
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				// Fixed: expecting pointer type
				removedPayloadPtr, ok := returnedResult.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantRemovedPayload, got %T", returnedResult.Success)
					return
				}

				if removedPayloadPtr.UserID != sharedtypes.DiscordID("non_existent_participant") {
					t.Errorf("Expected UserID to be 'non_existent_participant', got '%s'", removedPayloadPtr.UserID)
				}

				// Since this is testing the "user not found" case, the lists should contain the existing participant
				if len(removedPayloadPtr.AcceptedParticipants) != 1 {
					t.Errorf("Expected 1 accepted participant (existing one), got %d", len(removedPayloadPtr.AcceptedParticipants))
				}
				if len(removedPayloadPtr.TentativeParticipants) != 0 {
					t.Errorf("Expected 0 tentative participants, got %d", len(removedPayloadPtr.TentativeParticipants))
				}
				if len(removedPayloadPtr.DeclinedParticipants) != 0 {
					t.Errorf("Expected 0 declined participants, got %d", len(removedPayloadPtr.DeclinedParticipants))
				}
			},
		},
		{
			name: "Remove participant from round with multiple participants of same response type",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_3"),
					Title:     "Round with multiple accepted participants",
					State:     roundtypes.RoundStateFinalized,
				})

				// Add multiple participants with same response type
				scoreZero := sharedtypes.Score(0)
				tagNum5 := sharedtypes.TagNumber(5)
				tagNum15 := sharedtypes.TagNumber(15)
				tagNum25 := sharedtypes.TagNumber(25)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("accepted_participant_1"),
						TagNumber: &tagNum5,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
					{
						UserID:    sharedtypes.DiscordID("accepted_participant_2"),
						TagNumber: &tagNum15,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
					{
						UserID:    sharedtypes.DiscordID("accepted_participant_to_remove"),
						TagNumber: &tagNum25,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, roundevents.ParticipantRemovalRequestPayload{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("accepted_participant_to_remove"),
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				// Fixed: expecting pointer type
				removedPayloadPtr, ok := returnedResult.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantRemovedPayload, got %T", returnedResult.Success)
					return
				}

				if removedPayloadPtr.UserID != sharedtypes.DiscordID("accepted_participant_to_remove") {
					t.Errorf("Expected UserID to be 'accepted_participant_to_remove', got '%s'", removedPayloadPtr.UserID)
				}

				// Should have 2 remaining accepted participants
				if len(removedPayloadPtr.AcceptedParticipants) != 2 {
					t.Errorf("Expected 2 accepted participants after removal, got %d", len(removedPayloadPtr.AcceptedParticipants))
				}
				if len(removedPayloadPtr.TentativeParticipants) != 0 {
					t.Errorf("Expected 0 tentative participants, got %d", len(removedPayloadPtr.TentativeParticipants))
				}
				if len(removedPayloadPtr.DeclinedParticipants) != 0 {
					t.Errorf("Expected 0 declined participants, got %d", len(removedPayloadPtr.DeclinedParticipants))
				}

				// Verify the removed participant is not in the remaining list
				for _, p := range removedPayloadPtr.AcceptedParticipants {
					if p.UserID == sharedtypes.DiscordID("accepted_participant_to_remove") {
						t.Errorf("Removed participant should not appear in remaining accepted participants")
					}
				}
			},
		},
		{
			name: "Attempt to remove participant from non-existent round - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				return nonExistentID, roundevents.ParticipantRemovalRequestPayload{
					RoundID: nonExistentID,
					UserID:  sharedtypes.DiscordID("some_user"),
				}
			},
			expectedFailure:          true,                            // Changed from expectedError
			expectedErrorMessagePart: "failed to fetch round details", // Updated to match implementation
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}

				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.ParticipantRemovalErrorPayload)
				if !ok {
					t.Errorf("Expected *ParticipantRemovalErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details") {
					t.Errorf("Expected failure error to contain 'failed to fetch round details', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Remove last participant from round - Should result in empty lists",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantRemovalRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_removal_4"),
					Title:     "Round with single participant",
					State:     roundtypes.RoundStateUpcoming,
				})

				// Add only one participant
				scoreZero := sharedtypes.Score(0)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("only_participant"),
						TagNumber: nil, // No tag number assigned yet
						Response:  roundtypes.ResponseTentative,
						Score:     &scoreZero,
					},
				}

				err := deps.DB.CreateRound(ctx, "test-guild", &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				return roundForDBInsertion.ID, roundevents.ParticipantRemovalRequestPayload{
					RoundID: roundForDBInsertion.ID,
					UserID:  sharedtypes.DiscordID("only_participant"),
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}

				// Fixed: expecting pointer type
				removedPayloadPtr, ok := returnedResult.Success.(*roundevents.ParticipantRemovedPayload)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantRemovedPayload, got %T", returnedResult.Success)
					return
				}

				if removedPayloadPtr.UserID != sharedtypes.DiscordID("only_participant") {
					t.Errorf("Expected UserID to be 'only_participant', got '%s'", removedPayloadPtr.UserID)
				}

				// All lists should be empty after removing the only participant
				if len(removedPayloadPtr.AcceptedParticipants) != 0 {
					t.Errorf("Expected 0 accepted participants after removing last participant, got %d", len(removedPayloadPtr.AcceptedParticipants))
				}
				if len(removedPayloadPtr.TentativeParticipants) != 0 {
					t.Errorf("Expected 0 tentative participants after removing last participant, got %d", len(removedPayloadPtr.TentativeParticipants))
				}
				if len(removedPayloadPtr.DeclinedParticipants) != 0 {
					t.Errorf("Expected 0 declined participants after removing last participant, got %d", len(removedPayloadPtr.DeclinedParticipants))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)

			var payload roundevents.ParticipantRemovalRequestPayload
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
				err := deps.DB.CreateRound(deps.Ctx, "test-guild", &dummyRound)
				if err != nil {
					t.Fatalf("Failed to create default round for test setup: %v", err)
				}
				payload = roundevents.ParticipantRemovalRequestPayload{
					RoundID: dummyRound.ID,
					UserID:  sharedtypes.DiscordID("default_user_payload"),
				}
			}

			// Call the actual service method: ParticipantRemoval
			result, err := deps.Service.ParticipantRemoval(deps.Ctx, payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.ParticipantRemovalErrorPayload)
					if !ok {
						t.Errorf("Expected *ParticipantRemovalErrorPayload, got %T", result.Failure)
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
