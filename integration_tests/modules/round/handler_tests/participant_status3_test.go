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

func TestHandleParticipantStatusUpdateRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with Tag Number",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				tagNumber := sharedtypes.TagNumber(42)
				payload := createStatusUpdatePayload(data.RoundID, data.UserID, roundtypes.ResponseAccept, &tagNumber, boolPtr(false))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Decline Response",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				payload := createStatusUpdatePayload(data.RoundID, data.UserID, roundtypes.ResponseDecline, nil, nil)
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Tentative Response with Late Join",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				data = data.WithRoundID(roundID)
				tagNumber := sharedtypes.TagNumber(99)
				payload := createStatusUpdatePayload(data.RoundID, data.UserID, roundtypes.ResponseTentative, &tagNumber, boolPtr(true))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Accept Response without Tag Number",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				data = data.WithRoundID(roundID)
				payload := createStatusUpdatePayload(data.RoundID, data.UserID, roundtypes.ResponseAccept, nil, boolPtr(false))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createStatusUpdatePayload(nonExistentRoundID, data.UserID, roundtypes.ResponseDecline, nil, nil)
				publishAndExpectStatusUpdateError(t, deps, deps.MessageCapture, payload, nonExistentRoundID, data.UserID)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoStatusUpdateMessages(t, deps, deps.MessageCapture)
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

// Test-specific helper functions for HandleParticipantStatusUpdateRequest

func createStatusUpdatePayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response, tagNumber *sharedtypes.TagNumber, joinedLate *bool) roundevents.ParticipantJoinRequestPayloadV1 {
	return roundevents.ParticipantJoinRequestPayloadV1{
		RoundID:    roundID,
		UserID:     userID,
		Response:   response,
		TagNumber:  tagNumber,
		JoinedLate: joinedLate,
		GuildID:    "test-guild",
	}
}

func publishParticipantStatusUpdateRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantJoinRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantStatusUpdateRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForParticipantJoined(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinedV1, count, defaultTimeout)
}

func waitForParticipantJoinErrorFromStatusUpdate(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, count, defaultTimeout)
}

func getParticipantJoinedMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinedV1)
}

func getParticipantJoinErrorFromStatusUpdateMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

func validateParticipantJoinedMessage(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant joined message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

func validateParticipantJoinErrorFromStatusUpdate(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
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

func publishAndExpectParticipantJoined(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinRequestPayloadV1) {
	publishParticipantStatusUpdateRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantJoinedMessages(capture)
		// Find the message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayloadV1](msg)
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
		t.Fatalf("Expected participant joined message for round %s", payload.RoundID)
	}

	validateParticipantJoinedMessage(t, foundMsg, payload.RoundID)
}

func publishAndExpectStatusUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinRequestPayloadV1, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantStatusUpdateRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantJoinErrorFromStatusUpdateMessages(capture)
		// Find the message matching THIS test's round ID
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

	validateParticipantJoinErrorFromStatusUpdate(t, foundMsg, expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoStatusUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	// Count messages BEFORE
	joinedMsgsBefore := len(getParticipantJoinedMessages(capture))
	errorMsgsBefore := len(getParticipantJoinErrorFromStatusUpdateMessages(capture))

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantStatusUpdateRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Count messages AFTER
	joinedMsgsAfter := len(getParticipantJoinedMessages(capture))
	errorMsgsAfter := len(getParticipantJoinErrorFromStatusUpdateMessages(capture))

	newJoinedMsgs := joinedMsgsAfter - joinedMsgsBefore
	newErrorMsgs := errorMsgsAfter - errorMsgsBefore

	if newJoinedMsgs > 0 || newErrorMsgs > 0 {
		t.Errorf("Expected no NEW messages for invalid JSON, got %d new joined msgs and %d new error msgs",
			newJoinedMsgs, newErrorMsgs)
	}
}
