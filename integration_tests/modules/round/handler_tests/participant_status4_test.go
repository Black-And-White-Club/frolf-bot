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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Remove Existing Accepted Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, data.UserID, roundtypes.ResponseAccept)
				payload := createRemovalPayload(roundID, data.UserID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Remove Existing Tentative Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, data.UserID, roundtypes.ResponseTentative)
				payload := createRemovalPayload(roundID, data.UserID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Remove Existing Declined Participant",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundWithParticipantInDB(t, deps.DB, data.UserID, roundtypes.ResponseDecline)
				payload := createRemovalPayload(roundID, data.UserID)
				publishAndExpectParticipantRemoved(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createRemovalPayload(nonExistentRoundID, data.UserID)
				publishAndExpectRemovalError(t, deps, deps.MessageCapture, payload, nonExistentRoundID, data.UserID)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoRemovalMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Clear message capture before each subtest
			deps.MessageCapture.Clear()
			// Create helper for each subtest since these tests use custom helper functions
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			// Run the test
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Test-specific helper functions for HandleParticipantRemovalRequest

func createRemovalPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.ParticipantRemovalRequestPayloadV1 {
	return roundevents.ParticipantRemovalRequestPayloadV1{
		RoundID: roundID,
		UserID:  userID,
		GuildID: "test-guild",
	}
}

func publishParticipantRemovalRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantRemovalRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantRemovalRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForParticipantRemoved(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantRemovedV1, count, defaultTimeout)
}

func waitForParticipantRemovalError(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantRemovalErrorV1, count, defaultTimeout)
}

func getParticipantRemovedMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantRemovedV1)
}

func getParticipantRemovalErrorMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantRemovalErrorV1)
}

func validateParticipantRemovedMessage(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) *roundevents.ParticipantRemovedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantRemovedPayloadV1](msg)
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

	result, err := testutils.ParsePayload[roundevents.ParticipantRemovalErrorPayloadV1](msg)
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

func publishAndExpectParticipantRemoved(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantRemovalRequestPayloadV1) {
	publishParticipantRemovalRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantRemovedMessages(capture)
		// Find the message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.ParticipantRemovedPayloadV1](msg)
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
		t.Fatalf("Expected participant removed message for round %s", payload.RoundID)
	}

	validateParticipantRemovedMessage(t, foundMsg, payload.RoundID, payload.UserID)
}

func publishAndExpectRemovalError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantRemovalRequestPayloadV1, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantRemovalRequest(t, deps, &payload)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantRemovalErrorMessages(capture)
		// Find the message matching THIS test's round ID
		for _, msg := range msgs {
			parsed, err := testutils.ParsePayload[roundevents.ParticipantRemovalErrorPayloadV1](msg)
			if err == nil && parsed.RoundID == expectedRoundID {
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
		t.Fatalf("Expected participant removal error message for round %s", expectedRoundID)
	}

	validateParticipantRemovalError(t, foundMsg, expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoRemovalMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	// Count messages BEFORE
	removedMsgsBefore := len(getParticipantRemovedMessages(capture))
	errorMsgsBefore := len(getParticipantRemovalErrorMessages(capture))

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantRemovalRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Count messages AFTER
	removedMsgsAfter := len(getParticipantRemovedMessages(capture))
	errorMsgsAfter := len(getParticipantRemovalErrorMessages(capture))

	newRemovedMsgs := removedMsgsAfter - removedMsgsBefore
	newErrorMsgs := errorMsgsAfter - errorMsgsBefore

	if newRemovedMsgs > 0 || newErrorMsgs > 0 {
		t.Errorf("Expected no NEW messages for invalid JSON, got %d new removed msgs and %d new error msgs",
			newRemovedMsgs, newErrorMsgs)
	}
}
