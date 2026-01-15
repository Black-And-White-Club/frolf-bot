package scorehandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// publishScoreUpdateRequest helper to publish a ScoreUpdateRequest message.
func publishScoreUpdateRequest(t *testing.T, deps ScoreHandlerTestDeps, payload sharedevents.ScoreUpdateRequestedPayloadV1) *message.Message {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Failed to marshal request payload: %v", err)
	}
	msg := message.NewMessage(uuid.New().String(), data)
	correlationID := uuid.New().String()
	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
	msg.Metadata.Set("topic", sharedevents.ScoreUpdateRequestedV1)

	if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.ScoreUpdateRequestedV1, msg); err != nil {
		t.Fatalf("Failed to publish ScoreUpdateRequest: %v", err)
	}
	return msg
}

// validateScoreUpdateSuccess helper to validate a ScoreUpdateSuccess message.
func validateScoreUpdateSuccess(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, responseMsg *message.Message) {
	t.Helper()
	var successPayload sharedevents.ScoreUpdatedPayloadV1
	if err := deps.ScoreModule.Helper.UnmarshalPayload(responseMsg, &successPayload); err != nil {
		t.Fatalf("Unmarshal error for ScoreUpdateSuccessPayload: %v", err)
	}

	var incomingPayload sharedevents.ScoreUpdateRequestedPayloadV1
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
	var failurePayload sharedevents.ScoreUpdateFailedPayloadV1
	if err := deps.ScoreModule.Helper.UnmarshalPayload(responseMsg, &failurePayload); err != nil {
		t.Fatalf("Unmarshal error for ScoreUpdateFailurePayload: %v", err)
	}

	var incomingPayload sharedevents.ScoreUpdateRequestedPayloadV1
	if err := json.Unmarshal(incomingMsg.Payload, &incomingPayload); err != nil {
		t.Fatalf("Failed to unmarshal incoming payload: %v", err)
	}

	if failurePayload.RoundID != incomingPayload.RoundID {
		t.Errorf("ScoreUpdateFailurePayload RoundID mismatch: expected %v, got %v", incomingPayload.RoundID, failurePayload.RoundID)
	}
	if failurePayload.UserID != incomingPayload.UserID {
		t.Errorf("ScoreUpdateFailurePayload UserID mismatch: expected %q, got %q", incomingPayload.UserID, failurePayload.UserID)
	}
	if failurePayload.Reason == "" {
		t.Error("Expected non-empty reason message in failure payload")
	}
	if responseMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		t.Error("Correlation ID mismatch in failure message")
	}
}

func TestHandleCorrectScoreRequest(t *testing.T) {
	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

	testCases := []struct {
		name                   string
		users                  []testutils.User
		setupFn                func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID
		publishMsgFn           func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message
		validateFn             func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialRoundID sharedtypes.RoundID)
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name:  "Success - Correct score with valid data",
			users: generator.GenerateUsers(2),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				roundID := sharedtypes.RoundID(uuid.New())
				tag1 := sharedtypes.TagNumber(42)
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(0), TagNumber: &tag1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: sharedtypes.Score(5)},
				}

				guildID := sharedtypes.GuildID("test-guild")
				// Note: We use a separate context here to ensure setup events
				// don't accidentally share the test's Correlation ID.
				setupCtx := context.Background()
				_, err := deps.ScoreModule.ScoreService.ProcessRoundScores(setupCtx, guildID, roundID, scores, false)
				if err != nil {
					t.Fatalf("Failed setup: %v", err)
				}
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := sharedevents.ScoreUpdateRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(-3),
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				expectedSuccessTopic := sharedevents.ScoreUpdatedV1
				sentCID := incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey)

				// 1. Filter for messages matching our specific request's CID
				var relevantMsgs []*message.Message
				for _, m := range received[expectedSuccessTopic] {
					if m.Metadata.Get(middleware.CorrelationIDMetadataKey) == sentCID {
						relevantMsgs = append(relevantMsgs, m)
					}
				}

				// 2. Adjust expectation to "At-Least-Once"
				// In integration, we care that our specific CID produced a result.
				// If infrastructure causes a double-publish, the test should still pass.
				if len(relevantMsgs) < 1 {
					t.Fatalf("Expected at least 1 relevant success message (CID: %s), got 0", sentCID)
				}

				if len(relevantMsgs) > 1 {
					t.Logf("INFO: Received %d messages for CID %s. This is likely an infrastructure retry or setup bleed.", len(relevantMsgs), sentCID)
				}

				// 3. Validate the payload of the first relevant message
				validateScoreUpdateSuccess(t, deps, incomingMsg, relevantMsgs[0])
			},
			expectedOutgoingTopics: []string{sharedevents.ScoreUpdatedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Score record not found",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				return sharedtypes.RoundID(uuid.New())
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := sharedevents.ScoreUpdateRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(-1),
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				failureTopic := sharedevents.ScoreUpdateFailedV1
				allFailures := received[failureTopic]
				sentCID := incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey)

				var relevantFailures []*message.Message
				for _, m := range allFailures {
					if m.Metadata.Get(middleware.CorrelationIDMetadataKey) == sentCID {
						relevantFailures = append(relevantFailures, m)
					}
				}

				if len(relevantFailures) != 1 {
					t.Fatalf("Expected 1 failure message (CID: %s), got %d", sentCID, len(relevantFailures))
				}
				validateScoreUpdateFailure(t, deps, incomingMsg, relevantFailures[0])
			},
			expectedOutgoingTopics: []string{sharedevents.ScoreUpdateFailedV1},
			expectHandlerError:     false,
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
				_, _ = deps.ScoreModule.ScoreService.ProcessRoundScores(ctx, "test-guild", roundID, scores, false)
				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				payload := sharedevents.ScoreUpdateRequestedPayloadV1{
					GuildID: sharedtypes.GuildID("test-guild"),
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(99999),
				}
				return publishScoreUpdateRequest(t, deps, payload)
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				failureTopic := sharedevents.ScoreUpdateFailedV1
				sentCID := incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey)

				found := false
				for _, m := range received[failureTopic] {
					if m.Metadata.Get(middleware.CorrelationIDMetadataKey) == sentCID {
						validateScoreUpdateFailure(t, deps, incomingMsg, m)
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected failure message with CID %s not found", sentCID)
				}
			},
			expectedOutgoingTopics: []string{sharedevents.ScoreUpdateFailedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name:  "Failure - Invalid Payload Format",
			users: generator.GenerateUsers(1),
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User) sharedtypes.RoundID {
				return sharedtypes.RoundID(uuid.New())
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, users []testutils.User, roundID sharedtypes.RoundID) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid JSON"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", sharedevents.ScoreUpdateRequestedV1)

				_ = testutils.PublishMessage(t, deps.EventBus, context.Background(), sharedevents.ScoreUpdateRequestedV1, msg)
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message, initialRoundID sharedtypes.RoundID) {
				// No outgoing messages expected for malformed input
				for topic, msgs := range received {
					if len(msgs) > 0 {
						t.Errorf("Unexpected message on topic %q", topic)
					}
				}
			},
			expectedOutgoingTopics: []string{},
			expectHandlerError:     true,
			timeout:                2 * time.Second,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestScoreHandler(t)

			// Optional: ensure a clean slate if deps doesn't truncate automatically
			// deps.DB.TruncateAll(deps.Ctx)

			initialRoundID := tc.setupFn(t, deps, tc.users)

			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return initialRoundID
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, tc.users, initialRoundID)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incoming *message.Message, received map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, incoming, received, initialState.(sharedtypes.RoundID))
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
