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
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with Tag Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(42)
				payload := createTagLookupFoundPayload(roundID, userID, &tagNumber, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFound)
			},
		},
		{
			name: "Success - Tentative Response with Tag Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				tagNumber := sharedtypes.TagNumber(99)
				payload := createTagLookupFoundPayload(roundID, userID, &tagNumber, roundtypes.ResponseTentative, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFound)
			},
		},
		{
			name: "Success - Accept Response with Late Join",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateInProgress)
				tagNumber := sharedtypes.TagNumber(77)
				payload := createTagLookupFoundPayload(roundID, userID, &tagNumber, roundtypes.ResponseAccept, boolPtr(true))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFound)
			},
		},
		{
			name: "Failure - Round Not Found (Tag Found)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				tagNumber := sharedtypes.TagNumber(42)
				payload := createTagLookupFoundPayload(nonExistentRoundID, userID, &tagNumber, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectJoinErrorFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupFound, nonExistentRoundID, userID)
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

func TestHandleTagNumberNotFound(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response with No Tag (Still Allows Join)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createTagLookupNotFoundPayload(roundID, userID, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFound)
			},
		},
		{
			name: "Success - Tentative Response with No Tag",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createTagLookupNotFoundPayload(roundID, userID, roundtypes.ResponseTentative, boolPtr(false))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFound)
			},
		},
		{
			name: "Success - Accept Response with Late Join and No Tag",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateInProgress)
				payload := createTagLookupNotFoundPayload(roundID, userID, roundtypes.ResponseAccept, boolPtr(true))
				publishAndExpectParticipantJoinedFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFound)
			},
		},
		{
			name: "Failure - Round Not Found (Tag Not Found)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createTagLookupNotFoundPayload(nonExistentRoundID, userID, roundtypes.ResponseAccept, boolPtr(false))
				publishAndExpectJoinErrorFromTagLookup(t, deps, deps.MessageCapture, payload, sharedevents.RoundTagLookupNotFound, nonExistentRoundID, userID)
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

func TestHandleTagLookupInvalidMessages(t *testing.T) {
	testCases := []struct {
		name        string
		topic       string
		setupAndRun func(t *testing.T, deps *RoundHandlerTestDeps)
	}{
		{
			name:  "Invalid JSON - Tag Found Handler",
			topic: sharedevents.RoundTagLookupFound,
			setupAndRun: func(t *testing.T, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoTagLookupMessages(t, deps, deps.MessageCapture, sharedevents.RoundTagLookupFound)
			},
		},
		{
			name:  "Invalid JSON - Tag Not Found Handler",
			topic: sharedevents.RoundTagLookupNotFound,
			setupAndRun: func(t *testing.T, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoTagLookupMessages(t, deps, deps.MessageCapture, sharedevents.RoundTagLookupNotFound)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)

			helper.ClearMessages()
			tc.setupAndRun(t, &deps)

			time.Sleep(1 * time.Second)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO TAG LOOKUP TESTS
func createTagLookupFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, tagNumber *sharedtypes.TagNumber, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayload {
	return sharedevents.RoundTagLookupResultPayload{
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          tagNumber,
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

func createTagLookupNotFoundPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, originalResponse roundtypes.Response, originalJoinedLate *bool) sharedevents.RoundTagLookupResultPayload {
	return sharedevents.RoundTagLookupResultPayload{
		RoundID:            roundID,
		UserID:             userID,
		TagNumber:          nil, // No tag found
		OriginalResponse:   originalResponse,
		OriginalJoinedLate: originalJoinedLate,
	}
}

// Publishing functions - UNIQUE TO TAG LOOKUP TESTS
func publishTagLookupResultMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *sharedevents.RoundTagLookupResultPayload, topic string) *message.Message {
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
	return capture.WaitForMessages(roundevents.RoundParticipantJoined, count, defaultTimeout)
}

func waitForParticipantJoinErrorFromTagLookup(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO TAG LOOKUP TESTS
func getParticipantJoinedFromTagLookupMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoined)
}

func getParticipantJoinErrorFromTagLookupMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinError)
}

// Validation functions - UNIQUE TO TAG LOOKUP TESTS
func validateParticipantJoinedFromTagLookup(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) *roundevents.ParticipantJoinedPayload {
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

func validateParticipantJoinErrorFromTagLookup(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
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

// Test expectation functions - UNIQUE TO TAG LOOKUP TESTS
func publishAndExpectParticipantJoinedFromTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload sharedevents.RoundTagLookupResultPayload, topic string) {
	publishTagLookupResultMessage(t, deps, &payload, topic)

	if !waitForParticipantJoinedFromTagLookup(capture, 1) {
		t.Fatalf("Expected participant joined message from %s", topic)
	}

	msgs := getParticipantJoinedFromTagLookupMessages(capture)
	validateParticipantJoinedFromTagLookup(t, msgs[0], payload.RoundID)
}

func publishAndExpectJoinErrorFromTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload sharedevents.RoundTagLookupResultPayload, topic string, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishTagLookupResultMessage(t, deps, &payload, topic)

	if !waitForParticipantJoinErrorFromTagLookup(capture, 1) {
		t.Fatalf("Expected participant join error message from %s", topic)
	}

	msgs := getParticipantJoinErrorFromTagLookupMessages(capture)
	validateParticipantJoinErrorFromTagLookup(t, msgs[0], expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoTagLookupMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, topic string) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), topic, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	joinedMsgs := getParticipantJoinedFromTagLookupMessages(capture)
	errorMsgs := getParticipantJoinErrorFromTagLookupMessages(capture)

	if len(joinedMsgs) > 0 || len(errorMsgs) > 0 {
		t.Errorf("Expected no messages for invalid JSON on %s, got %d joined, %d error msgs",
			topic, len(joinedMsgs), len(errorMsgs))
	}
}
