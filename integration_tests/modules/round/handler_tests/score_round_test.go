package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestHandleScoreUpdateRequest(t *testing.T) {
	// Setup ONCE for all subtests
	deps := SetupTestRoundHandler(t)


	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Valid Score Update Request (Under Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create valid score update request (-2 under par)
				score := sharedtypes.Score(-2)
				payload := createScoreUpdateRequestPayload(roundID, data2.UserID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result - access through nested structure
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != data2.UserID {
					t.Errorf("Expected Participant %s, got %s", data2.UserID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Success - Valid Score Update Request (Over Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create valid score update request (+5 over par)
				score := sharedtypes.Score(5)
				payload := createScoreUpdateRequestPayload(roundID, data.UserID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != data.UserID {
					t.Errorf("Expected Participant %s, got %s", data.UserID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Success - Valid Score Update Request (Even Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				data2 := NewTestData()
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create valid score update request (even par)
				score := sharedtypes.Score(0)
				payload := createScoreUpdateRequestPayload(roundID, data2.UserID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != data2.UserID {
					t.Errorf("Expected Participant %s, got %s", data2.UserID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Failure - Empty Round ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create score update request with empty round ID
				score := sharedtypes.Score(-1)
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), data.UserID, &score)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Empty Participant ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create score update request with empty participant ID
				score := sharedtypes.Score(2)
				payload := createScoreUpdateRequestPayload(roundID, "", &score)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Nil Score",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				data := NewTestData()
				// Create a round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create score update request with nil score
				payload := createScoreUpdateRequestPayload(roundID, data.UserID, nil)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Multiple Validation Errors",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				// Create score update request with multiple validation errors
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), "", nil)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				deps.MessageCapture.Clear()
				publishInvalidJSONAndExpectNoScoreUpdateMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE TESTS
func createScoreUpdateRequestPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateRequestPayloadV1 {
	return roundevents.ScoreUpdateRequestPayloadV1{
		GuildID:     "test-guild",
		RoundID:     roundID,
		Participant: participant,
		Score:       score,
	}
}

// Publishing functions - UNIQUE TO SCORE UPDATE TESTS
func publishScoreUpdateRequestMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ScoreUpdateRequestPayloadV1) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Count BEFORE
	successBefore := len(getScoreUpdateSuccessFromHandlerMessages(capture))
	errorBefore := len(getScoreUpdateErrorFromHandlerMessages(capture))

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateRequestedV1, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no NEW messages are published
	time.Sleep(500 * time.Millisecond)

	// Count AFTER
	successAfter := len(getScoreUpdateSuccessFromHandlerMessages(capture))
	errorAfter := len(getScoreUpdateErrorFromHandlerMessages(capture))

	newSuccess := successAfter - successBefore
	newErrors := errorAfter - errorBefore

	if newSuccess > 0 {
		t.Errorf("Expected no NEW success messages for invalid JSON, got %d", newSuccess)
	}

	if newErrors > 0 {
		t.Errorf("Expected no NEW error messages for invalid JSON, got %d", newErrors)
	}
}

// Wait functions - UNIQUE TO SCORE UPDATE TESTS
func waitForScoreUpdateSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoreUpdateValidatedV1, count, defaultTimeout)
}

func waitForScoreUpdateErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoreUpdateErrorV1, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO SCORE UPDATE TESTS
func getScoreUpdateSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateValidatedV1)
}

func getScoreUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateErrorV1)
}

// Validation functions - UNIQUE TO SCORE UPDATE TESTS
func validateScoreUpdateSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.ScoreUpdateValidatedPayloadV1 {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ScoreUpdateValidatedPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse score update success message: %v", err)
	}

	// Validate that required fields are set - access through nested structure
	if result.ScoreUpdateRequestPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
		t.Error("Expected RoundID to be set")
	}

	if result.ScoreUpdateRequestPayload.Participant == "" {
		t.Error("Expected Participant to be set")
	}

	if result.ScoreUpdateRequestPayload.Score == nil {
		t.Error("Expected Score to be set")
	}

	// Log what we got for debugging
	scoreText := "even"
	if *result.ScoreUpdateRequestPayload.Score > 0 {
		scoreText = fmt.Sprintf("+%d over", *result.ScoreUpdateRequestPayload.Score)
	} else if *result.ScoreUpdateRequestPayload.Score < 0 {
		scoreText = fmt.Sprintf("%d under", *result.ScoreUpdateRequestPayload.Score)
	}

	t.Logf("Score update validated successfully for round: %s, participant: %s, score: %s par",
		result.ScoreUpdateRequestPayload.RoundID, result.ScoreUpdateRequestPayload.Participant, scoreText)

	return result
}

func validateScoreUpdateErrorFromHandler(t *testing.T, msg *message.Message) {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.RoundScoreUpdateErrorPayloadV1](msg)
	if err != nil {
		t.Fatalf("Failed to parse score update error message: %v", err)
	}

	if result.Error == "" {
		t.Error("Expected error message to be populated")
	}

	if result.ScoreUpdateRequest == nil {
		t.Error("Expected ScoreUpdateRequest to be populated")
	}

	// Log what we got for debugging
	t.Logf("Score update validation failed with error: %s", result.Error)
}

// Test expectation functions - UNIQUE TO SCORE UPDATE TESTS
func publishAndExpectScoreUpdateSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateRequestPayloadV1) *roundevents.ScoreUpdateValidatedPayloadV1 {
	publishScoreUpdateRequestMessage(t, deps, &payload)

	if !waitForScoreUpdateSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected score update success message from %s", roundevents.RoundScoreUpdateValidatedV1)
	}

	msgs := getScoreUpdateSuccessFromHandlerMessages(capture)
	result := validateScoreUpdateSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectScoreUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateRequestPayloadV1) {
	publishScoreUpdateRequestMessage(t, deps, &payload)

	if !waitForScoreUpdateErrorFromHandler(capture, 1) {
		t.Fatalf("Expected score update error message from %s", roundevents.RoundScoreUpdateErrorV1)
	}

	msgs := getScoreUpdateErrorFromHandlerMessages(capture)
	validateScoreUpdateErrorFromHandler(t, msgs[0])
}
