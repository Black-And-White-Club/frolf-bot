package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleTagNumberFound(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with Tag Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(42)
				payload := createTagLookupFoundPayload(roundID, data.UserID, &tagNumber, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFoundV1)
			},
		},
		{
			name: "Success - Tentative Response with Tag Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(99)
				payload := createTagLookupFoundPayload(roundID, data.UserID, &tagNumber, roundtypes.ResponseTentative, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFoundV1)
			},
		},
		{
			name: "Success - Accept Response with Late Join",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				tagNumber := sharedtypes.TagNumber(77)
				payload := createTagLookupFoundPayload(roundID, data.UserID, &tagNumber, roundtypes.ResponseAccept, boolPtr(true))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFoundV1)
			},
		},
		{
			name: "Failure - Round Not Found (Tag Found)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				tagNumber := sharedtypes.TagNumber(42)
				payload := createTagLookupFoundPayload(nonExistentRoundID, data.UserID, &tagNumber, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectJoinErrorFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFoundV1, nonExistentRoundID, data.UserID)
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

func TestHandleTagNumberNotFound(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with No Tag (Still Allows Join)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				payload := createTagLookupNotFoundPayload(roundID, data.UserID, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFoundV1)
			},
		},
		{
			name: "Success - Tentative Response with No Tag",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)
				payload := createTagLookupNotFoundPayload(roundID, data.UserID, roundtypes.ResponseTentative, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFoundV1)
			},
		},
		{
			name: "Success - Accept Response with Late Join and No Tag",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)
				payload := createTagLookupNotFoundPayload(roundID, data.UserID, roundtypes.ResponseAccept, boolPtr(true))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFoundV1)
			},
		},
		{
			name: "Failure - Round Not Found (Tag Not Found)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createTagLookupNotFoundPayload(nonExistentRoundID, data.UserID, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectJoinErrorFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFoundV1, nonExistentRoundID, data.UserID)
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

func TestHandleTagLookupInvalidMessages(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		topic       string
		setupAndRun func(t *testing.T, deps *RoundHandlerTestDeps)
	}{
		{
			name:  "Invalid JSON - Tag Found Handler",
			topic: sharedevents.RoundTagLookupFoundV1,
			setupAndRun: func(t *testing.T, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoTagLookupMessages(t, deps, deps.MessageCapture, sharedevents.RoundTagLookupFoundV1)
			},
		},
		{
			name:  "Invalid JSON - Tag Not Found Handler",
			topic: sharedevents.RoundTagLookupNotFoundV1,
			setupAndRun: func(t *testing.T, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoTagLookupMessages(t, deps, deps.MessageCapture, sharedevents.RoundTagLookupNotFoundV1)
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
			_ = helper
			tc.setupAndRun(t, &deps)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO TAG LOOKUP TESTS
func createTagLookupFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, tagNumber *sharedtypes.TagNumber, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayloadV1 {
	return sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: "test-guild"},
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          tagNumber,
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

func createTagLookupNotFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayloadV1 {
	return sharedevents.RoundTagLookupResultPayloadV1{
		ScopedGuildID:      sharedevents.ScopedGuildID{GuildID: "test-guild"},
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          nil, // No tag found
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

// Publishing functions - UNIQUE TO TAG LOOKUP TESTS
func publishTagLookupResultMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *sharedevents.RoundTagLookupResultPayloadV1, topic string) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

// Wait functions - UNIQUE TO TAG LOOKUP TESTS
func waitForParticipantJoinedFromTagLookup(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinedV1, count, defaultTimeout)
}

func waitForParticipantJoinErrorFromTagLookup(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO TAG LOOKUP TESTS
func getParticipantJoinedFromTagLookupMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinedV1)
}

func getParticipantJoinErrorFromTagLookupMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinErrorV1)
}

// Validation functions - UNIQUE TO TAG LOOKUP TESTS
func validateParticipantJoinedFromTagLookup(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayloadV1 {
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

func validateParticipantJoinErrorFromTagLookup(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
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

// Test expectation functions - UNIQUE TO TAG LOOKUP TESTS
func publishAndExpectParticipantJoinedFromTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload sharedevents.RoundTagLookupResultPayloadV1, topic string) {
	publishTagLookupResultMessage(t, deps, &payload, topic)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantJoinedFromTagLookupMessages(capture)
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
		t.Fatalf("Expected participant joined message from %s for round %s", topic, payload.RoundID)
	}

	validateParticipantJoinedFromTagLookup(t, foundMsg, payload.RoundID)
}

func publishAndExpectJoinErrorFromTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload sharedevents.RoundTagLookupResultPayloadV1, topic string, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishTagLookupResultMessage(t, deps, &payload, topic)

	// Wait up to 1 second for the specific message for THIS round
	deadline := time.Now().Add(1 * time.Second)
	var foundMsg *message.Message
	for time.Now().Before(deadline) {
		msgs := getParticipantJoinErrorFromTagLookupMessages(capture)
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
		t.Fatalf("Expected participant join error message from %s for round %s", topic, expectedRoundID)
	}

	validateParticipantJoinErrorFromTagLookup(t, foundMsg, expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoTagLookupMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, topic string) {
	// Count messages BEFORE
	joinedMsgsBefore := len(getParticipantJoinedFromTagLookupMessages(capture))
	errorMsgsBefore := len(getParticipantJoinErrorFromTagLookupMessages(capture))

	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Count messages AFTER
	joinedMsgsAfter := len(getParticipantJoinedFromTagLookupMessages(capture))
	errorMsgsAfter := len(getParticipantJoinErrorFromTagLookupMessages(capture))

	newJoinedMsgs := joinedMsgsAfter - joinedMsgsBefore
	newErrorMsgs := errorMsgsAfter - errorMsgsBefore

	if newJoinedMsgs > 0 || newErrorMsgs > 0 {
		t.Errorf("Expected no NEW messages for invalid JSON on %s, got %d new joined, %d new error msgs",
			topic, newJoinedMsgs, newErrorMsgs)
	}
}
