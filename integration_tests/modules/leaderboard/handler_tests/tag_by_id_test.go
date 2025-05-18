package leaderboardhandler_integration_tests

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"strings"
// 	"sync"
// 	"testing"
// 	"time"

// 	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// 	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
// 	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
// 	"github.com/ThreeDotsLabs/watermill/message"
// 	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
// 	"github.com/google/uuid"
// )

// // TestHandleGetTagByUserIDRequest is an integration test for the HandleGetTagByUserIDRequest handler.
// func TestHandleGetTagByUserIDRequest(t *testing.T) {
// 	deps := SetupTestLeaderboardHandler(t, testEnv)
// 	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())

// 	type TestCase struct {
// 		name string
// 		// setupFn prepares the database and NATS streams for the test case.
// 		setupFn func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
// 		// publishMsgFn creates and publishes the input message for the handler.
// 		publishMsgFn func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
// 		// validateFn asserts the outcome of the handler execution, including published messages and database state.
// 		validateFn func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard, receivedMsgsMutex *sync.Mutex, tc TestCase)
// 		// expectedOutgoingTopics lists the topics the test expects messages to be published on.
// 		expectedOutgoingTopics []string
// 		// expectHandlerError indicates if the handler function itself is expected to return an error to Watermill.
// 		expectHandlerError bool
// 		// timeout allows specifying a custom timeout for specific test cases.
// 		timeout time.Duration
// 	}

// 	tests := []TestCase{
// 		{
// 			name: "Success - Tag found for user on active leaderboard",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				entries := []leaderboardtypes.LeaderboardEntry{
// 					{UserID: "user_with_tag", TagNumber: tagPtr(42)},
// 					{UserID: "another_user_1", TagNumber: tagPtr(1)},
// 					{UserID: "another_user_2", TagNumber: tagPtr(23)},
// 				}
// 				return setupLeaderboardWithEntries(t, deps, entries, true, sharedtypes.RoundID(uuid.New()), "setup_user")
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := sharedevents.DiscordTagLookupRequestPayload{
// 					UserID: "user_with_tag",
// 				}
// 				return publishTagNumberRequest(t, deps, payload, uuid.New().String(), uuid.New().String())
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard, receivedMsgsMutex *sync.Mutex, tc TestCase) {
// 				expectedTopic := sharedevents.DiscordTagLookupByUserIDSuccess
// 				unexpectedTopic := sharedevents.DiscordTagLoopupByUserIDNotFound
// 				unexpectedFailTopic := sharedevents.DiscordTagLookupByUserIDFailed

// 				msgs := waitForMessages(t, receivedMsgs, receivedMsgsMutex, expectedTopic, 1, tc.timeout)
// 				if len(msgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none. Received map: %+v", expectedTopic, receivedMsgs)
// 				}
// 				if len(msgs) > 1 {
// 					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
// 				}

// 				receivedMsgsMutex.Lock()
// 				if len(receivedMsgs[unexpectedTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
// 				}
// 				if len(receivedMsgs[unexpectedFailTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailTopic, len(receivedMsgs[unexpectedFailTopic]))
// 				}
// 				receivedMsgsMutex.Unlock()

// 				receivedMsg := msgs[0]
// 				var successPayload sharedevents.DiscordTagLookupResultPayload
// 				if err := deps.LeaderboardModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
// 					t.Fatalf("Failed to unmarshal DiscordTagLookupResultPayload: %v", err)
// 				}

// 				var requestPayload sharedevents.DiscordTagLookupRequestPayload
// 				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
// 					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
// 				}

// 				if successPayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, successPayload.UserID)
// 				}
// 				if !successPayload.Found {
// 					t.Error("Expected Found to be true in success payload")
// 				}
// 				if successPayload.TagNumber == nil {
// 					t.Error("Expected TagNumber to be non-nil in success payload")
// 				} else if *successPayload.TagNumber != 42 {
// 					t.Errorf("TagNumber mismatch: expected %d, got %d", 42, *successPayload.TagNumber)
// 				}

// 				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 					t.Errorf("Correlation ID mismatch: expected %q, got %q",
// 						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
// 						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
// 				}

// 				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
// 			},
// 			expectedOutgoingTopics: []string{sharedevents.DiscordTagLookupByUserIDSuccess},
// 			expectHandlerError:     false,
// 			timeout:                5 * time.Second,
// 		},
// 		{
// 			name: "Success - User not found on active leaderboard",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				entries := []leaderboardtypes.LeaderboardEntry{
// 					{UserID: "another_user_1", TagNumber: tagPtr(1)},
// 					{UserID: "another_user_2", TagNumber: tagPtr(2)},
// 				}
// 				return setupLeaderboardWithEntries(t, deps, entries, true, sharedtypes.RoundID(uuid.New()), "setup_user")
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := sharedevents.DiscordTagLookupRequestPayload{
// 					UserID: "user_not_on_leaderboard",
// 				}
// 				return publishTagNumberRequest(t, deps, payload, uuid.New().String(), uuid.New().String())
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard, receivedMsgsMutex *sync.Mutex, tc TestCase) {
// 				expectedTopic := sharedevents.DiscordTagLoopupByUserIDNotFound
// 				unexpectedTopic := sharedevents.DiscordTagLookupByUserIDSuccess
// 				unexpectedFailTopic := sharedevents.DiscordTagLookupByUserIDFailed

// 				msgs := waitForMessages(t, receivedMsgs, receivedMsgsMutex, expectedTopic, 1, tc.timeout)
// 				if len(msgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none. Received map: %+v", expectedTopic, receivedMsgs)
// 				}
// 				if len(msgs) > 1 {
// 					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
// 				}

// 				receivedMsgsMutex.Lock()
// 				if len(receivedMsgs[unexpectedTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
// 				}
// 				if len(receivedMsgs[unexpectedFailTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedFailTopic, len(receivedMsgs[unexpectedFailTopic]))
// 				}
// 				receivedMsgsMutex.Unlock()

// 				receivedMsg := msgs[0]
// 				var notFoundPayload sharedevents.DiscordTagLookupResultPayload
// 				if err := deps.LeaderboardModule.Helper.UnmarshalPayload(receivedMsg, &notFoundPayload); err != nil {
// 					t.Fatalf("Failed to unmarshal DiscordTagLookupResultPayload: %v", err)
// 				}

// 				var requestPayload sharedevents.DiscordTagLookupRequestPayload
// 				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
// 					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
// 				}

// 				if notFoundPayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, notFoundPayload.UserID)
// 				}
// 				if notFoundPayload.Found {
// 					t.Error("Expected Found to be false in not-found payload")
// 				}
// 				if notFoundPayload.TagNumber != nil {
// 					t.Errorf("Expected nil TagNumber in not-found payload, got %v", notFoundPayload.TagNumber)
// 				}

// 				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 					t.Errorf("Correlation ID mismatch: expected %q, got %q",
// 						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
// 						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
// 				}

// 				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
// 			},
// 			expectedOutgoingTopics: []string{sharedevents.DiscordTagLoopupByUserIDNotFound},
// 			expectHandlerError:     false,
// 			timeout:                5 * time.Second,
// 		},
// 		{
// 			name: "Failure - No active leaderboard exists",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				return setupLeaderboardWithEntries(t, deps, []leaderboardtypes.LeaderboardEntry{}, false, sharedtypes.RoundID(uuid.New()), "setup_user")
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				payload := sharedevents.DiscordTagLookupRequestPayload{
// 					UserID: "any_user",
// 				}
// 				msg := publishTagNumberRequest(t, deps, payload, uuid.New().String(), uuid.New().String())
// 				time.Sleep(100 * time.Millisecond)
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard, receivedMsgsMutex *sync.Mutex, tc TestCase) {
// 				expectedTopic := sharedevents.DiscordTagLookupByUserIDFailed
// 				unexpectedTopic1 := sharedevents.DiscordTagLookupByUserIDSuccess
// 				unexpectedTopic2 := sharedevents.DiscordTagLoopupByUserIDNotFound

// 				msgs := waitForMessages(t, receivedMsgs, receivedMsgsMutex, expectedTopic, 1, tc.timeout)
// 				if len(msgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none. Received map: %+v", expectedTopic, receivedMsgs)
// 				}
// 				if len(msgs) > 1 {
// 					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
// 				}

// 				receivedMsgsMutex.Lock()
// 				if len(receivedMsgs[unexpectedTopic1]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic1, len(receivedMsgs[unexpectedTopic1]))
// 				}
// 				if len(receivedMsgs[unexpectedTopic2]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic2, len(receivedMsgs[unexpectedTopic2]))
// 				}
// 				receivedMsgsMutex.Unlock()

// 				receivedFailedMsg := msgs[0]
// 				var failurePayload sharedevents.DiscordTagLookupByUserIDFailedPayload
// 				if err := deps.LeaderboardModule.Helper.UnmarshalPayload(receivedFailedMsg, &failurePayload); err != nil {
// 					t.Fatalf("Failed to unmarshal DiscordTagLookupByUserIDFailedPayload: %v", err)
// 				}

// 				var requestPayload sharedevents.DiscordTagLookupRequestPayload
// 				if err := json.Unmarshal(incomingMsg.Payload, &requestPayload); err != nil {
// 					t.Fatalf("Failed to unmarshal incoming request payload: %v", err)
// 				}

// 				if failurePayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch: expected %q, got %q", requestPayload.UserID, failurePayload.UserID)
// 				}

// 				expectedReasonSubstring := "No active leaderboard found"
// 				if !strings.Contains(failurePayload.Reason, expectedReasonSubstring) {
// 					t.Errorf("Failure reason mismatch: expected reason to contain %q, got %q",
// 						expectedReasonSubstring, failurePayload.Reason)
// 				}

// 				if receivedFailedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 					t.Errorf("Correlation ID mismatch: expected %q, got %q",
// 						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
// 						receivedFailedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
// 				}

// 				assertLeaderboardState(t, deps, initialLeaderboard, 0, false)
// 			},
// 			expectedOutgoingTopics: []string{sharedevents.DiscordTagLookupByUserIDFailed},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Failure - Invalid incoming message payload (Unmarshal error)",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				entries := []leaderboardtypes.LeaderboardEntry{
// 					{UserID: "some_user", TagNumber: tagPtr(99)},
// 				}
// 				return setupLeaderboardWithEntries(t, deps, entries, true, sharedtypes.RoundID(uuid.New()), "setup_user")
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				msg := message.NewMessage(uuid.New().String(), []byte("invalid json payload"))
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

// 				inputTopic := sharedevents.DiscordTagLookUpByUserIDRequest
// 				err := deps.EventBus.Publish(inputTopic, msg)
// 				if err != nil {
// 					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
// 				}
// 				log.Printf("Test Case %q Publish: Published message %s to topic %q", t.Name(), msg.UUID, inputTopic)
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard, receivedMsgsMutex *sync.Mutex, tc TestCase) {
// 				unexpectedTopic1 := sharedevents.DiscordTagLookupByUserIDSuccess
// 				unexpectedTopic2 := sharedevents.DiscordTagLoopupByUserIDNotFound
// 				unexpectedTopic3 := sharedevents.DiscordTagLookupByUserIDFailed

// 				time.Sleep(tc.timeout)

// 				receivedMsgsMutex.Lock()
// 				if len(receivedMsgs[unexpectedTopic1]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic1, len(receivedMsgs[unexpectedTopic1]))
// 				}
// 				if len(receivedMsgs[unexpectedTopic2]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic2, len(receivedMsgs[unexpectedTopic2]))
// 				}
// 				if len(receivedMsgs[unexpectedTopic3]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic3, len(receivedMsgs[unexpectedTopic3]))
// 				}
// 				receivedMsgsMutex.Unlock()

// 				assertLeaderboardState(t, deps, initialLeaderboard, 1, true)
// 			},
// 			expectedOutgoingTopics: []string{},
// 			expectHandlerError:     true,
// 			timeout:                2 * time.Second,
// 		},
// 	}

// 	for _, tc := range tests {
// 		tc := tc
// 		t.Run(tc.name, func(t *testing.T) {
// 			localReceivedMsgs := make(map[string][]*message.Message)
// 			localReceivedMutex := &sync.Mutex{}
// 			subscriberWg := &sync.WaitGroup{}

// 			initialLeaderboard := tc.setupFn(t, deps, generator)

// 			subCtx, cancelSub := context.WithCancel(deps.Ctx)
// 			t.Cleanup(func() {
// 				log.Printf("Test Case %q: Canceling subscription context.", t.Name())
// 				cancelSub()
// 				log.Printf("Test Case %q: Waiting for subscriber goroutines to finish.", t.Name())
// 				waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
// 				defer waitCancel()
// 				waitCh := make(chan struct{})
// 				go func() {
// 					subscriberWg.Wait()
// 					close(waitCh)
// 				}()
// 				select {
// 				case <-waitCh:
// 					log.Printf("Test Case %q: Subscriber goroutines finished.", t.Name())
// 				case <-waitCtx.Done():
// 					log.Printf("Test Case %q: WARNING: Subscriber goroutines did not finish within timeout after context cancellation.", t.Name())
// 				}
// 			})

// 			for _, topic := range tc.expectedOutgoingTopics {
// 				consumerName := fmt.Sprintf("test-consumer-%s-%s-%s", sanitizeForNATS(topic), sanitizeForNATS(tc.name), uuid.New().String()[:8])
// 				messageChannel, err := deps.EventBus.Subscribe(subCtx, topic)
// 				if err != nil {
// 					t.Fatalf("Failed to subscribe to topic %q with consumer %q: %v", topic, consumerName, err)
// 				}

// 				log.Printf("Test Case %q: Subscribed to topic %q with consumer %q", tc.name, topic, consumerName)

// 				subscriberWg.Add(1)
// 				go func(topic string, messages <-chan *message.Message, receivedMsgs map[string][]*message.Message, mutex *sync.Mutex, wg *sync.WaitGroup, ctx context.Context) {
// 					defer wg.Done()
// 					log.Printf("Test Case %q Subscriber: Started goroutine for topic %q", tc.name, topic)
// 					for {
// 						select {
// 						case msg, ok := <-messages:
// 							if !ok {
// 								log.Printf("Test Case %q Subscriber: Channel closed for topic %q. Exiting goroutine.", tc.name, topic)
// 								return
// 							}
// 							log.Printf("Test Case %q Subscriber: Received message %s on topic %q", tc.name, msg.UUID, topic)
// 							mutex.Lock()
// 							receivedMsgs[topic] = append(receivedMsgs[topic], msg)
// 							mutex.Unlock()
// 							msg.Ack()
// 							log.Printf("Test Case %q Subscriber: Acknowledged message %s on topic %q", tc.name, msg.UUID, topic)
// 						case <-ctx.Done():
// 							log.Printf("Test Case %q Subscriber: Context canceled for topic %q. Exiting goroutine.", tc.name, topic)
// 							return
// 						}
// 					}
// 				}(topic, messageChannel, localReceivedMsgs, localReceivedMutex, subscriberWg, subCtx)
// 			}

// 			incomingMsg := tc.publishMsgFn(t, deps, generator)
// 			tc.validateFn(t, deps, incomingMsg, localReceivedMsgs, initialLeaderboard, localReceivedMutex, tc)
// 		})
// 	}
// }

// // publishTagNumberRequest marshals and publishes a TagNumberRequestPayload.
// func publishTagNumberRequest(t *testing.T, deps LeaderboardHandlerTestDeps, payload interface{}, msgUUID string, correlationID string) *message.Message {
// 	t.Helper()
// 	payloadBytes, err := json.Marshal(payload)
// 	if err != nil {
// 		t.Fatalf("Failed to marshal payload: %v", err)
// 	}

// 	msg := message.NewMessage(msgUUID, payloadBytes)
// 	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, correlationID)

// 	inputTopic := sharedevents.DiscordTagLookUpByUserIDRequest
// 	err = deps.EventBus.Publish(inputTopic, msg)
// 	if err != nil {
// 		t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
// 	}
// 	log.Printf("Publish: Published message %s to topic %q", msg.UUID, inputTopic)
// 	return msg
// }
