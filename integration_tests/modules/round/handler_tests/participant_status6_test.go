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

func TestHandleParticipantDeclined(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Participant Declined (Upcoming Round)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createParticipantDeclinedPayload(roundID, userID)
				publishAndExpectParticipantDeclinedFromHandler(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Participant Declined (In Progress Round)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateInProgress)
				payload := createParticipantDeclinedPayload(roundID, userID)
				publishAndExpectParticipantDeclinedFromHandler(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found (Participant Declined)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createParticipantDeclinedPayload(nonExistentRoundID, userID)
				publishAndExpectDeclineErrorFromHandler(t, deps, deps.MessageCapture, payload, nonExistentRoundID, userID)
			},
		},
		{
			name: "Invalid JSON - Participant Declined Handler",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoDeclineMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO PARTICIPANT DECLINED TESTS
func createParticipantDeclinedPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.ParticipantDeclinedPayload {
	return roundevents.ParticipantDeclinedPayload{
		RoundID: roundID,
		UserID:  userID,
	}
}

// Publishing functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func publishParticipantDeclinedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantDeclinedPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantDeclined, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// Wait functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func waitForParticipantDeclinedFromHandler(capture *testutils.MessageCapture, count int) bool {
	// Success cases (including declines) should go to RoundParticipantJoined
	return capture.WaitForMessages(roundevents.RoundParticipantJoined, count, defaultTimeout)
}

func waitForParticipantDeclineErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func getParticipantDeclinedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	// Success cases (including declines) should go to RoundParticipantJoined
	return capture.GetMessages(roundevents.RoundParticipantJoined)
}

func getParticipantDeclineErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinError)
}

// Validation functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func validateParticipantDeclinedFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant declined message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	return result
}

func validateParticipantDeclineErrorFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant decline error message: %v", err)
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

	if result.ParticipantJoinRequest.Response != roundtypes.ResponseDecline {
		t.Errorf("Response mismatch: expected %s, got %s", roundtypes.ResponseDecline, result.ParticipantJoinRequest.Response)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Test expectation functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func publishAndExpectParticipantDeclinedFromHandler(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantDeclinedPayload) {
	publishParticipantDeclinedMessage(t, deps, &payload)

	if !waitForParticipantDeclinedFromHandler(capture, 1) {
		t.Fatalf("Expected participant declined success message from %s", roundevents.RoundParticipantJoined)
	}

	msgs := getParticipantDeclinedFromHandlerMessages(capture)
	validateParticipantDeclinedFromHandler(t, msgs[0], payload.RoundID)
}

func publishAndExpectDeclineErrorFromHandler(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantDeclinedPayload, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantDeclinedMessage(t, deps, &payload)

	if !waitForParticipantDeclineErrorFromHandler(capture, 1) {
		t.Fatalf("Expected participant decline error message from %s", roundevents.RoundParticipantJoinError)
	}

	msgs := getParticipantDeclineErrorFromHandlerMessages(capture)
	validateParticipantDeclineErrorFromHandler(t, msgs[0], expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoDeclineMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantDeclined, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	declinedMsgs := getParticipantDeclinedFromHandlerMessages(capture)
	errorMsgs := getParticipantDeclineErrorFromHandlerMessages(capture)

	if len(declinedMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages for invalid JSON on %s, got %d success, %d error msgs",
			roundevents.RoundParticipantDeclined, len(declinedMsgs), len(errorMsgs))
	}
}
