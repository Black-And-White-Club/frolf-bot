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

const defaultTimeout = 3 * time.Second

func TestHandleParticipantJoinValidationRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	user := generator.GenerateUsers(1)[0]
	userID := sharedtypes.DiscordID(user.UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Accept Response for Scheduled Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createPayload(roundID, userID, roundtypes.ResponseAccept)
				publishAndExpectTagLookup(t, deps, deps.MessageCapture, payload, false)
			},
		},
		{
			name: "Success - Accept Response for InProgress Round (Late Join)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateInProgress)
				payload := createPayload(roundID, userID, roundtypes.ResponseAccept)
				publishAndExpectTagLookup(t, deps, deps.MessageCapture, payload, true)
			},
		},
		{
			name: "Success - Decline Response for Scheduled Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createPayload(roundID, userID, roundtypes.ResponseDecline)
				publishAndExpectStatusUpdate(t, deps, deps.MessageCapture, payload, false)
			},
		},
		{
			name: "Failure - Round Not Found",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createPayload(nonExistentRoundID, userID, roundtypes.ResponseAccept)
				publishAndExpectError(t, deps, deps.MessageCapture, payload, nonExistentRoundID, userID)
			},
		},
		{
			name: "Failure - Empty User ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, userID, roundtypes.RoundStateUpcoming)
				payload := createPayload(roundID, "", roundtypes.ResponseAccept)
				publishAndExpectError(t, deps, deps.MessageCapture, payload, roundID, "")
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoMessages(t, deps, deps.MessageCapture)
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

// Test-specific helper functions - only used in this file

func createPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID, response roundtypes.Response) roundevents.ParticipantJoinValidationRequestPayload {
	return roundevents.ParticipantJoinValidationRequestPayload{
		RoundID:  roundID,
		UserID:   userID,
		Response: response,
	}
}

func publishParticipantJoinValidationRequest(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ParticipantJoinValidationRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantJoinValidationRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func waitForTagLookupRequest(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.LeaderboardGetTagNumberRequest, count, defaultTimeout)
}

func getTagLookupRequestMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.LeaderboardGetTagNumberRequest)
}

func waitForParticipantStatusUpdateRequest(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantStatusUpdateRequest, count, defaultTimeout)
}

func waitForParticipantJoinError(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundParticipantJoinError, count, defaultTimeout)
}

func getParticipantStatusUpdateRequestMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantStatusUpdateRequest)
}

func getParticipantJoinErrorMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundParticipantJoinError)
}

func validateTagLookupRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.TagLookupRequestPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.TagLookupRequestPayload](msg)
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

func validateParticipantStatusUpdateRequest(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID, expectedResponse roundtypes.Response) *roundevents.ParticipantJoinRequestPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ParticipantJoinRequestPayload](msg)
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

func publishAndExpectTagLookup(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayload, expectedLateJoin bool) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	if !waitForTagLookupRequest(capture, 1) {
		t.Fatalf("Expected tag lookup request message")
	}

	msgs := getTagLookupRequestMessages(capture)
	result := validateTagLookupRequest(t, msgs[0], payload.RoundID, payload.UserID, payload.Response)

	if result.JoinedLate == nil || *result.JoinedLate != expectedLateJoin {
		t.Errorf("Expected JoinedLate=%t, got %v", expectedLateJoin, result.JoinedLate)
	}
}

func publishAndExpectStatusUpdate(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayload, expectedLateJoin bool) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	if !waitForParticipantStatusUpdateRequest(capture, 1) {
		t.Fatalf("Expected participant status update request message")
	}

	msgs := getParticipantStatusUpdateRequestMessages(capture)
	result := validateParticipantStatusUpdateRequest(t, msgs[0], payload.RoundID, payload.UserID, payload.Response)

	if result.JoinedLate == nil || *result.JoinedLate != expectedLateJoin {
		t.Errorf("Expected JoinedLate=%t, got %v", expectedLateJoin, result.JoinedLate)
	}
}

func publishAndExpectError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ParticipantJoinValidationRequestPayload, expectedRoundID sharedtypes.RoundID, expectedUserID sharedtypes.DiscordID) {
	publishParticipantJoinValidationRequest(t, deps, &payload)

	if !waitForParticipantJoinError(capture, 1) {
		t.Fatalf("Expected participant join error message")
	}

	msgs := getParticipantJoinErrorMessages(capture)
	validateParticipantJoinError(t, msgs[0], expectedRoundID, expectedUserID)
}

func publishInvalidJSONAndExpectNoMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	msg := message.NewMessage(uuid.New().String(), []byte("not valid json"))
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundParticipantJoinValidationRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	time.Sleep(1 * time.Second)

	tagMsgs := getTagLookupRequestMessages(capture)
	statusMsgs := getParticipantStatusUpdateRequestMessages(capture)

	if len(tagMsgs) > 0 || len(statusMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d tag msgs and %d status msgs",
			len(tagMsgs), len(statusMsgs))
	}
}
