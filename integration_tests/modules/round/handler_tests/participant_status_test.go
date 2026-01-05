package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidParticipantJoinRequestPayload creates a valid ParticipantJoinRequestPayload for testing
func createValidParticipantJoinRequestPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinRequestPayloadV1 {
	return roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
		GuildID:  "test-guild",
		// TagNumber and JoinedLate will be determined by the service
	}
}

// createExistingRoundWithParticipant creates a round with an existing participant for testing toggles
func createExistingRoundWithParticipant(t *testing.T, userID sharedtypes.DiscordID, existingResponse roundtypes.Response, db bun.IDB) sharedtypes.RoundID {
	t.Helper()
	helper := testutils.NewRoundTestHelper(nil, nil) // Don't need event bus
	return helper.CreateRoundWithParticipantInDB(t, db, userID, existingResponse)
}

// Helper function to create a basic round for testing
func createExistingRoundForTesting(t *testing.T, userID sharedtypes.DiscordID, db bun.IDB) sharedtypes.RoundID {
	t.Helper()
	helper := testutils.NewRoundTestHelper(nil, nil) // Don't need event bus
	return helper.CreateRoundInDB(t, db, userID)
}

// TestHandleParticipantJoinRequest tests the participant join request handler integration
func TestHandleParticipantJoinRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - New Join Request (Accept)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForTesting(t, data.UserID, deps.DB)
				// Use a different user for the join request
				joinerID := sharedtypes.DiscordID(uuid.New().String())
				payload := createValidParticipantJoinRequestPayload(roundID, joinerID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, roundID, true, false, false, false)
			},
		},
		{
			name: "Success - New Join Request (Tentative)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForTesting(t, data.UserID, deps.DB)
				joinerID := sharedtypes.DiscordID(uuid.New().String())
				payload := createValidParticipantJoinRequestPayload(roundID, joinerID, roundtypes.ResponseTentative)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, roundID, true, false, false, false)
			},
		},
		{
			name: "Success - New Join Request (Decline)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundForTesting(t, data.UserID, deps.DB)
				joinerID := sharedtypes.DiscordID(uuid.New().String())
				payload := createValidParticipantJoinRequestPayload(roundID, joinerID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				// Expect join validation AND status update
				validateJoinRequestMessages(t, helper, roundID, true, false, true, false)
			},
		},
		{
			name: "Success - Toggle Removal (Accept to Accept)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundWithParticipant(t, data.UserID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, data.UserID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, roundID, false, true, false, false)
			},
		},
		{
			name: "Success - Toggle Removal (Decline to Decline)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundWithParticipant(t, data.UserID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, data.UserID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, roundID, false, true, false, false)
			},
		},
		{
			name: "Success - Status Change (Accept to Decline)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundWithParticipant(t, data.UserID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, data.UserID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				// Status change produces validation request, and Decline triggers status update
				validateJoinRequestMessages(t, helper, roundID, true, false, true, false)
			},
		},
		{
			name: "Success - Status Change (Decline to Accept)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := createExistingRoundWithParticipant(t, data.UserID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, data.UserID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, roundID, true, false, false, false)
			},
		},
		{
			name: "Failure - Non-Existent Round ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidParticipantJoinRequestPayload(nonExistentRoundID, data.UserID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)

				validateJoinRequestMessages(t, helper, nonExistentRoundID, false, false, false, true)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear message capture before each subtest
			deps.MessageCapture.Clear()
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

func validateJoinRequestMessages(t *testing.T, helper *testutils.RoundTestHelper, roundID sharedtypes.RoundID, expectJoinValidation, expectRemoval, expectStatusUpdate, expectError bool) {
	t.Helper()
	deadline := time.Now().Add(1 * time.Second)

	var foundJoinValidation *message.Message
	var foundRemoval *message.Message
	var foundStatusUpdate *message.Message
	var foundError *message.Message

	for time.Now().Before(deadline) {
		if expectJoinValidation && foundJoinValidation == nil {
			msgs := helper.GetParticipantJoinValidationRequestMessages()
			for _, msg := range msgs {
				parsed, err := testutils.ParsePayload[roundevents.ParticipantJoinValidationRequestPayloadV1](msg)
				if err == nil && parsed.RoundID == roundID {
					foundJoinValidation = msg
					break
				}
			}
		}

		if expectRemoval && foundRemoval == nil {
			msgs := helper.GetParticipantRemovalRequestMessages()
			for _, msg := range msgs {
				parsed, err := testutils.ParsePayload[roundevents.ParticipantRemovalRequestPayloadV1](msg)
				if err == nil && parsed.RoundID == roundID {
					foundRemoval = msg
					break
				}
			}
		}

		if expectStatusUpdate && foundStatusUpdate == nil {
			msgs := helper.GetParticipantStatusUpdateRequestMessages()
			for _, msg := range msgs {
				parsed, err := testutils.ParsePayload[roundevents.ParticipantJoinRequestPayloadV1](msg)
				if err == nil && parsed.RoundID == roundID {
					foundStatusUpdate = msg
					break
				}
			}
		}

		if expectError && foundError == nil {
			msgs := helper.GetParticipantStatusCheckErrorMessages()
			for _, msg := range msgs {
				parsed, err := testutils.ParsePayload[roundevents.ParticipantStatusCheckErrorPayloadV1](msg)
				if err == nil && parsed.RoundID == roundID {
					foundError = msg
					break
				}
			}
		}

		if (!expectJoinValidation || foundJoinValidation != nil) &&
			(!expectRemoval || foundRemoval != nil) &&
			(!expectStatusUpdate || foundStatusUpdate != nil) &&
			(!expectError || foundError != nil) {
			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	if expectJoinValidation && foundJoinValidation == nil {
		t.Error("Expected participant join validation request message, got none")
	}
	if expectRemoval && foundRemoval == nil {
		t.Error("Expected participant removal request message, got none")
	}
	if expectStatusUpdate && foundStatusUpdate == nil {
		t.Error("Expected participant status update request message, got none")
	}
	if expectError && foundError == nil {
		t.Error("Expected participant status check error message, got none")
	}
}
