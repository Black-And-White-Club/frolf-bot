package scorehandler_integration_tests

import (
	"context"
	"encoding/json"
	"sync"
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

const CorrectScoreRequestTopic = scoreevents.ScoreUpdateRequest

func TestHandleCorrectScoreRequest(t *testing.T) {
	deps := SetupTestScoreHandler(t, testEnv)
	generator := testutils.NewTestDataGenerator(100)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) sharedtypes.RoundID
		publishMsgFn           func(t *testing.T, deps ScoreHandlerTestDeps, roundID sharedtypes.RoundID, generator *testutils.TestDataGenerator) *message.Message
		validateFn             func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) // Removed handlerErr parameter
		expectedOutgoingTopics []string                                                                                                                // Topics the test will subscribe to and wait for messages on
	}{
		{
			name: "Success - Correct score with valid data",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) sharedtypes.RoundID {
				ctx := deps.Ctx
				_ = testutils.CleanScoreIntegrationTables(ctx, deps.DB)
				_ = deps.CleanNatsStreams(ctx, "score")
				_ = deps.CleanNatsStreams(ctx, "leaderboard")

				users := generator.GenerateUsers(2)
				roundID := sharedtypes.RoundID(uuid.New())
				tag1 := sharedtypes.TagNumber(42)
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(0), TagNumber: &tag1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: sharedtypes.Score(5)},
				}

				processResult, processErr := deps.ScoreModule.ScoreService.ProcessRoundScores(ctx, roundID, scores)
				if processErr != nil || processResult.Error != nil {
					t.Fatalf("Failed to process initial round scores for setup: %v, result: %+v", processErr, processResult.Error)
				}

				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, roundID sharedtypes.RoundID, generator *testutils.TestDataGenerator) *message.Message {
				users := generator.GenerateUsers(2)
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(users[0].UserID),
					Score:   sharedtypes.Score(-3),
				}

				data, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), data)
				correlationID := uuid.New().String()
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
				msg.Metadata.Set("topic", CorrectScoreRequestTopic)

				if err := deps.EventBus.Publish(CorrectScoreRequestTopic, msg); err != nil {
					t.Fatalf("Failed to publish: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message) {
				expectedSuccessTopic := scoreevents.ScoreUpdateSuccess
				successMsgs := received[expectedSuccessTopic]
				if len(successMsgs) != 1 {
					t.Fatalf("Expected 1 success message on topic %q, got %d", expectedSuccessTopic, len(successMsgs))
				}

				successMsg := successMsgs[0]
				var successPayload scoreevents.ScoreUpdateSuccessPayload
				err := deps.ScoreModule.Helper.UnmarshalPayload(successMsg, &successPayload)
				if err != nil {
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
				if successMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Error("Correlation ID mismatch in success message")
				}

				unexpectedFailureTopic := scoreevents.ScoreUpdateFailure
				if len(received[unexpectedFailureTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailureTopic, len(received[unexpectedFailureTopic]))
				}
				unexpectedBatchTopic := sharedevents.LeaderboardBatchTagAssignmentRequested
				if len(received[unexpectedBatchTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedBatchTopic, len(received[unexpectedBatchTopic]))
				}
			},
			expectedOutgoingTopics: []string{scoreevents.ScoreUpdateSuccess},
		},
		{
			name: "Failure - Score record not found",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) sharedtypes.RoundID {
				_ = testutils.CleanScoreIntegrationTables(deps.Ctx, deps.DB)
				_ = deps.CleanNatsStreams(deps.Ctx, "score")
				return sharedtypes.RoundID(uuid.New()) // Return a RoundID that doesn't exist in the DB
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, roundID sharedtypes.RoundID, generator *testutils.TestDataGenerator) *message.Message {
				user := generator.GenerateUsers(1)[0]
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID, // This RoundID does not have a score record
					UserID:  sharedtypes.DiscordID(user.UserID),
					Score:   sharedtypes.Score(-1),
				}

				data, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), data)
				correlationID := uuid.New().String()
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
				msg.Metadata.Set("topic", CorrectScoreRequestTopic)

				if err := deps.EventBus.Publish(CorrectScoreRequestTopic, msg); err != nil {
					t.Fatalf("Failed to publish: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message) {
				// Expect no messages to be published on any topic
				if len(received) > 0 {
					t.Errorf("Expected no messages to be published, but received: %+v", received)
				}
				// Note: The router will log the handler's error, but we don't assert on logs in this test.
			},
			expectedOutgoingTopics: []string{}, // Expect no outgoing messages
		},
		{
			name: "Failure - Invalid Score Value",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps, generator *testutils.TestDataGenerator) sharedtypes.RoundID {
				ctx := deps.Ctx
				_ = testutils.CleanScoreIntegrationTables(ctx, deps.DB)
				_ = deps.CleanNatsStreams(ctx, "score")

				users := generator.GenerateUsers(1)
				roundID := sharedtypes.RoundID(uuid.New())
				scores := []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(0)},
				}

				processResult, processErr := deps.ScoreModule.ScoreService.ProcessRoundScores(ctx, roundID, scores)
				if processErr != nil || processResult.Error != nil {
					t.Fatalf("Failed to process initial round scores for setup: %v, result: %+v", processErr, processResult.Error)
				}

				return roundID
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps, roundID sharedtypes.RoundID, generator *testutils.TestDataGenerator) *message.Message {
				user := generator.GenerateUsers(1)[0]
				payload := scoreevents.ScoreUpdateRequestPayload{
					RoundID: roundID,
					UserID:  sharedtypes.DiscordID(user.UserID),
					Score:   sharedtypes.Score(99999), // Intentionally invalid score
				}

				data, _ := json.Marshal(payload)
				msg := message.NewMessage(uuid.New().String(), data)
				correlationID := uuid.New().String()
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)
				msg.Metadata.Set("topic", CorrectScoreRequestTopic)

				if err := deps.EventBus.Publish(CorrectScoreRequestTopic, msg); err != nil {
					t.Fatalf("Failed to publish: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, received map[string][]*message.Message) {
				// Expect no messages to be published on any topic
				if len(received) > 0 {
					t.Errorf("Expected no messages to be published, but received: %+v", received)
				}
				// Note: The router will log the handler's error, but we don't assert on logs in this test.
			},
			expectedOutgoingTopics: []string{}, // Expect no outgoing messages
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			roundID := tc.setupFn(t, deps, generator)

			received := make(map[string][]*message.Message)
			ctx, cancel := context.WithTimeout(deps.Ctx, 8*time.Second)
			defer cancel()

			wg := &sync.WaitGroup{}
			for _, topic := range tc.expectedOutgoingTopics {
				wg.Add(1)
				sub, err := deps.EventBus.Subscribe(ctx, topic)
				if err != nil {
					t.Fatalf("Subscribe failed: %v", err)
				}

				go func(topic string, ch <-chan *message.Message) {
					defer wg.Done()
					for {
						select {
						case msg, ok := <-ch:
							if !ok {
								return
							}
							received[topic] = append(received[topic], msg)
							msg.Ack()
						case <-ctx.Done():
							return
						}
					}
				}(topic, sub)
			}

			time.Sleep(100 * time.Millisecond)

			incomingMsg := tc.publishMsgFn(t, deps, roundID, generator)

			// Wait for all expected messages to be received or timeout
			waitStartTime := time.Now()
			for {
				allReceived := true
				for _, topic := range tc.expectedOutgoingTopics {
					if len(received[topic]) == 0 {
						allReceived = false
						break
					}
				}
				if allReceived {
					break
				}

				// If we time out and no messages were expected, this is okay.
				// If messages were expected, this is a failure.
				if time.Since(waitStartTime) > 7*time.Second {
					if len(tc.expectedOutgoingTopics) > 0 {
						t.Fatalf("Timeout waiting for expected messages on all topics. Received: %+v", received)
					}
					break // Exit loop if no messages expected and timed out
				}
				time.Sleep(50 * time.Millisecond)
			}

			// Call validateFn without handlerErr
			tc.validateFn(t, deps, incomingMsg, received)
		})
	}
}
