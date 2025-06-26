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
		name                        string
		setupAndRun                 func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
		expectJoinValidationRequest bool // Expect join validation request for Accept/Tentative
		expectRemovalRequest        bool // Expect removal request for toggle scenarios
		expectStatusUpdateRequest   bool // Expect status update request for Decline responses
		expectStatusCheckError      bool
	}{
		{
			name:                        "Success - New Join Request (Accept)",
			expectJoinValidationRequest: true, // Expect join validation request for Accept
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                        "Success - New Join Request (Tentative)",
			expectJoinValidationRequest: true, // Expect join validation request for Tentative
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseTentative)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                        "Success - New Join Request (Decline)",
			expectJoinValidationRequest: true, // All new joins go through validation first
			expectStatusUpdateRequest:   true, // Decline responses also trigger status update
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundForTesting(t, userID, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, anotherUserID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                 "Success - Toggle Removal (Accept to Accept)",
			expectRemovalRequest: true, // Expect removal request for toggle
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                 "Success - Toggle Removal (Decline to Decline)",
			expectRemovalRequest: true, // Expect removal request for toggle
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                        "Success - Status Change (Accept to Decline)",
			expectJoinValidationRequest: true, // Status change from Accept to Decline produces validation request
			expectStatusUpdateRequest:   true, // Decline responses also trigger status update
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseAccept, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseDecline)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                        "Success - Status Change (Decline to Accept)",
			expectJoinValidationRequest: true, // Status change from Decline to Accept produces validation request
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := createExistingRoundWithParticipant(t, userID, roundtypes.ResponseDecline, deps.DB)
				payload := createValidParticipantJoinRequestPayload(roundID, userID, roundtypes.ResponseAccept)
				helper.PublishParticipantJoinRequest(t, context.Background(), payload)
			},
		},
		{
			name:                   "Failure - Non-Existent Round ID",
			expectStatusCheckError: true, // Service still returns error for non-existent rounds
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

			// Clear messages at the start and end of each test for isolation
			helper.ClearMessages()
			t.Cleanup(func() {
				helper.ClearMessages()
			})

			// Run the test scenario
			tc.setupAndRun(t, helper, &deps)

			// Wait for message processing with longer timeout for chained events
			time.Sleep(2 * time.Second)

			// Get messages for verification
			joinValidationMsgs := helper.GetParticipantJoinValidationRequestMessages()
			removalRequestMsgs := helper.GetParticipantRemovalRequestMessages()
			statusUpdateMsgs := helper.GetParticipantStatusUpdateRequestMessages()
			errorMsgs := helper.GetParticipantStatusCheckErrorMessages()

			t.Logf("Message counts - JoinValidation: %d, RemovalRequest: %d, StatusUpdate: %d, Error: %d",
				len(joinValidationMsgs), len(removalRequestMsgs), len(statusUpdateMsgs), len(errorMsgs))

			// Check what messages were produced by the handler
			// HandleParticipantJoinRequest currently produces:
			// - RoundParticipantJoinValidationRequest for Accept/Tentative new joins and status changes
			// - RoundParticipantRemovalRequest for toggle removals
			// - RoundParticipantStatusUpdateRequest for Decline responses (via validation handler)
			// - RoundParticipantStatusCheckError for failures

			if tc.expectJoinValidationRequest && tc.expectStatusUpdateRequest {
				// Both join validation and status update are expected (e.g., Decline responses)
				if len(joinValidationMsgs) == 0 {
					t.Error("Expected participant join validation request message, got none")
				}
				if len(statusUpdateMsgs) == 0 {
					t.Error("Expected participant status update request message, got none")
				}
				if len(removalRequestMsgs) > 0 {
					t.Errorf("Expected no removal request messages, got %d", len(removalRequestMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectJoinValidationRequest {
				if len(joinValidationMsgs) == 0 {
					t.Error("Expected participant join validation request message, got none")
				}
				if len(removalRequestMsgs) > 0 {
					t.Errorf("Expected no removal request messages, got %d", len(removalRequestMsgs))
				}
				if len(statusUpdateMsgs) > 0 {
					t.Errorf("Expected no status update messages, got %d", len(statusUpdateMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectRemovalRequest {
				if len(removalRequestMsgs) == 0 {
					t.Error("Expected participant removal request message, got none")
				}
				if len(joinValidationMsgs) > 0 {
					t.Errorf("Expected no join validation messages, got %d", len(joinValidationMsgs))
				}
				if len(statusUpdateMsgs) > 0 {
					t.Errorf("Expected no status update messages, got %d", len(statusUpdateMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectStatusUpdateRequest {
				if len(statusUpdateMsgs) == 0 {
					t.Error("Expected participant status update request message, got none")
				}
				if len(joinValidationMsgs) > 0 {
					t.Errorf("Expected no join validation messages, got %d", len(joinValidationMsgs))
				}
				if len(removalRequestMsgs) > 0 {
					t.Errorf("Expected no removal request messages, got %d", len(removalRequestMsgs))
				}
				if len(errorMsgs) > 0 {
					t.Errorf("Expected no error messages, got %d", len(errorMsgs))
				}
			} else if tc.expectStatusCheckError {
				if len(errorMsgs) == 0 {
					t.Error("Expected participant status check error message, got none")
				}
				if len(joinValidationMsgs) > 0 {
					t.Errorf("Expected no join validation messages, got %d", len(joinValidationMsgs))
				}
				if len(removalRequestMsgs) > 0 {
					t.Errorf("Expected no removal request messages, got %d", len(removalRequestMsgs))
				}
				if len(statusUpdateMsgs) > 0 {
					t.Errorf("Expected no status update messages, got %d", len(statusUpdateMsgs))
				}
			}
		})
	}
}
