package scorehandler_integration_tests

import (
	"context"
	"encoding/json"
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

// publishScoreUpdateRequest helper to publish a ScoreUpdateRequest message.
func publishScoreUpdateRequest(t *testing.T, deps ScoreHandlerTestDeps, payload scoreevents.ScoreUpdateRequestPayload) *message.Message {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request payload: %v", err)
	}
	msg := message.NewMessage(uuid.New().String(), data)
	correlationID := uuid.New().String()
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	msg.Metadata.Set("topic", scoreevents.ScoreUpdateRequest)

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), scoreevents.ScoreUpdateRequest, msg); err != nil {
		t.Fatalf("Failed to publish ScoreUpdateRequest: %v", err)
	}
	return msg
}

// validateScoreUpdateSuccess helper to validate a ScoreUpdateSuccess message.
func validateScoreUpdateSuccess(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, responseMsg *message.Message) {
	t.Helper()
	var successPayload scoreevents.ScoreUpdateSuccessPayload
	if err := deps.ScoreModule.Helper.UnmarshalPayload(responseMsg, &successPayload); err != nil {
		t.Fatalf("Unmarshal error for ScoreUpdateSuccessPayload: %v", err)
	}

	var incomingPayload scoreevents.ScoreUpdateRequestPayload
	if err := json.Unmarshal(incomingMsg.Payload, &incomingPayload); err != nil {
		t.Fatalf("Failed to unmarshal incoming payload: %v", err)
	}

	if successPayload.RoundID != incomingPayload.RoundID {
		t.Errorf("ScoreUpdateSuccessPayload RoundID mismatch: expected %v, got %v", incomingPayload.RoundID, successPayload.RoundID)
	}
	if successPayload.UserID != incomingPayload.UserID {
		t.Errorf("ScoreUpdateSuccessPayload UserID mismatch: expected %q, got %q", incomingPayload.UserID, successPayload.UserID)
	}
	if successPayload.Score != incomingPayload.Score {
		t.Errorf("ScoreUpdateSuccessPayload Score mismatch: expected %v, got %v", incomingPayload.Score, successPayload.Score)
	}
	if responseMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Error("Correlation ID mismatch in success message")
	}
}

// validateScoreUpdateFailure helper to validate a ScoreUpdateFailure message.
func validateScoreUpdateFailure(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, responseMsg *message.Message) {
	t.Helper()
	var failurePayload scoreevents.ScoreUpdateFailurePayload
	if err := deps.ScoreModule.Helper.UnmarshalPayload(responseMsg, &failurePayload); err != nil {
		t.Fatalf("Unmarshal error for ScoreUpdateFailurePayload: %v", err)
	}

	var incomingPayload scoreevents.ScoreUpdateRequestPayload
	if err := json.Unmarshal(incomingMsg.Payload, &incomingPayload); err != nil {
		t.Fatalf("Failed to unmarshal incoming payload: %v", err)
	}

	if failurePayload.RoundID != incomingPayload.RoundID {
		t.Errorf("ScoreUpdateFailurePayload RoundID mismatch: expected %v, got %v", incomingPayload.RoundID, failurePayload.RoundID)
	}
	if failurePayload.UserID != incomingPayload.UserID {
		t.Errorf("ScoreUpdateFailurePayload UserID mismatch: expected %q, got %q", incomingPayload.UserID, failurePayload.UserID)
	}
	if failurePayload.Error == "" {
		t.Error("Expected non-empty error message in failure payload")
	}
	if responseMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Error("Correlation ID mismatch in failure message")
	}
}

func TestHandleCorrectScoreRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	testCases := []struct {
		name                   string
		users                  []testutils.User // Users defined at the test case level
		setupFn                func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID
		publishMsgFn           func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message
		validateFn             func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialRoundID sharedtypes.RoundID)
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name:  "Success - Correct score with valid data",
			users: generator.GenerateUsers(2), // Pre-generate users for this test case
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				ctx := deps.Ctx

				roundID := sharedtypes.RoundID(uuid.New())
				tag1 := sharedtypes.TagNumber(42)
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(0), TagNumber: &tag1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: sharedtypes.Score(5)},
				}

				guildID := sharedtypes.GuildID("test-guild") // Use a test guild ID
				processResult, processErr := deps.ScoreModule.ScoreService.ProcessRoundScores(ctx, guildID, roundID, scores)
				if processErr != nil || processResult.Error != nil {
					t.Fatalf("Failed to process initial round scores for setup: %v, result: %+v", processErr, processResult.Error)
				}
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(-3),
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				expectedSuccessTopic := scoreevents.ScoreUpdateSuccess
				successMsgs := received[expectedSuccessTopic]
				if len(successMsgs) != 1 {
					t.Fatalf("Expected 1 success message on topic %q, got %d", expectedSuccessTopic, len(successMsgs))
				}
				validateScoreUpdateSuccess(t, deps, incomingMsg, successMsgs[0])

				unexpectedFailureTopic := scoreevents.ScoreUpdateFailure
				if len(received[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(received[unexpectedFailureTopic]))
				}
				unexpectedBatchTopic := sharedevents.LeaderboardBatchTagAssignmentRequested
				if len(received[unexpectedBatchTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedBatchTopic, len(received[unexpectedBatchTopic]))
				}
			},
			// Update to use the actual event topics defined in scoreevents package
			expectedOutgoingTopics: []string{scoreevents.ScoreUpdateSuccess},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Score record not found",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				return sharedtypes.RoundID(uuid.New()) // Return a RoundID that doesn't exist in the DB
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID, // This RoundID does not have a score record
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(-1),
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				// Based on the handler, we should expect a failure message
				failureTopic := scoreevents.ScoreUpdateFailure
				failureMsgs := received[failureTopic]
				if len(failureMsgs) != 1 {
					t.Fatalf("Expected 1 failure message on topic %q, got %d", failureTopic, len(failureMsgs))
				}
				validateScoreUpdateFailure(t, deps, incomingMsg, failureMsgs[0])

				// Verify no success message was sent
				successTopic := scoreevents.ScoreUpdateSuccess
				if len(received[successTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", successTopic, len(received[successTopic]))
				}
			},
			// Update to use the actual event topics defined in scoreevents package
			expectedOutgoingTopics: []string{scoreevents.ScoreUpdateFailure},
			expectHandlerError:     false, // The handler should handle this gracefully with a failure message
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Invalid Score Value",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				ctx := deps.Ctx

				roundID := sharedtypes.RoundID(uuid.New())
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(0)},
				}

				guildID := sharedtypes.GuildID("test-guild") // Use a test guild ID
				processResult, processErr := deps.ScoreModule.ScoreService.ProcessRoundScores(ctx, guildID, roundID, scores)
				if processErr != nil || processResult.Error != nil {
					t.Fatalf("Failed to process initial round scores for setup: %v, result: %+v", processErr, processResult.Error)
				}
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(99999), // Intentionally invalid score
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				// Based on the handler, we should expect a failure message for invalid score
				failureTopic := scoreevents.ScoreUpdateFailure
				failureMsgs := received[failureTopic]
				if len(failureMsgs) != 1 {
					t.Fatalf("Expected 1 failure message on topic %q, got %d", failureTopic, len(failureMsgs))
				}
				validateScoreUpdateFailure(t, deps, incomingMsg, failureMsgs[0])

				// Verify no success message was sent
				successTopic := scoreevents.ScoreUpdateSuccess
				if len(received[successTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", successTopic, len(received[successTopic]))
				}
			},
			// Update to use the actual event topics defined in scoreevents package
			expectedOutgoingTopics: []string{scoreevents.ScoreUpdateFailure},
			expectHandlerError:     false, // The handler should handle this gracefully with a failure message
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Invalid Payload Format",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				return sharedtypes.RoundID(uuid.New())
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				// Create a message with invalid JSON
				msg := message.NewMessage(uuid.New().String(), []byte("invalid JSON"))
				correlationID := uuid.New().String()
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
				msg.Metadata.Set("topic", scoreevents.ScoreUpdateRequest)

				if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), scoreevents.ScoreUpdateRequest, msg); err != nil {
					t.Fatalf("Failed to publish invalid message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				// No messages should be published for unmarshal errors
				for topic, msgs := range received {
					if len(msgs) > 0 {
						t.Errorf("Expected no messages on topic %q, but received %d", topic, len(msgs))
					}
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true, // We expect an error for unmarshal failure
			timeout:                5 * time.Second,
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable for use in closures
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestScoreHandler(t) // Setup per-test dependencies for this subtest
			users := tc.users                // Use the pre-generated users for this test case

			// Execute the custom setup function to get the initial state (roundID)
			initialRoundID := tc.setupFn(t, deps, users)

			// Construct the generic testutils.TestCase
			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					// This SetupFn for testutils.TestCase should just return the already setup initialState
					return initialRoundID
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					// Now we can use the pre-generated users directly
					return tc.publishMsgFn(t, deps, users, initialRoundID)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					// Pass the initialState (which is initialRoundID) to the custom validateFn
					tc.validateFn(t, deps, incoming, received, initialState.(sharedtypes.RoundID))
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout, // Pass the timeout
			}
			// Run the test using the testutils.RunTest helper
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
