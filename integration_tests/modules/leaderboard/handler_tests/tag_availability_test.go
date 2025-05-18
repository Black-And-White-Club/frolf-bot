package leaderboardhandler_integration_tests

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"sync"
// 	"testing"
// 	"time"

// 	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
// 	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// 	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
// 	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
// 	"github.com/ThreeDotsLabs/watermill/message"
// 	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
// 	"github.com/google/uuid"
// )

// // TestCase represents a test case for tag availability check handler
// type tagAvailabilityTestCase struct {
// 	name                   string
// 	setupFn                func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard
// 	publishMsgFn           func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message
// 	validateFn             func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard)
// 	expectedOutgoingTopics []string
// 	expectedMessageCounts  map[string]int
// 	expectHandlerError     bool
// }

// // Helper function to create and publish a tag availability check request message
// func createTagAvailabilityCheckRequestMessage(
// 	t *testing.T,
// 	userID sharedtypes.DiscordID,
// 	tagNumber sharedtypes.TagNumber,
// ) *message.Message {
// 	t.Helper()

// 	tagPtr := tagPtr(tagNumber)
// 	payload := leaderboardevents.TagAvailabilityCheckRequestedPayload{
// 		UserID:    userID,
// 		TagNumber: tagPtr,
// 	}

// 	payloadBytes, err := json.Marshal(payload)
// 	if err != nil {
// 		t.Fatalf("failed to marshal payload: %v", err)
// 	}

// 	msg := message.NewMessage(uuid.New().String(), payloadBytes)
// 	msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 	return msg
// }

// // Main test runner for each test case
// func (tc *tagAvailabilityTestCase) runTest(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) {
// 	// Generate a unique ID for this test case to isolate subscriptions
// 	testID := fmt.Sprintf("%s-%s", sanitizeForNATS(t.Name()), uuid.New().String()[:8])

// 	// Setup initial leaderboard
// 	initialLeaderboard := tc.setupFn(t, deps, generator)

// 	// Setup message tracking
// 	receivedMsgs := make(map[string][]*message.Message)
// 	mu := &sync.Mutex{}
// 	subscriberWg := &sync.WaitGroup{}

// 	// Use a cancelable context for subscriptions
// 	subCtx, cancelSub := context.WithCancel(deps.Ctx)

// 	// Ensure proper cleanup
// 	t.Cleanup(func() {
// 		log.Printf("Test Case %q: Canceling subscription context.", t.Name())
// 		cancelSub()

// 		log.Printf("Test Case %q: Waiting for subscriber goroutines to finish.", t.Name())
// 		waitCtx, waitCancel := context.WithTimeout(context.Background(), 5*time.Second)
// 		defer waitCancel()

// 		waitCh := make(chan struct{})
// 		go func() {
// 			subscriberWg.Wait()
// 			close(waitCh)
// 		}()

// 		select {
// 		case <-waitCh:
// 			log.Printf("Test Case %q: Subscriber goroutines finished.", t.Name())
// 		case <-waitCtx.Done():
// 			log.Printf("Test Case %q: WARNING: Subscriber goroutines did not finish within timeout after context cancellation.", t.Name())
// 		}
// 	})

// 	// Setup subscribers for all expected topics
// 	for _, topic := range tc.expectedOutgoingTopics {
// 		msgCh, err := deps.EventBus.Subscribe(subCtx, topic)
// 		if err != nil {
// 			t.Fatalf("Failed to subscribe to topic %q: %v", topic, err)
// 		}

// 		subscriberWg.Add(1)
// 		go func(topic string, messages <-chan *message.Message) {
// 			defer subscriberWg.Done()
// 			for {
// 				select {
// 				case msg, ok := <-messages:
// 					if !ok {
// 						return
// 					}
// 					log.Printf("Test Received message %s on topic %q", msg.UUID, topic)
// 					mu.Lock()
// 					receivedMsgs[topic] = append(receivedMsgs[topic], msg)
// 					mu.Unlock()
// 					msg.Ack()
// 				case <-subCtx.Done():
// 					return
// 				}
// 			}
// 		}(topic, msgCh)
// 	}

// 	// Wait a moment to ensure all subscribers are ready
// 	time.Sleep(200 * time.Millisecond)

// 	// Publish the message to start the flow
// 	incomingMsg := tc.publishMsgFn(t, deps, generator)

// 	// Wait for expected messages on all topics
// 	for _, topic := range tc.expectedOutgoingTopics {
// 		success, err := waitForMessagess(subCtx, topic, receivedMsgs, 5*time.Second, mu)
// 		if err != nil {
// 			log.Printf("⏰ Timeout waiting for topic %q messages: %v", topic, err)

// 			mu.Lock()
// 			for k, msgs := range receivedMsgs {
// 				log.Printf("DEBUG: receivedMsgs has topic %q with %d message(s)", k, len(msgs))
// 			}
// 			mu.Unlock()

// 			if tc.expectHandlerError {
// 				t.Logf("Expected error received: %v", err)
// 			} else {
// 				t.Fatalf("Timed out waiting for messages on topic %q: %v", topic, err)
// 			}
// 		}

// 		if !success && !tc.expectHandlerError {
// 			t.Fatalf("Failed to receive message on topic %q within timeout", topic)
// 		}
// 	}

// 	// Final validation using test case's validate function
// 	tc.validateFn(t, deps, incomingMsg, receivedMsgs, initialLeaderboard)

// 	// Additional cleanup - delete consumers for this test
// 	for _, topic := range tc.expectedOutgoingTopics {
// 		consumerName := fmt.Sprintf("test-%s-%s", sanitizeForNATS(topic), testID)
// 		streamName := determineStreamName(topic)

// 		if streamName != "" {
// 			if err := deleteConsumer(deps.EventBus, streamName, consumerName); err != nil {
// 				t.Logf("Warning: failed to delete consumer %s from stream %s: %v",
// 					consumerName, streamName, err)
// 			}
// 		}
// 	}
// }

// // TestHandleTagAvailabilityCheckRequested runs integration tests for the tag availability handler
// func TestHandleTagAvailabilityCheckRequested(t *testing.T) {
// 	tests := []tagAvailabilityTestCase{
// 		{
// 			name: "Success - Tag Available",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				users := generator.GenerateUsers(1)
// 				initialData := leaderboardtypes.LeaderboardData{
// 					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: tagPtr(1)},
// 				}

// 				initialLeaderboard, err := testutils.InsertLeaderboard(t, testEnv.DB, initialData)
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}
// 				return initialLeaderboard
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				newUser := generator.GenerateUsers(1)[0]

// 				msg := createTagAvailabilityCheckRequestMessage(
// 					t,
// 					sharedtypes.DiscordID(newUser.UserID),
// 					10, // Tag number that should be available
// 				)

// 				if err := testutils.PublishMessage(t, testEnv.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequest, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
// 				// Check TagAvailable response
// 				availableTopic := leaderboardevents.TagAvailable
// 				availableMsgs := receivedMsgs[availableTopic]
// 				if len(availableMsgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none", availableTopic)
// 				}

// 				// Check LeaderboardTagAssignmentRequested message
// 				assignTopic := leaderboardevents.LeaderboardTagAssignmentRequested
// 				assignMsgs := receivedMsgs[assignTopic]
// 				if len(assignMsgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none", assignTopic)
// 				}

// 				// Parse payloads
// 				requestPayload, err := parsePayload[leaderboardevents.TagAvailabilityCheckRequestedPayload](incomingMsg)
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}

// 				availablePayload, err := parsePayload[leaderboardevents.TagAvailablePayload](availableMsgs[0])
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}

// 				assignPayload, err := parsePayload[leaderboardevents.TagAssignmentRequestedPayload](assignMsgs[0])
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}

// 				// Validate available payload
// 				if availablePayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch in available payload: expected %q, got %q",
// 						requestPayload.UserID, availablePayload.UserID)
// 				}
// 				if *availablePayload.TagNumber != *requestPayload.TagNumber {
// 					t.Errorf("TagNumber mismatch in available payload: expected %d, got %d",
// 						*requestPayload.TagNumber, *availablePayload.TagNumber)
// 				}

// 				// Validate assignment payload
// 				if assignPayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch in assign payload: expected %q, got %q",
// 						requestPayload.UserID, assignPayload.UserID)
// 				}
// 				if *assignPayload.TagNumber != *requestPayload.TagNumber {
// 					t.Errorf("TagNumber mismatch in assign payload: expected %d, got %d",
// 						*requestPayload.TagNumber, *assignPayload.TagNumber)
// 				}
// 				if assignPayload.Source != string(leaderboarddb.ServiceUpdateSourceCreateUser) {
// 					t.Errorf("Source mismatch: expected %q, got %q",
// 						leaderboarddb.ServiceUpdateSourceCreateUser, assignPayload.Source)
// 				}
// 				if assignPayload.UpdateType != "automatic" {
// 					t.Errorf("UpdateType mismatch: expected %q, got %q",
// 						"automatic", assignPayload.UpdateType)
// 				}

// 				// Validate correlation ID
// 				if availableMsgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) !=
// 					incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 					t.Errorf("Correlation ID mismatch: expected %q, got %q",
// 						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
// 						availableMsgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
// 				}

// 				// Check for unexpected error messages
// 				unavailableTopic := leaderboardevents.TagUnavailable
// 				failureTopic := leaderboardevents.TagAvailableCheckFailure
// 				if len(receivedMsgs[unavailableTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						unavailableTopic, len(receivedMsgs[unavailableTopic]))
// 				}
// 				if len(receivedMsgs[failureTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						failureTopic, len(receivedMsgs[failureTopic]))
// 				}
// 			},
// 			expectedOutgoingTopics: []string{
// 				leaderboardevents.TagAvailable,
// 				leaderboardevents.LeaderboardTagAssignmentRequested,
// 			},
// 			expectedMessageCounts: map[string]int{
// 				leaderboardevents.TagAvailable:                      1,
// 				leaderboardevents.LeaderboardTagAssignmentRequested: 1,
// 			},
// 			expectHandlerError: false,
// 		},
// 		{
// 			name: "Success - Tag Unavailable (Already Taken)",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				users := generator.GenerateUsers(1)
// 				initialData := leaderboardtypes.LeaderboardData{
// 					{UserID: sharedtypes.DiscordID(users[0].UserID), TagNumber: tagPtr(42)},
// 				}

// 				initialLeaderboard, err := testutils.InsertLeaderboard(t, testEnv.DB, initialData)
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}
// 				return initialLeaderboard
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				newUser := generator.GenerateUsers(1)[0]

// 				msg := createTagAvailabilityCheckRequestMessage(
// 					t,
// 					sharedtypes.DiscordID(newUser.UserID),
// 					42, // Tag number that is already taken
// 				)

// 				if err := testutils.PublishMessage(t, testEnv.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequest, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
// 				unavailableTopic := leaderboardevents.TagUnavailable
// 				unavailableMsgs := receivedMsgs[unavailableTopic]

// 				if len(unavailableMsgs) == 0 {
// 					t.Fatalf("Expected at least one message on topic %q, but received none", unavailableTopic)
// 				}

// 				// Parse payloads
// 				requestPayload, err := parsePayload[leaderboardevents.TagAvailabilityCheckRequestedPayload](incomingMsg)
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}

// 				unavailablePayload, err := parsePayload[leaderboardevents.TagUnavailablePayload](unavailableMsgs[0])
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}

// 				// Validate payload
// 				if unavailablePayload.UserID != requestPayload.UserID {
// 					t.Errorf("UserID mismatch: expected %q, got %q",
// 						requestPayload.UserID, unavailablePayload.UserID)
// 				}
// 				if *unavailablePayload.TagNumber != *requestPayload.TagNumber {
// 					t.Errorf("TagNumber mismatch: expected %d, got %d",
// 						*requestPayload.TagNumber, *unavailablePayload.TagNumber)
// 				}

// 				// Validate correlation ID
// 				if unavailableMsgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) !=
// 					incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
// 					t.Errorf("Correlation ID mismatch: expected %q, got %q",
// 						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
// 						unavailableMsgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey))
// 				}

// 				// Check for unexpected messages
// 				availableTopic := leaderboardevents.TagAvailable
// 				assignTopic := leaderboardevents.LeaderboardTagAssignmentRequested
// 				failureTopic := leaderboardevents.TagAvailableCheckFailure

// 				if len(receivedMsgs[availableTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						availableTopic, len(receivedMsgs[availableTopic]))
// 				}
// 				if len(receivedMsgs[assignTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						assignTopic, len(receivedMsgs[assignTopic]))
// 				}
// 				if len(receivedMsgs[failureTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						failureTopic, len(receivedMsgs[failureTopic]))
// 				}
// 			},
// 			expectedOutgoingTopics: []string{leaderboardevents.TagUnavailable},
// 			expectHandlerError:     false,
// 		},
// 		{
// 			name: "Failure - Invalid Message Payload",
// 			setupFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *leaderboarddb.Leaderboard {
// 				initialLeaderboard, err := testutils.InsertLeaderboard(t, testEnv.DB, leaderboardtypes.LeaderboardData{})
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}
// 				return initialLeaderboard
// 			},
// 			publishMsgFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, generator *testutils.TestDataGenerator) *message.Message {
// 				msg := message.NewMessage(uuid.New().String(), []byte("invalid json payload"))
// 				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
// 				if err := testutils.PublishMessage(t, testEnv.EventBus, context.Background(), leaderboardevents.TagAvailabilityCheckRequest, msg); err != nil {
// 					t.Fatalf("Failed to publish message: %v", err)
// 				}
// 				return msg
// 			},
// 			validateFn: func(t *testing.T, deps LeaderboardHandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialLeaderboard *leaderboarddb.Leaderboard) {
// 				// Check for unexpected messages
// 				availableTopic := leaderboardevents.TagAvailable
// 				unavailableTopic := leaderboardevents.TagUnavailable
// 				assignTopic := leaderboardevents.LeaderboardTagAssignmentRequested
// 				failureTopic := leaderboardevents.TagAvailableCheckFailure

// 				if len(receivedMsgs[availableTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						availableTopic, len(receivedMsgs[availableTopic]))
// 				}
// 				if len(receivedMsgs[unavailableTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						unavailableTopic, len(receivedMsgs[unavailableTopic]))
// 				}
// 				if len(receivedMsgs[assignTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						assignTopic, len(receivedMsgs[assignTopic]))
// 				}
// 				if len(receivedMsgs[failureTopic]) > 0 {
// 					t.Errorf("Expected no messages on topic %q, but received %d",
// 						failureTopic, len(receivedMsgs[failureTopic]))
// 				}

// 				// Validate leaderboard state in database
// 				leaderboards, err := testutils.QueryLeaderboards(t, context.Background(), testEnv.DB)
// 				if err != nil {
// 					t.Fatalf("%v", err)
// 				}
// 				if len(leaderboards) != 1 {
// 					t.Fatalf("Expected 1 leaderboard record in DB, got %d", len(leaderboards))
// 				}
// 				leaderboard := leaderboards[0]
// 				if leaderboard.ID != initialLeaderboard.ID {
// 					t.Errorf("Expected leaderboard ID %d, got %d", initialLeaderboard.ID, leaderboard.ID)
// 				}
// 				if !leaderboard.IsActive {
// 					t.Error("Expected leaderboard to remain active")
// 				}
// 			},
// 			expectedOutgoingTopics: []string{},
// 			expectHandlerError:     true,
// 		},
// 	}

// 	// Run all test cases
// 	for _, tc := range tests {
// 		t.Run(tc.name, func(t *testing.T) {
// 			// Create a fresh, isolated setup for each test case
// 			testDeps := SetupTestLeaderboardHandler(t)
// 			testGenerator := testutils.NewTestDataGenerator(time.Now().UnixNano())

// 			tc.runTest(t, testDeps, testGenerator)
// 		})
// 	}
// }
