package scorehandler_integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
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

func TestHandleProcessRoundScoresRequest(t *testing.T) {
	deps := SetupTestScoreHandler(t, testEnv)

	generator := testutils.NewTestDataGenerator(42)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps ScoreHandlerTestDeps)
		publishMsgFn           func(t *testing.T, deps ScoreHandlerTestDeps) *message.Message
		validateFn             func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - Process Valid Round Scores",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps) {
				if err := testutils.CleanScoreIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "score"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
				// Also clean the leaderboard stream since the output goes there
				if err := deps.CleanNatsStreams(deps.Ctx, "leaderboard"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps) *message.Message {
				users := generator.GenerateUsers(5)
				var scores []sharedtypes.ScoreInfo
				discGolfScores := []float64{0, -2, +3, -4, +1}
				tag1 := sharedtypes.TagNumber(42)
				tag3 := sharedtypes.TagNumber(17)

				scores = append(scores, []sharedtypes.ScoreInfo{
					{UserID: sharedtypes.DiscordID(users[0].UserID), Score: sharedtypes.Score(discGolfScores[0]), TagNumber: &tag1},
					{UserID: sharedtypes.DiscordID(users[1].UserID), Score: sharedtypes.Score(discGolfScores[1])},
					{UserID: sharedtypes.DiscordID(users[2].UserID), Score: sharedtypes.Score(discGolfScores[2]), TagNumber: &tag3},
					{UserID: sharedtypes.DiscordID(users[3].UserID), Score: sharedtypes.Score(discGolfScores[3])},
					{UserID: sharedtypes.DiscordID(users[4].UserID), Score: sharedtypes.Score(discGolfScores[4])},
				}...)

				roundID := sharedtypes.RoundID(uuid.New())
				payload := scoreevents.ProcessRoundScoresRequestPayload{RoundID: roundID, Scores: scores}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}

				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				inputTopic := scoreevents.ProcessRoundScoresRequest
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				// Removed debug log: log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				var requestPayload scoreevents.ProcessRoundScoresRequestPayload
				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
					t.Fatalf("Failed to unmarshal request payload: %v", err)
				}

				expectedTopic := sharedevents.LeaderboardBatchTagAssignmentRequested

				timeout := time.After(10 * time.Second) // Reduced timeout slightly, relying on better setup timing
				checkInterval := 100 * time.Millisecond
				var receivedMsg *message.Message
				found := false

				// Wait for the expected message to appear in the receivedMsgs map
				for !found {
					select {
					case <-timeout:
						t.Fatalf("Timeout waiting for message on topic %q. Received messages: %+v", expectedTopic, receivedMsgs)
					case <-time.After(checkInterval):
						msgs := receivedMsgs[expectedTopic]
						if len(msgs) > 0 {
							receivedMsg = msgs[0]
							found = true
						}
					}
				}

				var batchPayload sharedevents.BatchTagAssignmentRequestedPayload
				if err := deps.ScoreModule.Helper.UnmarshalPayload(receivedMsg, &batchPayload); err != nil {
					t.Fatalf("Failed to unmarshal BatchTagAssignmentRequestedPayload: %v", err)
				}

				expectedBatchIDPrefix := requestPayload.RoundID.String() + "-"
				if !strings.HasPrefix(batchPayload.BatchID, expectedBatchIDPrefix) {
					t.Errorf("BatchID format mismatch: expected prefix %q, got %q",
						expectedBatchIDPrefix, batchPayload.BatchID)
				}

				if len(batchPayload.Assignments) != 2 {
					t.Errorf("Expected 2 tag assignments, got %d", len(batchPayload.Assignments))
				}

				// Create a map of expected UserID to TagNumber from the incoming message payload
				expectedAssignments := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
				for _, scoreInfo := range requestPayload.Scores {
					if scoreInfo.TagNumber != nil {
						expectedAssignments[scoreInfo.UserID] = *scoreInfo.TagNumber
					}
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
					// Remove from expected map to ensure all expected assignments were received
					delete(expectedAssignments, receivedAssignment.UserID)
				}

				// Check if any expected assignments were not received
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
			},
			expectedOutgoingTopics: []string{sharedevents.LeaderboardBatchTagAssignmentRequested},
		},
		{
			name: "Failure - Empty Scores List",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps) {
				if err := testutils.CleanScoreIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "score"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps) *message.Message {
				roundID := sharedtypes.RoundID(uuid.New())
				payload := scoreevents.ProcessRoundScoresRequestPayload{RoundID: roundID, Scores: []sharedtypes.ScoreInfo{}}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}

				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", scoreevents.ProcessRoundScoresRequest)

				inputTopic := scoreevents.ProcessRoundScoresRequest
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				var requestPayload scoreevents.ProcessRoundScoresRequestPayload
				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
					t.Fatalf("Failed to unmarshal request payload: %v", err)
				}

				expectedTopic := scoreevents.ProcessRoundScoresFailure
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var failurePayload scoreevents.ProcessRoundScoresFailurePayload
				if err := deps.ScoreModule.Helper.UnmarshalPayload(receivedMsg, &failurePayload); err != nil {
					t.Fatalf("Failed to unmarshal ProcessRoundScoresFailurePayload: %v", err)
				}

				if failurePayload.RoundID != requestPayload.RoundID {
					t.Errorf("ProcessRoundScoresFailurePayload RoundID mismatch: expected %v, got %v",
						requestPayload.RoundID, failurePayload.RoundID)
				}

				if failurePayload.Error == "" {
					t.Errorf("Expected non-empty error message in failure payload")
				}
				if !strings.Contains(strings.ToLower(failurePayload.Error), "empty") &&
					!strings.Contains(strings.ToLower(failurePayload.Error), "no scores") {
					t.Errorf("Expected error message to mention empty scores, got: %s", failurePayload.Error)
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
			},
			expectedOutgoingTopics: []string{scoreevents.ProcessRoundScoresFailure},
		},
		{
			name: "Failure - Invalid Message Payload",
			setupFn: func(t *testing.T, deps ScoreHandlerTestDeps) {
				if err := testutils.CleanScoreIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "score"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps ScoreHandlerTestDeps) *message.Message {
				msg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", scoreevents.ProcessRoundScoresRequest)

				inputTopic := scoreevents.ProcessRoundScoresRequest
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps ScoreHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				unexpectedTopic := sharedevents.LeaderboardBatchTagAssignmentRequested
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}

				failureTopic := scoreevents.ProcessRoundScoresFailure
				if len(receivedMsgs[failureTopic]) > 0 {
					receivedMsg := receivedMsgs[failureTopic][0]
					var failurePayload scoreevents.ProcessRoundScoresFailurePayload
					if err := deps.ScoreModule.Helper.UnmarshalPayload(receivedMsg, &failurePayload); err != nil {
						t.Fatalf("Failed to unmarshal ProcessRoundScoresFailurePayload: %v", err)
					}

					if failurePayload.Error == "" {
						t.Errorf("Expected non-empty error message in failure payload")
					}

					if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
						t.Errorf("Correlation ID mismatch: expected %q, got %q",
							incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
							receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
					}
				}
			},
			expectedOutgoingTopics: []string{scoreevents.ProcessRoundScoresFailure},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFn(t, deps)
			receivedMsgs := make(map[string][]*message.Message)
			// Increased timeout to 10 seconds
			subCtx, cancelSub := context.WithTimeout(deps.Ctx, 10*time.Second)
			defer cancelSub()

			for _, topic := range tc.expectedOutgoingTopics {
				consumerName := fmt.Sprintf("test-consumer-%s-%s", sanitizeTopicForNATS(topic), uuid.New().String()[:8])
				messageChannel, err := deps.EventBus.Subscribe(subCtx, topic)
				if err != nil {
					t.Fatalf("Failed to subscribe to topic %q: %v", topic, err)
				}

				log.Printf("Test Subscribed to topic %q with consumer %q", topic, consumerName)

				go func(topic string, messages <-chan *message.Message) {
					for msg := range messages {
						log.Printf("Test Received message %s on topic %q", msg.UUID, topic)
						receivedMsgs[topic] = append(receivedMsgs[topic], msg)
						msg.Ack()
					}
				}(topic, messageChannel)
			}

			incomingMsg := tc.publishMsgFn(t, deps)
			time.Sleep(500 * time.Millisecond) // Give router time to pick up message

			tc.validateFn(t, deps, incomingMsg, receivedMsgs)
		})
	}
}

func sanitizeTopicForNATS(topic string) string {
	sanitized := strings.ReplaceAll(topic, ".", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	sanitized = reg.ReplaceAllString(sanitized, "")
	sanitized = strings.TrimPrefix(sanitized, "-")
	sanitized = strings.TrimSuffix(sanitized, "-")
	return sanitized
}
