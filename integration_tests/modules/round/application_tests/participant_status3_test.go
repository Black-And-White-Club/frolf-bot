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

func TestUpdateParticipantStatus(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload)
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Internal update: Participant status and tag number set after lookup", // Renamed test case
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_1"),
					Title:     "Round for direct update",
					State:     roundtypes.RoundStateUpcoming,
				})
				// Add a participant initially tentative
				scoreZero := sharedtypes.Score(0)
				tagNum1 := sharedtypes.TagNumber(1)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("existing_participant"),
						TagNumber: &tagNum1,
						Response:  roundtypes.ResponseTentative,
						Score:     &scoreZero,
					},
				}
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				tag := sharedtypes.TagNumber(123)
				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("existing_participant"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				joinedPayload, ok := returnedResult.Success.(roundevents.ParticipantJoinedPayload)
				if !ok {
					t.Errorf("Expected roundevents.ParticipantJoinedPayload, got %T", returnedResult.Success)
				}

				if len(joinedPayload.AcceptedParticipants) != 1 {
					t.Errorf("Expected 1 accepted participant, got %d", len(joinedPayload.AcceptedParticipants))
				}
				if *joinedPayload.AcceptedParticipants[0].TagNumber != 123 {
					t.Errorf("Expected accepted participant's TagNumber to be 123, got %d", *joinedPayload.AcceptedParticipants[0].TagNumber)
				}
				if joinedPayload.AcceptedParticipants[0].Response != roundtypes.ResponseAccept {
					t.Errorf("Expected accepted participant's Response to be Accept, got %s", joinedPayload.AcceptedParticipants[0].Response)
				}
				if *joinedPayload.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %t", *joinedPayload.JoinedLate)
				}

				// Verify DB state
				roundInDB, err := deps.DB.GetRound(ctx, joinedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to get round from DB: %v", err)
				}
				found := false
				for _, p := range roundInDB.Participants {
					if p.UserID == sharedtypes.DiscordID("existing_participant") {
						found = true
						if *p.TagNumber != 123 {
							t.Errorf("DB: Expected participant's TagNumber to be 123, got %d", *p.TagNumber)
						}
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("DB: Expected participant's Response to be Accept, got %s", p.Response)
						}
						break
					}
				}
				if !found {
					t.Errorf("DB: Participant 'existing_participant' not found after update")
				}
			},
		},

		{
			name: "Participant accepts without TagNumber (triggers TagLookupRequestPayload)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_3"),
					Title:     "Round for tag lookup trigger",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("participant_needs_tag"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  nil, // Crucial for this test case: user does not provide tag
					JoinedLate: &joinedLate,
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				tagLookupPayload, ok := returnedResult.Success.(roundevents.TagLookupRequestPayload)
				if !ok {
					t.Errorf("Expected roundevents.TagLookupRequestPayload, got %T", returnedResult.Success)
				}

				if tagLookupPayload.UserID != sharedtypes.DiscordID("participant_needs_tag") {
					t.Errorf("Expected UserID to be 'participant_needs_tag', got '%s'", tagLookupPayload.UserID)
				}
				if tagLookupPayload.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be Accept, got %s", tagLookupPayload.Response)
				}
				if *tagLookupPayload.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %t", *tagLookupPayload.JoinedLate)
				}

				// Verify DB state - participant should NOT be in DB yet, or should be in old state if existing
				roundInDB, err := deps.DB.GetRound(ctx, tagLookupPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to get round from DB: %v", err)
				}
				for _, p := range roundInDB.Participants {
					if p.UserID == sharedtypes.DiscordID("participant_needs_tag") {
						t.Errorf("DB: Participant 'participant_needs_tag' should not have been added/updated directly in DB yet")
					}
				}
			},
		},
		{
			name: "Participant declines (direct update)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_4"),
					Title:     "Round for decline",
					State:     roundtypes.RoundStateUpcoming,
				})
				// Add a participant initially accepted
				scoreZero := sharedtypes.Score(0)
				tagNum2 := sharedtypes.TagNumber(2)
				roundForDBInsertion.Participants = []roundtypes.Participant{
					{
						UserID:    sharedtypes.DiscordID("participant_to_decline"),
						TagNumber: &tagNum2,
						Response:  roundtypes.ResponseAccept,
						Score:     &scoreZero,
					},
				}
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					// In a real test suite, `t.Fatalf` would be called here.
					// For this structure, we assume the outer test runner handles fatal errors from setup.
					panic("Failed to create initial round in DB for test setup: " + err.Error())
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("participant_to_decline"),
					Response:   roundtypes.ResponseDecline,
					TagNumber:  nil, // TagNumber should be nil for decline
					JoinedLate: &joinedLate,
				}
			},
			expectedError: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				joinedPayload, ok := returnedResult.Success.(roundevents.ParticipantJoinedPayload)
				if !ok {
					t.Errorf("Expected roundevents.ParticipantJoinedPayload, got %T", returnedResult.Success)
				}

				if len(joinedPayload.DeclinedParticipants) != 1 {
					t.Errorf("Expected 1 declined participant, got %d", len(joinedPayload.DeclinedParticipants))
				}
				if joinedPayload.DeclinedParticipants[0].UserID != sharedtypes.DiscordID("participant_to_decline") {
					t.Errorf("Expected declined participant to be 'participant_to_decline', got '%s'", joinedPayload.DeclinedParticipants[0].UserID)
				}
				if joinedPayload.DeclinedParticipants[0].Response != roundtypes.ResponseDecline {
					t.Errorf("Expected declined participant's Response to be Decline, got %s", joinedPayload.DeclinedParticipants[0].Response)
				}
				if joinedPayload.DeclinedParticipants[0].TagNumber != nil {
					t.Errorf("Expected declined participant's TagNumber to be nil, got %v", joinedPayload.DeclinedParticipants[0].TagNumber)
				}
				if *joinedPayload.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %t", *joinedPayload.JoinedLate)
				}

				// Verify DB state
				roundInDB, err := deps.DB.GetRound(ctx, joinedPayload.RoundID)
				if err != nil {
					t.Fatalf("Failed to get round from DB: %v", err)
				}
				found := false
				for _, p := range roundInDB.Participants {
					if p.UserID == sharedtypes.DiscordID("participant_to_decline") {
						found = true
						if p.Response != roundtypes.ResponseDecline {
							t.Errorf("DB: Expected participant's Response to be Decline, got %s", p.Response)
						}
						if p.TagNumber != nil {
							t.Errorf("DB: Expected participant's TagNumber to be nil, got %v", p.TagNumber)
						}
						break
					}
				}
				if !found {
					t.Errorf("DB: Participant 'participant_to_decline' not found after update")
				}
			},
		},
		{
			name: "Attempt to update participant in non-existent round (with tag) - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				tag := sharedtypes.TagNumber(789)
				joinedLate := false
				return nonExistentID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    nonExistentID,
					UserID:     sharedtypes.DiscordID("some_user"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
				}
			},
			expectedError:            true,
			expectedErrorMessagePart: "failed to fetch round details for tag update",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on error, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(roundevents.ParticipantUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected ParticipantUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details for tag update") {
					t.Errorf("Expected error message to contain 'failed to fetch round details for tag update', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Attempt to update participant in non-existent round (decline) - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				nonExistentID := sharedtypes.RoundID(uuid.New())
				joinedLate := false
				return nonExistentID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    nonExistentID,
					UserID:     sharedtypes.DiscordID("some_user"),
					Response:   roundtypes.ResponseDecline,
					TagNumber:  nil,
					JoinedLate: &joinedLate,
				}
			},
			expectedError:            true,
			expectedErrorMessagePart: "failed to fetch round details for decline update",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on error, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(roundevents.ParticipantUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected ParticipantUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details for decline update") {
					t.Errorf("Expected error message to contain 'failed to fetch round details for decline update', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Unknown response type - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayload) {
				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_7"),
					Title:     "Round for unknown response",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayload{
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("unknown_resp_user"),
					Response:   roundtypes.Response("INVALID_RESPONSE"), // Invalid response
					TagNumber:  nil,
					JoinedLate: &joinedLate,
				}
			},
			expectedError:            true,
			expectedErrorMessagePart: "unknown response type: INVALID_RESPONSE",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on error, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				failurePayload, ok := returnedResult.Failure.(roundevents.ParticipantUpdateErrorPayload)
				if !ok {
					t.Errorf("Expected ParticipantUpdateErrorPayload, got %T", returnedResult.Failure)
				}
				if !strings.Contains(failurePayload.Error, "unknown response type: INVALID_RESPONSE") {
					t.Errorf("Expected error message to contain 'unknown response type: INVALID_RESPONSE', got '%s'", failurePayload.Error)
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
				// Default payload if setupTestEnv is nil (should not happen with current test structure)
				t.Fatal("setupTestEnv should not be nil")
			}

			// Call the actual service method: UpdateParticipantStatus
			result, err := deps.Service.UpdateParticipantStatus(deps.Ctx, payload)

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
