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

func TestHandleParticipantRemovalRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Remove Existing Accepted Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, userID, roundtypes.ResponseAccept)
				payload := createRemovalPayload(roundID, userID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Remove Existing Tentative Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, userID, roundtypes.ResponseTentative)
				payload := createRemovalPayload(roundID, userID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Remove Existing Declined Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, userID, roundtypes.ResponseDecline)
				payload := createRemovalPayload(roundID, userID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createRemovalPayload(nonExistentRoundID, userID)
				publishAndExpectRemovalError(t, deps, deps.MessageCapture, payload, nonExistentRoundID, userID)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoRemovalMessages(t, deps, deps.MessageCapture)
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

// Test-specific helper functions for HandleParticipantRemovalRequest

func createRemovalPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.ParticipantRemovalRequestPayload {
	return roundevents.ParticipantRemovalRequestPayload{
		RoundID: roundID,
		UserID:  userID,
	}
}

func publishParticipantRemovalRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantRemovalRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantRemovalRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForParticipantRemoved(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantRemoved, count, defaultTimeout)
}

func waitForParticipantRemovalError(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantRemovalError, count, defaultTimeout)
}

func getParticipantRemovedMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantRemoved)
}

func getParticipantRemovalErrorMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantRemovalError)
}

func validateParticipantRemovedMessage(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) *roundevents.ParticipantRemovedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantRemovedPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant removed message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	return result
}

func validateParticipantRemovalError(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantRemovalErrorPayload](msg)
	if err != nil {
		t.Fatalf("Failed to parse participant removal error message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.UserID != expectedUserID {
		t.Errorf("UserID mismatch: expected %s, got %s", expectedUserID, result.UserID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

func publishAndExpectParticipantRemoved(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantRemovalRequestPayload) {
	publishParticipantRemovalRequest(t, deps, &payload)

	if !waitForParticipantRemoved(capture, 1) {
		t.Fatalf("Expected participant removed message")
	}

	msgs := getParticipantRemovedMessages(capture)
	validateParticipantRemovedMessage(t, msgs[0], payload.RoundID, payload.UserID)
}

func publishAndExpectRemovalError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantRemovalRequestPayload, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantRemovalRequest(t, deps, &payload)

	if !waitForParticipantRemovalError(capture, 1) {
		t.Fatalf("Expected participant removal error message")
	}

	msgs := getParticipantRemovalErrorMessages(capture)
	validateParticipantRemovalError(t, msgs[0], expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoRemovalMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantRemovalRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	removedMsgs := getParticipantRemovedMessages(capture)
	errorMsgs := getParticipantRemovalErrorMessages(capture)

	if len(removedMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages for invalid JSON, got %d removed msgs and %d error msgs",
			len(removedMsgs), len(errorMsgs))
	}
}
