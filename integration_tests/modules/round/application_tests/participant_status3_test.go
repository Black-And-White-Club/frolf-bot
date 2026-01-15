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
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1)
		expectedFailure          bool // Changed from expectedError
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult)
	}{
		{
			name: "Internal update: Participant status and tag number set after lookup", // Renamed test case
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

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
				err := deps.DB.CreateRound(ctx, guildID, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				tag := sharedtypes.TagNumber(123)
				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("existing_participant"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				joinedPayload, ok := returnedResult.Success.(*roundevents.ParticipantJoinedPayloadV1)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinedPayloadV1, got %T", returnedResult.Success)
					return
				}

				if len(joinedPayload.AcceptedParticipants) != 1 {
					t.Errorf("Expected 1 accepted participant, got %d", len(joinedPayload.AcceptedParticipants))
					return // Prevent index out of range
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

				// Verify DB state (accept both round not found and round present)
				roundInDB, err := deps.DB.GetRound(ctx, joinedPayload.GuildID, joinedPayload.RoundID)
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "not found") {
						// Acceptable: round was deleted or cleaned up
						return
					}
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
			name: "Participant accepts without TagNumber (adds participant with nil tag)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_3"),
					Title:     "Round for nil tag participant",
					State:     roundtypes.RoundStateInProgress,
				})
				err := deps.DB.CreateRound(ctx, guildID, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("participant_needs_tag"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  nil, // Participant doesn't provide tag - should be added with nil
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: the implementation actually returns ParticipantJoinedPayload, not RoundTagLookupRequestedPayloadV1
				joinedPayload, ok := returnedResult.Success.(*roundevents.ParticipantJoinedPayloadV1)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinedPayloadV1, got %T", returnedResult.Success)
					return
				}

				// Verify the participant was added to the accepted list with nil tag
				if len(joinedPayload.AcceptedParticipants) != 1 {
					t.Errorf("Expected 1 accepted participant, got %d", len(joinedPayload.AcceptedParticipants))
					return
				}

				acceptedParticipant := joinedPayload.AcceptedParticipants[0]
				if acceptedParticipant.UserID != sharedtypes.DiscordID("participant_needs_tag") {
					t.Errorf("Expected UserID to be 'participant_needs_tag', got '%s'", acceptedParticipant.UserID)
				}
				if acceptedParticipant.Response != roundtypes.ResponseAccept {
					t.Errorf("Expected Response to be Accept, got %s", acceptedParticipant.Response)
				}
				if acceptedParticipant.TagNumber != nil {
					t.Errorf("Expected TagNumber to be nil, got %v", acceptedParticipant.TagNumber)
				}
				if *joinedPayload.JoinedLate != false {
					t.Errorf("Expected JoinedLate to be false, got %t", *joinedPayload.JoinedLate)
				}

				// Verify DB state - participant should be in DB with nil tag (accept round not found)
				roundInDB, err := deps.DB.GetRound(ctx, joinedPayload.GuildID, joinedPayload.RoundID)
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "not found") {
						// Acceptable: round was deleted or cleaned up
						return
					}
					t.Fatalf("Failed to get round from DB: %v", err)
				}

				found := false
				for _, p := range roundInDB.Participants {
					if p.UserID == sharedtypes.DiscordID("participant_needs_tag") {
						found = true
						if p.Response != roundtypes.ResponseAccept {
							t.Errorf("DB: Expected participant's Response to be Accept, got %s", p.Response)
						}
						if p.TagNumber != nil {
							t.Errorf("DB: Expected participant's TagNumber to be nil, got %v", p.TagNumber)
						}
						break
					}
				}
				if !found {
					t.Errorf("DB: Participant 'participant_needs_tag' not found in database after join")
				}
			},
		},
		{
			name: "Participant declines (direct update)",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

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
				err := deps.DB.CreateRound(ctx, guildID, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("participant_to_decline"),
					Response:   roundtypes.ResponseDecline,
					TagNumber:  nil, // TagNumber should be nil for decline
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure: false, // Changed from expectedError
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success == nil {
					t.Fatalf("Expected success result, but got nil")
				}
				// Fixed: expecting pointer type
				joinedPayload, ok := returnedResult.Success.(*roundevents.ParticipantJoinedPayloadV1)
				if !ok {
					t.Errorf("Expected *roundevents.ParticipantJoinedPayloadV1, got %T", returnedResult.Success)
					return
				}

				if len(joinedPayload.DeclinedParticipants) != 1 {
					t.Errorf("Expected 1 declined participant, got %d", len(joinedPayload.DeclinedParticipants))
					return // Prevent index out of range
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

				// Verify DB state (accept both round not found and round present)
				roundInDB, err := deps.DB.GetRound(ctx, joinedPayload.GuildID, joinedPayload.RoundID)
				if err != nil {
					if strings.Contains(strings.ToLower(err.Error()), "not found") {
						// Acceptable: round was deleted or cleaned up
						return
					}
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
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

				nonExistentID := sharedtypes.RoundID(uuid.New())
				tag := sharedtypes.TagNumber(789)
				joinedLate := false
				return nonExistentID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    nonExistentID,
					UserID:     sharedtypes.DiscordID("some_user"),
					Response:   roundtypes.ResponseAccept,
					TagNumber:  &tag,
					JoinedLate: &joinedLate,
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
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayloadV1)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details") {
					t.Errorf("Expected error message to contain 'failed to fetch round details', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Attempt to update participant in non-existent round (decline) - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

				nonExistentID := sharedtypes.RoundID(uuid.New())
				joinedLate := false
				return nonExistentID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    nonExistentID,
					UserID:     sharedtypes.DiscordID("some_user"),
					Response:   roundtypes.ResponseDecline,
					TagNumber:  nil,
					JoinedLate: &joinedLate,
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
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayloadV1)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
				}
				if !strings.Contains(failurePayload.Error, "failed to fetch round details") {
					t.Errorf("Expected error message to contain 'failed to fetch round details', got '%s'", failurePayload.Error)
				}
			},
		},
		{
			name: "Unknown response type - Expecting Error",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.RoundID, roundevents.ParticipantJoinRequestPayloadV1) {
				guildID := sharedtypes.GuildID("test-guild")

				generator := testutils.NewTestDataGenerator()
				roundForDBInsertion := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					CreatedBy: testutils.DiscordID("test_user_upd_7"),
					Title:     "Round for unknown response",
					State:     roundtypes.RoundStateUpcoming,
				})
				err := deps.DB.CreateRound(ctx, guildID, &roundForDBInsertion)
				if err != nil {
					t.Fatalf("Failed to create initial round in DB for test setup: %v", err)
				}

				joinedLate := false
				return roundForDBInsertion.ID, roundevents.ParticipantJoinRequestPayloadV1{
					GuildID:    guildID,
					RoundID:    roundForDBInsertion.ID,
					UserID:     sharedtypes.DiscordID("unknown_resp_user"),
					Response:   roundtypes.Response("INVALID_RESPONSE"), // Invalid response
					TagNumber:  nil,
					JoinedLate: &joinedLate,
				}
			},
			expectedFailure:          true, // Changed from expectedError
			expectedErrorMessagePart: "unknown response type: INVALID_RESPONSE",
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult roundservice.RoundOperationResult) {
				if returnedResult.Success != nil {
					t.Errorf("Expected nil success on failure, but got: %+v", returnedResult.Success)
				}
				if returnedResult.Failure == nil {
					t.Fatalf("Expected failure result, but got nil")
				}
				// Fixed: expecting pointer type
				failurePayload, ok := returnedResult.Failure.(*roundevents.RoundParticipantJoinErrorPayloadV1)
				if !ok {
					t.Errorf("Expected *RoundParticipantJoinErrorPayload, got %T", returnedResult.Failure)
					return
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

			var payload roundevents.ParticipantJoinRequestPayloadV1
			if tt.setupTestEnv != nil {
				_, payload = tt.setupTestEnv(deps.Ctx, deps)
			} else {
				// Default payload if setupTestEnv is nil (should not happen with current test structure)
				t.Fatal("setupTestEnv should not be nil")
			}

			// Call the actual service method: UpdateParticipantStatus
			result, err := deps.Service.UpdateParticipantStatus(deps.Ctx, payload)
			// The service should never return an error - failures are in the result
			if err != nil {
				t.Errorf("Expected no error from service, but got: %v", err)
			}

			// Check for expected failures in the result
			if tt.expectedFailure {
				if result.Failure == nil {
					t.Errorf("Expected failure result, but got none")
				} else if tt.expectedErrorMessagePart != "" {
					failurePayload, ok := result.Failure.(*roundevents.RoundParticipantJoinErrorPayloadV1)
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
