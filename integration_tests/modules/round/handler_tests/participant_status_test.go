package roundhandler_integration_tests

import (
	"context"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidParticipantJoinRequestPayload creates a valid ParticipantJoinRequestPayload for testing
func createValidParticipantJoinRequestPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinRequestPayload {
	return roundevents.ParticipantJoinRequestPayload{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
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
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)
	anotherUser := generator.GenerateUsers(1)[0]
	anotherUserID := sharedtypes.DiscordID(anotherUser.UserID)

	testCases := []struct {
		name                    string
		setupAndRun             func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
		expectRemovalRequest    bool
		expectValidationRequest bool
		expectStatusCheckError  bool
	}{
		{
			name:                    "Success - New Join Request (Accept)",
			expectValidationRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                    "Success - New Join Request (Tentative)",
			expectValidationRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseTentative)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                    "Success - New Join Request (Decline)",
			expectValidationRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                 "Success - Toggle Removal (Accept to Accept)",
			expectRemovalRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                 "Success - Toggle Removal (Decline to Decline)",
			expectRemovalRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                    "Success - Status Change (Accept to Decline)",
			expectValidationRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                    "Success - Status Change (Decline to Accept)",
			expectValidationRequest: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                   "Failure - Non-Existent Round ID",
			expectStatusCheckError: true,
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createValidParticipantJoinRequestPayload(nonExistentRoundID, userID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			helper.ClearMessages()

			// Log initial state
			t.Logf("Starting test case: %s", tc.name)

			// Run the test scenario
			tc.setupAndRun(t, helper, &deps)

			// MUCH longer wait for message processing and capture
			t.Logf("Waiting for message processing and capture...")
			time.Sleep(5 * time.Second) // Increased from 1 second

			// Check what messages were captured
			validationMsgs := helper.GetParticipantJoinValidationRequestMessages()
			removalMsgs := helper.GetParticipantRemovalRequestMessages()
			errorMsgs := helper.GetParticipantStatusCheckErrorMessages()

			// Log what we found
			t.Logf("Messages captured: validation=%d, removal=%d, error=%d",
				len(validationMsgs), len(removalMsgs), len(errorMsgs))

			// Check ALL captured messages for debugging
			if allMsgs := helper.GetAllCapturedMessages(); len(allMsgs) > 0 {
				for topic, msgs := range allMsgs {
					if len(msgs) > 0 {
						t.Logf("Captured topic %s: %d messages", topic, len(msgs))
						for i, msg := range msgs {
							t.Logf("  Message %d ID: %s", i, msg.UUID)
						}
					}
				}
			} else {
				t.Logf("NO messages captured at all - check message capture setup")
			}

			if tc.expectValidationRequest {
				if len(validationMsgs) == 0 {
					t.Errorf("Expected participant validation request message, got none. Handler published message UUID f37e7f14-6ed6-46ab-b421-3372d3752472 but capture didn't receive it.")
				}
				if len(removalMsgs) > 0 {
					t.Errorf("Expected no removal messages, got %d", len(removalMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectRemovalRequest {
				if len(removalMsgs) == 0 {
					t.Error("Expected participant removal request message, got none")
				}
				if len(validationMsgs) > 0 {
					t.Errorf("Expected no validation messages, got %d", len(validationMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectStatusCheckError {
				if len(errorMsgs) == 0 {
					t.Error("Expected participant status check error message, got none")
				}
				if len(validationMsgs) > 0 {
					t.Errorf("Expected no validation messages, got %d", len(validationMsgs))
				}
				if len(removalMsgs) > 0 {
					t.Errorf("Expected no removal messages, got %d", len(removalMsgs))
				}
			}
		})
	}
}
