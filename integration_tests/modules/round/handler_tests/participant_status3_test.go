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
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with Tag Number",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(42)
				payload := createStatusUpdatePayload(roundID, userID, roundtypes.ResponseAccept, &tagNumber, boolPtr(false))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Decline Response",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createStatusUpdatePayload(roundID, userID, roundtypes.ResponseDecline, nil, nil)
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Tentative Response with Late Join",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateInProgress)
				tagNumber := sharedtypes.TagNumber(99)
				payload := createStatusUpdatePayload(roundID, userID, roundtypes.ResponseTentative, &tagNumber, boolPtr(true))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Accept Response without Tag Number",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createStatusUpdatePayload(roundID, userID, roundtypes.ResponseAccept, nil, boolPtr(false))
				publishAndExpectParticipantJoined(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createStatusUpdatePayload(nonExistentRoundID, userID, roundtypes.ResponseDecline, nil, nil)
				publishAndExpectStatusUpdateError(t, deps, deps.MessageCapture, payload, nonExistentRoundID, userID)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoStatusUpdateMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			helper.ClearMessages()
			tc.setupAndRun(t, helper, &deps)

			time.Sleep(1 * time.Second)
		})
	}
}

// Test-specific helper functions for HandleParticipantStatusUpdateRequest

func createStatusUpdatePayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response, tagNumber *sharedtypes.TagNumber, joinedLate *bool) roundevents.ParticipantJoinRequestPayload {
	return roundevents.ParticipantJoinRequestPayload{
		RoundID:    roundID,
		UserID:     userID,
		Response:   response,
		TagNumber:  tagNumber,
		JoinedLate: joinedLate,
	}
}

func publishParticipantStatusUpdateRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantJoinRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantStatusUpdateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForParticipantJoined(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoined, count, defaultTimeout)
}

func waitForParticipantJoinErrorFromStatusUpdate(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinError, count, defaultTimeout)
}

func getParticipantJoinedMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoined)
}

func getParticipantJoinErrorFromStatusUpdateMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinError)
}

func validateParticipantJoinedMessage(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayload](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayload](msg)
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

func publishAndExpectParticipantJoined(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinRequestPayload) {
	publishParticipantStatusUpdateRequest(t, deps, &payload)

	if !waitForParticipantJoined(capture, 1) {
		t.Fatalf("Expected participant joined message")
	}

	msgs := getParticipantJoinedMessages(capture)
	validateParticipantJoinedMessage(t, msgs[0], payload.RoundID)
}

func publishAndExpectStatusUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinRequestPayload, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantStatusUpdateRequest(t, deps, &payload)

	if !waitForParticipantJoinErrorFromStatusUpdate(capture, 1) {
		t.Fatalf("Expected participant join error message")
	}

	msgs := getParticipantJoinErrorFromStatusUpdateMessages(capture)
	validateParticipantJoinErrorFromStatusUpdate(t, msgs[0], expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoStatusUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantStatusUpdateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	joinedMsgs := getParticipantJoinedMessages(capture)
	errorMsgs := getParticipantJoinErrorFromStatusUpdateMessages(capture)

	if len(joinedMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages for invalid JSON, got %d joined msgs and %d error msgs",
			len(joinedMsgs), len(errorMsgs))
	}
}
