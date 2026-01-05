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
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Participant Declined (Upcoming Round)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				payload := createParticipantDeclinedPayload(roundID, data.UserID)
				publishAndExpectParticipantDeclinedFromHandler(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Success - Participant Declined (In Progress Round)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				payload := createParticipantDeclinedPayload(roundID, data.UserID)
				publishAndExpectParticipantDeclinedFromHandler(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Round Not Found (Participant Declined)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createParticipantDeclinedPayload(nonExistentRoundID, data.UserID)
				publishAndExpectDeclineErrorFromHandler(t, deps, deps.MessageCapture, payload, nonExistentRoundID, data.UserID)
			},
		},
		{
			name: "Invalid JSON - Participant Declined Handler",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoDeclineMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO PARTICIPANT DECLINED TESTS
func createParticipantDeclinedPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.ParticipantDeclinedPayloadV1 {
	return roundevents.ParticipantDeclinedPayloadV1{
		GuildID: "test-guild", // Must match the guild ID in CreateRoundInDBWithState
		RoundID: roundID,
		UserID:  userID,
	}
}

// Publishing functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func publishParticipantDeclinedMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantDeclinedPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantDeclinedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// Wait functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func waitForParticipantDeclinedFromHandler(capture *testutils.MessageCapture, count int) bool {
	// Success cases (including declines) should go to RoundParticipantJoined
	return capture.WaitForMessages(roundevents.RoundParticipantJoinedV1, count, defaultTimeout)
}

func waitForParticipantDeclineErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func getParticipantDeclinedFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	// Success cases (including declines) should go to RoundParticipantJoined
	return capture.GetMessages(roundevents.RoundParticipantJoinedV1)
}

func getParticipantDeclineErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

// Validation functions - UNIQUE TO PARTICIPANT DECLINED TESTS
func validateParticipantDeclinedFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayloadV1](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundParticipantJoinErrorPayloadV1](msg)
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
func publishAndExpectParticipantDeclinedFromHandler(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantDeclinedPayloadV1) {
	publishParticipantDeclinedMessage(t, deps, &payload)

	// Wait up to default timeout for the specific message for THIS round
	deadline := time.Now().Add(defaultTimeout)
	var foundMsg *message.Message
	
	// Poll more frequently for better responsiveness
	for time.Now().Before(deadline) {
		msgs := getParticipantDeclinedFromHandlerMessages(capture)
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
		time.Sleep(50 * time.Millisecond)
	}

	if foundMsg == nil {
		// Debug: show what messages we did receive
		allMsgs := getParticipantDeclinedFromHandlerMessages(capture)
		t.Logf("DEBUG: Found %d total ParticipantJoined messages, but none matched round %s", len(allMsgs), payload.RoundID)
		for i, msg := range allMsgs {
			if parsed, err := testutils.ParsePayload[roundevents.ParticipantJoinedPayloadV1](msg); err == nil {
				t.Logf("  Message %d: round_id=%s", i+1, parsed.RoundID)
			}
		}
		t.Fatalf("Expected participant declined success message from %s for round %s", roundevents.RoundParticipantJoinedV1, payload.RoundID)
	}

	validateParticipantDeclinedFromHandler(t, foundMsg, payload.RoundID)
}

func publishAndExpectDeclineErrorFromHandler(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantDeclinedPayloadV1, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantDeclinedMessage(t, deps, &payload)

	// Wait up to default timeout for the specific message for THIS round
	deadline := time.Now().Add(defaultTimeout)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantDeclineErrorFromHandlerMessages(capture)
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
		time.Sleep(50 * time.Millisecond)
	}

	if foundMsg == nil {
		t.Fatalf("Expected participant decline error message from %s for round %s", roundevents.RoundParticipantJoinErrorV1, expectedRoundID)
	}

	validateParticipantDeclineErrorFromHandler(t, foundMsg, expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoDeclineMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	// Count messages BEFORE
	declinedMsgsBefore := len(getParticipantDeclinedFromHandlerMessages(capture))
	errorMsgsBefore := len(getParticipantDeclineErrorFromHandlerMessages(capture))

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantDeclinedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Count messages AFTER
	declinedMsgsAfter := len(getParticipantDeclinedFromHandlerMessages(capture))
	errorMsgsAfter := len(getParticipantDeclineErrorFromHandlerMessages(capture))

	newDeclinedMsgs := declinedMsgsAfter - declinedMsgsBefore
	newErrorMsgs := errorMsgsAfter - errorMsgsBefore

	if newDeclinedMsgs > 0 || newErrorMsgs > 0 {
		t.Errorf("Expected no NEW messages for invalid JSON on %s, got %d new success, %d new error msgs",
			roundevents.RoundParticipantDeclinedV1, newDeclinedMsgs, newErrorMsgs)
	}
}
