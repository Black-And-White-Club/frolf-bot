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

func TestHandleGetRoundRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Retrieve Existing Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round in the database
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateUpcoming)

				// Create get round request payload
				payload := createGetRoundRequestPayload(roundID, data.UserID)

				result := publishAndExpectGetRoundSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ID)
				}
				if result.CreatedBy != data.UserID {
					t.Errorf("Expected CreatedBy %s, got %s", data.UserID, result.CreatedBy)
				}
				if result.State != roundtypes.RoundStateUpcoming {
					t.Errorf("Expected State %s, got %s", roundtypes.RoundStateUpcoming, result.State)
				}
			},
		},
		{
			name: "Success - Retrieve Round with Participants",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create round with participants
				response := roundtypes.ResponseAccept
				tagNumber := sharedtypes.TagNumber(1)
				roundID := helper.CreateUpcomingRoundWithParticipantAndTagInDB(t, deps.DB, data.UserID, response, &tagNumber)

				// Create get round request payload
				payload := createGetRoundRequestPayload(roundID, data.UserID)

				result := publishAndExpectGetRoundSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ID)
				}
				if len(result.Participants) == 0 {
					t.Error("Expected round to have participants")
				}
			},
		},
		{
			name: "Success - Retrieve In-Progress Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create an in-progress round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create get round request payload
				payload := createGetRoundRequestPayload(roundID, data.UserID)

				result := publishAndExpectGetRoundSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ID)
				}
				if result.State != roundtypes.RoundStateInProgress {
					t.Errorf("Expected State %s, got %s", roundtypes.RoundStateInProgress, result.State)
				}
			},
		},
		{
			name: "Success - Retrieve Completed Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a completed round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateFinalized)

				// Create get round request payload
				payload := createGetRoundRequestPayload(roundID, data.UserID)

				result := publishAndExpectGetRoundSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ID)
				}
				if result.State != roundtypes.RoundStateFinalized {
					t.Errorf("Expected State %s, got %s", roundtypes.RoundStateFinalized, result.State)
				}
			},
		},
		{
			name: "Failure - Retrieve Non-Existent Round",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Use a non-existent round ID
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				payload := createGetRoundRequestPayload(nonExistentRoundID, data.UserID)

				publishAndExpectGetRoundError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoGetRoundMessages(t, deps, deps.MessageCapture)
			},
		},
	}

	// Run all subtests with SHARED setup
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			helper := testutils.NewRoundTestHelper(deps.EventBus, deps.MessageCapture)
			tc.setupAndRun(t, helper, &deps)
		})
	}
}

// Helper functions for creating payloads - UNIQUE TO ROUND RETRIEVAL TESTS
func createGetRoundRequestPayload(roundID sharedtypes.RoundID, userID sharedtypes.DiscordID) roundevents.GetRoundRequestPayloadV1 {
	return roundevents.GetRoundRequestPayloadV1{
		GuildID:        "test-guild",
		RoundID:        roundID,
		EventMessageID: "test-event-message-id",
		UserID:         userID,
	}
}

// Publishing functions - UNIQUE TO ROUND RETRIEVAL TESTS
func publishGetRoundRequestMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.GetRoundRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.GetRoundRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoGetRoundMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.GetRoundRequestedV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getGetRoundSuccessFromHandlerMessages(capture)
	errorMsgs := getGetRoundErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO ROUND RETRIEVAL TESTS
func waitForGetRoundSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundRetrievedV1, count, defaultTimeout)
}

func waitForGetRoundErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO ROUND RETRIEVAL TESTS
func getGetRoundSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundRetrievedV1)
}

func getGetRoundErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundErrorV1)
}

// Validation functions - UNIQUE TO ROUND RETRIEVAL TESTS
func validateGetRoundSuccessFromHandler(t *testing.T, msg *message.Message) *roundtypes.Round {
	t.Helper()

	result, err := testutils.ParsePayload[roundtypes.Round](msg)
	if err != nil {
		t.Fatalf("Failed to parse get round success message: %v", err)
	}

	// Validate that RoundID is set
	if result.ID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	// Validate that required fields are present
	if result.CreatedBy == "" {
		t.Error("Expected CreatedBy to be set")
	}

	if result.Title == "" {
		t.Error("Expected Title to be set")
	}

	// Log what we got for debugging
	t.Logf("Round retrieved successfully: %s", result.ID)

	return result
}

func validateGetRoundErrorFromHandler(t *testing.T, msg *message.Message, expectedRoundID sharedtypes.RoundID) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse get round error message: %v", err)
	}

	if result.RoundID != expectedRoundID {
		t.Errorf("RoundID mismatch: expected %s, got %s", expectedRoundID, result.RoundID)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}
}

// Test expectation functions - UNIQUE TO ROUND RETRIEVAL TESTS
func publishAndExpectGetRoundSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.GetRoundRequestPayloadV1) *roundtypes.Round {
	publishGetRoundRequestMessage(t, deps, &payload)

	if !waitForGetRoundSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected get round success message from %s", roundevents.RoundRetrievedV1)
	}

	msgs := getGetRoundSuccessFromHandlerMessages(capture)
	result := validateGetRoundSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectGetRoundError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.GetRoundRequestPayloadV1) {
	publishGetRoundRequestMessage(t, deps, &payload)

	if !waitForGetRoundErrorFromHandler(capture, 1) {
		t.Fatalf("Expected get round error message from %s", roundevents.RoundErrorV1)
	}

	msgs := getGetRoundErrorFromHandlerMessages(capture)
	validateGetRoundErrorFromHandler(t, msgs[0], payload.RoundID)
}
