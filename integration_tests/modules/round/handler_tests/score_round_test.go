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
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	users := generator.GenerateUsers(3)
	user1ID := sharedtypes.DiscordID(users[0].UserID)
	user2ID := sharedtypes.DiscordID(users[1].UserID)

	testCases := []struct {
		name        string
		setupAndRun func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps)
	}{
		{
			name: "Success - Valid Score Update Request (Under Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateInProgress)

				// Create valid score update request (-2 under par)
				score := sharedtypes.Score(-2)
				payload := createScoreUpdateRequestPayload(roundID, user2ID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result - access through nested structure
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != user2ID {
					t.Errorf("Expected Participant %s, got %s", user2ID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Success - Valid Score Update Request (Over Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateInProgress)

				// Create valid score update request (+5 over par)
				score := sharedtypes.Score(5)
				payload := createScoreUpdateRequestPayload(roundID, user1ID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != user1ID {
					t.Errorf("Expected Participant %s, got %s", user1ID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Success - Valid Score Update Request (Even Par)",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round in progress
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateInProgress)

				// Create valid score update request (even par)
				score := sharedtypes.Score(0)
				payload := createScoreUpdateRequestPayload(roundID, user2ID, &score)

				result := publishAndExpectScoreUpdateSuccess(t, deps, deps.MessageCapture, payload)

				// Validate the result
				if result.ScoreUpdateRequestPayload.RoundID != roundID {
					t.Errorf("Expected RoundID %s, got %s", roundID, result.ScoreUpdateRequestPayload.RoundID)
				}
				if result.ScoreUpdateRequestPayload.Participant != user2ID {
					t.Errorf("Expected Participant %s, got %s", user2ID, result.ScoreUpdateRequestPayload.Participant)
				}
				if *result.ScoreUpdateRequestPayload.Score != score {
					t.Errorf("Expected Score %d, got %d", score, *result.ScoreUpdateRequestPayload.Score)
				}
			},
		},
		{
			name: "Failure - Empty Round ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create score update request with empty round ID
				score := sharedtypes.Score(-1)
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), user1ID, &score)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Empty Participant ID",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateInProgress)

				// Create score update request with empty participant ID
				score := sharedtypes.Score(2)
				payload := createScoreUpdateRequestPayload(roundID, "", &score)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Nil Score",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create a round
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, user1ID, roundtypes.RoundStateInProgress)

				// Create score update request with nil score
				payload := createScoreUpdateRequestPayload(roundID, user1ID, nil)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Multiple Validation Errors",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				// Create score update request with multiple validation errors
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), "", nil)

				publishAndExpectScoreUpdateError(t, deps, deps.MessageCapture, payload)
			},
		},
		{
			name: "Failure - Invalid JSON Message",
			setupAndRun: func(t *testing.T, helper *testutils.RoundTestHelper, deps *RoundHandlerTestDeps) {
				publishInvalidJSONAndExpectNoScoreUpdateMessages(t, deps, deps.MessageCapture)
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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE TESTS
func createScoreUpdateRequestPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateRequestPayload {
	return roundevents.ScoreUpdateRequestPayload{
		RoundID:     roundID,
		Participant: participant,
		Score:       score,
	}
}

// Publishing functions - UNIQUE TO SCORE UPDATE TESTS
func publishScoreUpdateRequestMessage(t *testing.T, deps *RoundHandlerTestDeps, payload *roundevents.ScoreUpdateRequestPayload) *message.Message {
	t.Helper()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateRequest, msg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	return msg
}

func publishInvalidJSONAndExpectNoScoreUpdateMessages(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture) {
	t.Helper()

	// Create invalid JSON message
	invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
	invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundScoreUpdateRequest, invalidMsg); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Wait a bit to ensure no messages are published
	time.Sleep(500 * time.Millisecond)

	successMsgs := getScoreUpdateSuccessFromHandlerMessages(capture)
	errorMsgs := getScoreUpdateErrorFromHandlerMessages(capture)

	if len(successMsgs) > 0 {
		t.Errorf("Expected no success messages for invalid JSON, got %d", len(successMsgs))
	}

	if len(errorMsgs) > 0 {
		t.Errorf("Expected no error messages for invalid JSON, got %d", len(errorMsgs))
	}
}

// Wait functions - UNIQUE TO SCORE UPDATE TESTS
func waitForScoreUpdateSuccessFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoreUpdateValidated, count, defaultTimeout)
}

func waitForScoreUpdateErrorFromHandler(capture *testutils.MessageCapture, count int) bool {
	return capture.WaitForMessages(roundevents.RoundScoreUpdateError, count, defaultTimeout)
}

// Message retrieval functions - UNIQUE TO SCORE UPDATE TESTS
func getScoreUpdateSuccessFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateValidated)
}

func getScoreUpdateErrorFromHandlerMessages(capture *testutils.MessageCapture) []*message.Message {
	return capture.GetMessages(roundevents.RoundScoreUpdateError)
}

// Validation functions - UNIQUE TO SCORE UPDATE TESTS
func validateScoreUpdateSuccessFromHandler(t *testing.T, msg *message.Message) *roundevents.ScoreUpdateValidatedPayload {
	t.Helper()

	result, err := testutils.ParsePayload[roundevents.ScoreUpdateValidatedPayload](msg)
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

	result, err := testutils.ParsePayload[roundevents.RoundScoreUpdateErrorPayload](msg)
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
func publishAndExpectScoreUpdateSuccess(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateRequestPayload) *roundevents.ScoreUpdateValidatedPayload {
	publishScoreUpdateRequestMessage(t, deps, &payload)

	if !waitForScoreUpdateSuccessFromHandler(capture, 1) {
		t.Fatalf("Expected score update success message from %s", roundevents.RoundScoreUpdateValidated)
	}

	msgs := getScoreUpdateSuccessFromHandlerMessages(capture)
	result := validateScoreUpdateSuccessFromHandler(t, msgs[0])

	return result
}

func publishAndExpectScoreUpdateError(t *testing.T, deps *RoundHandlerTestDeps, capture *testutils.MessageCapture, payload roundevents.ScoreUpdateRequestPayload) {
	publishScoreUpdateRequestMessage(t, deps, &payload)

	if !waitForScoreUpdateErrorFromHandler(capture, 1) {
		t.Fatalf("Expected score update error message from %s", roundevents.RoundScoreUpdateError)
	}

	msgs := getScoreUpdateErrorFromHandlerMessages(capture)
	validateScoreUpdateErrorFromHandler(t, msgs[0])
}
