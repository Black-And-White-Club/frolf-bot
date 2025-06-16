package scorehandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	scoreevents "github.com/Black-And-White-Club/frolf-bot-shared/events/score"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// publishProcessRoundScoresRequest helper to publish a ProcessRoundScoresRequest message.
func publishProcessRoundScoresRequest(t *testing.T, deps ScoreHandlerTestDeps, payload scoreevents.ProcessRoundScoresRequestPayload) *message.Message {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal payload: %v", err)
	}

	msg := message.NewMessage(uuid.New().String(), payloadBytes)
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
	msg.Metadata.Set("topic", scoreevents.ProcessRoundScoresRequest)

	inputTopic := scoreevents.ProcessRoundScoresRequest
	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), inputTopic, msg); err != nil {
		t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
	}
	return msg
}

// validateProcessRoundScoresSuccess helper to validate a LeaderboardBatchTagAssignmentRequested message.
func validateProcessRoundScoresSuccess(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
	t.Helper()
	var requestPayload scoreevents.ProcessRoundScoresRequestPayload
	if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
		t.Fatalf("Failed to unmarshal request payload: %v", err)
	}

	expectedTopic := sharedevents.LeaderboardBatchTagAssignmentRequested
	msgs := receivedMsgs[expectedTopic]
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message on topic %q, got %d", expectedTopic, len(msgs))
	}
	receivedMsg := msgs[0]

	var batchPayload sharedevents.BatchTagAssignmentRequestedPayload
	if err := deps.ScoreModule.Helper.UnmarshalPayload(receivedMsg, &batchPayload); err != nil {
		t.Fatalf("Failed to unmarshal BatchTagAssignmentRequestedPayload: %v", err)
	}

	// Validate that we have a proper UUID for BatchID (not a concatenated string)
	if _, err := uuid.Parse(batchPayload.BatchID); err != nil {
		t.Errorf("BatchID is not a valid UUID: %q, error: %v", batchPayload.BatchID, err)
	}

	// Validate that RequestingUserID is from score service
	if batchPayload.RequestingUserID != "score-service" {
		t.Errorf("Expected RequestingUserID to be 'score-service', got %q", batchPayload.RequestingUserID)
	}

	// Create a map of expected UserID to TagNumber from the incoming message payload
	expectedAssignments := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, scoreInfo := range requestPayload.Scores {
		if scoreInfo.TagNumber != nil {
			expectedAssignments[scoreInfo.UserID] = *scoreInfo.TagNumber
		}
	}

	if len(batchPayload.Assignments) != len(expectedAssignments) {
		t.Errorf("Expected %d tag assignments, got %d", len(expectedAssignments), len(batchPayload.Assignments))
	}

	// Validate the received assignments against the expected assignments
	for _, receivedAssignment := range batchPayload.Assignments {
		expectedTag, ok := expectedAssignments[receivedAssignment.UserID]
		if !ok {
			t.Errorf("Received unexpected tag assignment for UserID %q", receivedAssignment.UserID)
			continue
		}
		if receivedAssignment.TagNumber != expectedTag {
			t.Errorf("Tag number mismatch for UserID %q: expected %d, got %d",
				receivedAssignment.UserID, expectedTag, receivedAssignment.TagNumber)
		}
		delete(expectedAssignments, receivedAssignment.UserID)
	}

	if len(expectedAssignments) > 0 {
		missingAssignments := make([]string, 0, len(expectedAssignments))
		for userID, tagNumber := range expectedAssignments {
			missingAssignments = append(missingAssignments, fmt.Sprintf("UserID: %q, TagNumber: %d", userID, tagNumber))
		}
		t.Errorf("Missing expected tag assignments: %s", strings.Join(missingAssignments, ", "))
	}

	if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Errorf("Correlation ID mismatch: expected %q, got %q",
			incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
			receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
	}

	unexpectedTopic := scoreevents.ProcessRoundScoresFailure
	if len(receivedMsgs[unexpectedTopic]) > 0 {
		t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
	}
}

// validateProcessRoundScoresFailure helper to validate a ProcessRoundScoresFailure message.
func validateProcessRoundScoresFailure(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, expectedErrorSubstring string) {
	t.Helper()
	var requestPayload scoreevents.ProcessRoundScoresRequestPayload
	// Attempt to unmarshal incoming payload, it might be invalid JSON for some failure cases
	_ = json.Unmarshal(incomingMsg.Payload, &requestPayload)

	expectedTopic := scoreevents.ProcessRoundScoresFailure
	msgs := receivedMsgs[expectedTopic]
	if len(msgs) != 1 {
		t.Fatalf("Expected 1 message on topic %q, got %d", expectedTopic, len(msgs))
	}
	receivedMsg := msgs[0]

	var failurePayload scoreevents.ProcessRoundScoresFailurePayload
	if err := deps.ScoreModule.Helper.UnmarshalPayload(receivedMsg, &failurePayload); err != nil {
		t.Fatalf("Failed to unmarshal ProcessRoundScoresFailurePayload: %v", err)
	}

	if requestPayload.RoundID != sharedtypes.RoundID(uuid.Nil) && failurePayload.RoundID != requestPayload.RoundID { // Only check if original RoundID was valid
		t.Errorf("ProcessRoundScoresFailurePayload RoundID mismatch: expected %v, got %v",
			requestPayload.RoundID, failurePayload.RoundID)
	}

	if failurePayload.Error == "" {
		t.Errorf("Expected non-empty error message in failure payload")
	}
	if expectedErrorSubstring != "" && !strings.Contains(strings.ToLower(failurePayload.Error), strings.ToLower(expectedErrorSubstring)) {
		t.Errorf("Expected error message to contain %q, got: %s", expectedErrorSubstring, failurePayload.Error)
	}

	if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Errorf("Correlation ID mismatch: expected %q, got %q",
			incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
			receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
	}

	unexpectedTopic := sharedevents.LeaderboardBatchTagAssignmentRequested
	if len(receivedMsgs[unexpectedTopic]) > 0 {
		t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
	}
}

func TestHandleProcessRoundScoresRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(42)

	testCases := []struct {
		name                   string
		setupFn                func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) interface{} // Returns initial state
		publishMsgFn           func(t *testing.T, deps ScoreHandlerTestDeps, initialState interface{}, generator *testutils.TestDataGenerator) *message.Message
		validateFn             func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name: "Success - Process Valid Round Scores",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) interface{} {
				return nil // No specific initial state needed for publishMsgFn
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, initialState interface{}, generator *testutils.TestDataGenerator) *message.Message {
				users := generator.GenerateUsers(5)
				discGolfScores := []float64{0, -2, +3, -4, +1}
				tag1 := sharedtypes.TagNumber(42)
				tag3 := sharedtypes.TagNumber(17)

				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(discGolfScores[0]), TagNumber: &tag1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: sharedtypes.Score(discGolfScores[1])},
					{UserID: sharedtypes.DiscordID(users[2].UserID), Score: sharedtypes.Score(discGolfScores[2]), TagNumber: &tag3},
					{UserID: sharedtypes.DiscordID(users[3].UserID), Score: sharedtypes.Score(discGolfScores[3])},
					{UserID: sharedtypes.DiscordID(users[4].UserID), Score: sharedtypes.Score(discGolfScores[4])},
				}

				roundID := sharedtypes.RoundID(uuid.New())
				payload := scoreevents.ProcessRoundScoresRequestPayload{RoundID: roundID, Scores: scores}
				return publishProcessRoundScoresRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validateProcessRoundScoresSuccess(t, deps, incomingMsg, receivedMsgs)
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardBatchTagAssignmentRequested},
			expectHandlerError:     false,
			timeout:                10 * time.Second,
		},
		{
			name: "Failure - Empty Scores List",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, initialState interface{}, generator *testutils.TestDataGenerator) *message.Message {
				roundID := sharedtypes.RoundID(uuid.New())
				payload := scoreevents.ProcessRoundScoresRequestPayload{RoundID: roundID, Scores: []sharedtypes.ScoreInfo{}}
				return publishProcessRoundScoresRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				validateProcessRoundScoresFailure(t, deps, incomingMsg, receivedMsgs, "empty")
			},
			expectedOutgoingTopics: []string{scoreevents.ProcessRoundScoresFailure},
			expectHandlerError:     false, // The handler should publish a failure message and acknowledge
			timeout:                5 * time.Second,
		},
		{
			name: "Failure - Invalid Message Payload",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, initialState interface{}, generator *testutils.TestDataGenerator) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", scoreevents.ProcessRoundScoresRequest)

				inputTopic := scoreevents.ProcessRoundScoresRequest
				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// For an unmarshal error, the handler returns an error to Watermill.
				// It does NOT publish an outgoing message in this scenario.
				// Therefore, we assert that no messages were received on any topic.
				for topic, msgs := range receivedMsgs {
					if len(msgs) > 0 {
						t.Errorf("Expected no messages on topic %q for unmarshal error, but received %d", topic, len(msgs))
					}
				}
			},
			expectedOutgoingTopics: []string{}, // <--- Changed to an empty slice
			expectHandlerError:     true,       // <--- Retained: handler is expected to return an error to Watermill
			timeout:                5 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for use in closures
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestScoreHandler(t) // Setup per-test dependencies for this subtest

			// Execute the custom setup function to get the initial state
			initialState := tc.setupFn(t, deps, generator)

			// Construct the generic testutils.TestCase
			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					// This SetupFn for testutils.TestCase should just return the already setup initialState
					return initialState
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					// Pass the initialState to the custom publishMsgFn
					return tc.publishMsgFn(t, deps, initialState, generator)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, receivedMsgs map[string][]*message.Message, currentState interface{}) {
					// Pass the initialState to the custom validateFn
					tc.validateFn(t, deps, incoming, receivedMsgs, currentState)
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout, // Pass the timeout
			}
			// Run the test using the testutils.RunTest helper
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
