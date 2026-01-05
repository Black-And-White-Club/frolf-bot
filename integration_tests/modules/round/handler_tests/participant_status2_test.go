package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleParticipantJoinValidationRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response for Scheduled Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				payload := createPayload(data.RoundID, data.UserID, roundtypes.ResponseAccept)
				publishAndExpectTagLookup(t, deps, deps.MessageCapture, payload, false)
			},
		},
		{
			name: "Success - Accept Response for InProgress Round (Late Join)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				data = data.WithRoundID(roundID)
				payload := createPayload(data.RoundID, data.UserID, roundtypes.ResponseAccept)
				publishAndExpectTagLookup(t, deps, deps.MessageCapture, payload, true)
			},
		},
		{
			name: "Success - Decline Response for Scheduled Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				payload := createPayload(data.RoundID, data.UserID, roundtypes.ResponseDecline)
				publishAndExpectStatusUpdate(t, deps, deps.MessageCapture, payload, false)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData().WithRoundID(sharedtypes.RoundID(uuid.New())) // Non-existent round
				payload := createPayload(data.RoundID, data.UserID, roundtypes.ResponseAccept)
				publishAndExpectError(t, deps, deps.MessageCapture, payload, data.RoundID, data.UserID)
			},
		},
		{
			name: "Failure - Empty User ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				payload := createPayload(data.RoundID, "", roundtypes.ResponseAccept)  // Empty user ID
				publishAndExpectError(t, deps, deps.MessageCapture, payload, data.RoundID, "")
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup - no need to clear messages between tests!
	// Each test uses unique IDs so messages won't interfere
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear message capture before each subtest
			deps.MessageCapture.Clear()
			// Create helper for each subtest
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test - no cleanup needed!
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Test-specific helper functions - only used in this file

func createPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinValidationRequestPayloadV1 {
	return roundevents.ParticipantJoinValidationRequestPayloadV1{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
		GuildID:  "test-guild",
	}
}

func publishParticipantJoinValidationRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantJoinValidationRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantJoinValidationRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForTagLookupRequest(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.LeaderboardGetTagNumberRequestedV1, count, defaultTimeout)
}

func getTagLookupRequestMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.LeaderboardGetTagNumberRequestedV1)
}

func waitForParticipantStatusUpdateRequest(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantStatusUpdateRequestedV1, count, defaultTimeout)
}

func waitForParticipantJoinError(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, count, defaultTimeout)
}

func getParticipantStatusUpdateRequestMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantStatusUpdateRequestedV1)
}

func getParticipantJoinErrorMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

func validateTagLookupRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.TagLookupRequestPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.TagLookupRequestPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse tag lookup request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return result
}

func validateParticipantStatusUpdateRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.ParticipantJoinRequestPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinRequestPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant status update request message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Response != expectedResponse {
		t.Errorf("Response mismatch: expected %s, got %s", expectedResponse, result.Response)
	}

	return result
}

func validateParticipantJoinError(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant join error message: %v", err)
	}

	if result.ParticipantJoinRequest == nil {
		t.Error("Expected ParticipantJoinRequest to be populated in error payload")
		return
	}

	if result.ParticipantJoinRequest.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.ParticipantJoinRequest.RoundID)
	}

	if result.ParticipantJoinRequest.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.ParticipantJoinRequest.UserID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

func publishAndExpectTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayloadV1, expectedLateJoin bool) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getTagLookupRequestMessages(capture)
		// Find the message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.TagLookupRequestPayloadV1](msg)
			if err == nil && parsed.RoundID == payload.RoundID {
				foundMsg = msg
				break
			}
		}
		if foundMsg != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if foundMsg == nil {
		// Debug: Show what messages we DID receive
		allTagMsgs := getTagLookupRequestMessages(capture)
		allErrorMsgs := getParticipantJoinErrorMessages(capture)
		allStatusMsgs := getParticipantStatusUpdateRequestMessages(capture)
		t.Logf("DEBUG: Looking for round %s, found %d tag lookup messages, %d error messages, %d status messages",
			payload.RoundID, len(allTagMsgs), len(allErrorMsgs), len(allStatusMsgs))

		for i, msg := range allTagMsgs {
			if parsed, err := testutils.ParsePayload[roundevents.TagLookupRequestPayloadV1](msg); err == nil {
				t.Logf("DEBUG: Tag message %d has round ID: %s", i, parsed.RoundID)
			}
		}
		for i, msg := range allErrorMsgs {
			if parsed, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg); err == nil {
				t.Logf("DEBUG: Error message %d: %s", i, parsed.Error)
			}
		}
		t.Fatalf("Expected tag lookup request message for round %s", payload.RoundID)
	}

	result := validateTagLookupRequest(t, foundMsg, payload.RoundID, payload.UserID, payload.Response)

	if result.JoinedLate == nil || *result.JoinedLate != expectedLateJoin {
		t.Errorf("Expected JoinedLate=%t, got %v", expectedLateJoin, result.JoinedLate)
	}
}

func publishAndExpectStatusUpdate(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayloadV1, expectedLateJoin bool) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantStatusUpdateRequestMessages(capture)
		// Find the message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.ParticipantJoinRequestPayloadV1](msg)
			if err == nil && parsed.RoundID == payload.RoundID {
				foundMsg = msg
				break
			}
		}
		if foundMsg != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if foundMsg == nil {
		t.Fatalf("Expected participant status update request message for round %s", payload.RoundID)
	}

	result := validateParticipantStatusUpdateRequest(t, foundMsg, payload.RoundID, payload.UserID, payload.Response)

	if result.JoinedLate == nil || *result.JoinedLate != expectedLateJoin {
		t.Errorf("Expected JoinedLate=%t, got %v", expectedLateJoin, result.JoinedLate)
	}
}

func publishAndExpectError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayloadV1, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	// Wait up to 1 second for the specific error message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantJoinErrorMessages(capture)
		// Find the error message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg)
			if err == nil && parsed.ParticipantJoinRequest != nil && parsed.ParticipantJoinRequest.RoundID == expectedRoundID {
				foundMsg = msg
				break
			}
		}
		if foundMsg != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	if foundMsg == nil {
		t.Fatalf("Expected participant join error message for round %s", expectedRoundID)
	}

	validateParticipantJoinError(t, foundMsg, expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	// Count messages before publishing invalid JSON
	tagMsgsBefore := len(getTagLookupRequestMessages(capture))
	statusMsgsBefore := len(getParticipantStatusUpdateRequestMessages(capture))

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantJoinValidationRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Count messages after - should be the same (no new messages)
	tagMsgsAfter := len(getTagLookupRequestMessages(capture))
	statusMsgsAfter := len(getParticipantStatusUpdateRequestMessages(capture))

	newTagMsgs := tagMsgsAfter - tagMsgsBefore
	newStatusMsgs := statusMsgsAfter - statusMsgsBefore

	if newTagMsgs > 0 || newStatusMsgs > 0 {
		t.Errorf("Expected no NEW success messages for invalid JSON, got %d new tag msgs and %d new status msgs",
			newTagMsgs, newStatusMsgs)
	}
}
