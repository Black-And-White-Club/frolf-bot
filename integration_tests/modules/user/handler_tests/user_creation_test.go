package handler_tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// TestHandleUserSignupRequest is an integration test for the HandleUserSignupRequest handler.
func TestHandleUserSignupRequest(t *testing.T) {
	// Use the handler-specific setup function
	deps := SetupTestUserHandler(t, testEnv)
	// Defer the handler-specific cleanup
	defer CleanupHandlerTestDeps(deps)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps)
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - User Signup without Tag",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				// Clean relevant NATS streams
				if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil { // Assuming user events go to "user" stream
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				userID := sharedtypes.DiscordID("testuser-notag-123")
				payload := userevents.UserSignupRequestPayload{
					UserID:    userID,
					TagNumber: nil, // No tag number
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.UserSignupRequest)

				inputTopic := userevents.UserSignupRequest // Assuming the constant is the topic name
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				// 1. Verify user was created in the database
				userID := sharedtypes.DiscordID("testuser-notag-123")
				// Corrected: Declare variable with the correct type *usertypes.UserData
				var createdUser *usertypes.UserData
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					// Use the service from the user module to check the DB
					getUserResult, getUserErr := deps.UserModule.UserService.GetUser(deps.Ctx, userID)
					if getUserErr != nil {
						// If the error is "user not found", keep waiting. Other errors are fatal.
						if errors.Is(getUserErr, errors.New("user not found")) { // Assuming service returns this specific error string on not found
							return errors.New("user not found in DB yet") // Continue waiting
						}
						return fmt.Errorf("unexpected error from GetUser: %w", getUserErr) // Fatal error
					}
					if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayload).User == nil {
						return errors.New("user not found in DB yet (success payload is nil)") // Continue waiting
					}
					// Corrected: Assign to the variable of type *usertypes.UserData
					createdUser = getUserResult.Success.(*userevents.GetUserResponsePayload).User
					return nil // Success, user found
				})
				if err != nil {
					t.Fatalf("User not found in database after waiting: %v", err)
				}
				if createdUser.UserID != userID {
					t.Errorf("Created user ID mismatch: expected %q, got %q", userID, createdUser.UserID)
				}

				// 2. Verify the UserCreated event was published
				expectedTopic := userevents.UserCreated
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var successPayload userevents.UserCreatedPayload
				// Assuming your utils.Helpers.UnmarshalPayload can unmarshal from message.Message
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal UserCreatedPayload: %v", err)
				}

				if successPayload.UserID != userID {
					t.Errorf("UserCreatedPayload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey), receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}
			},
			expectedOutgoingTopics: []string{userevents.UserCreated},
		},
		{
			name: "Success - User Signup with Tag",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				userID := sharedtypes.DiscordID("testuser-withtag-456")
				tagNumber := sharedtypes.TagNumber(24)
				payload := userevents.UserSignupRequestPayload{
					UserID:    userID,
					TagNumber: &tagNumber,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.UserSignupRequest)

				inputTopic := userevents.UserSignupRequest
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				userID := sharedtypes.DiscordID("testuser-withtag-456")

				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(deps.Ctx, userID)

				if getUserErr == nil && getUserResult.Success != nil {
					foundUser := getUserResult.Success.(*userevents.GetUserResponsePayload).User
					t.Fatalf("Expected user %q NOT to be created, but found: %+v", userID, foundUser)
				}

				if getUserErr == nil && getUserResult.Failure != nil {
					failurePayload, ok := getUserResult.Failure.(*userevents.GetUserFailedPayload)
					if !ok || failurePayload.Reason != "user not found" {
						t.Errorf("Expected GetUser to return 'user not found' failure or technical error, but got unexpected failure payload: %+v (error: %v)", getUserResult.Failure, getUserErr)
					}
				} else if getUserErr == nil && getUserResult.Failure == nil {
					t.Errorf("Expected GetUser to return 'user not found' failure or technical error, but got nil error and no payload")
				}

				expectedTopic := userevents.TagAvailabilityCheckRequested
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var checkPayload userevents.TagAvailabilityCheckRequestedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &checkPayload); err != nil {
					t.Fatalf("Failed to unmarshal TagAvailabilityCheckRequestedPayload: %v", err)
				}

				tagNumber := sharedtypes.TagNumber(24)
				if checkPayload.TagNumber != tagNumber {
					t.Errorf("TagAvailabilityCheckRequestedPayload TagNumber mismatch: expected %d, got %d", tagNumber, checkPayload.TagNumber)
				}
				if checkPayload.UserID != userID {
					t.Errorf("TagAvailabilityCheckRequestedPayload UserID mismatch: expected %q, got %q", userID, checkPayload.UserID)
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey), receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				unexpectedTopic := userevents.UserCreated
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectedOutgoingTopics: []string{userevents.TagAvailabilityCheckRequested},
		},
		{
			name: "Failure - User Already Exists",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean database before test case: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS streams before test case: %v", err)
				}

				// Pre-create the user to simulate "already exists" scenario
				userID := sharedtypes.DiscordID("testuser-exists-789")
				tag := sharedtypes.TagNumber(23) // Dummy tag
				// Use the service from the user module to create the user
				createResult, createErr := deps.UserModule.UserService.CreateUser(deps.Ctx, userID, &tag)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create user for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				log.Printf("Pre-created user %q for test", userID)
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				userID := sharedtypes.DiscordID("testuser-exists-789") // Same user ID as pre-created
				payload := userevents.UserSignupRequestPayload{
					UserID:    userID,
					TagNumber: nil, // No tag number, will attempt creation
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.UserSignupRequest)

				inputTopic := userevents.UserSignupRequest
				if err := deps.EventBus.Publish(inputTopic, msg); err != nil {
					t.Fatalf("Failed to publish message to handler input topic %q: %v", inputTopic, err)
				}
				log.Printf("Published message %s to topic %q", msg.UUID, inputTopic)
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				// 1. Verify user still exists (no change expected from signup attempt)
				userID := sharedtypes.DiscordID("testuser-exists-789")
				// Corrected: Declare variable with the correct type *usertypes.UserData
				var existingUser *usertypes.UserData
				// Use the service from the user module to check the DB
				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(deps.Ctx, userID)
				// Expect no error and a successful result
				if getUserErr != nil {
					t.Fatalf("Expected GetUser to succeed for existing user, but got error: %v", getUserErr)
				}
				if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayload).User == nil {
					t.Fatalf("Expected GetUser to return success payload for existing user, but got nil. Failure: %+v", getUserResult.Failure)
				}
				// Corrected: Assign to the variable of type *usertypes.UserData
				existingUser = getUserResult.Success.(*userevents.GetUserResponsePayload).User
				if existingUser.UserID != userID {
					t.Errorf("Existing user ID mismatch: expected %q, got %q", userID, existingUser.UserID)
				}
				// Assert original role/tag on existingUser (*usertypes.UserData) if applicable

				// 2. Verify the UserCreationFailed event was published
				expectedTopic := userevents.UserCreationFailed // Assuming the constant is the topic name
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var failedPayload userevents.UserCreationFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &failedPayload); err != nil {
					t.Fatalf("Failed to unmarshal UserCreationFailedPayload: %v", err)
				}

				if failedPayload.UserID != userID {
					t.Errorf("UserCreationFailedPayload UserID mismatch: expected %q, got %q", userID, failedPayload.UserID)
				}
				expectedReason := "user already exists" // Assuming this is the reason from your service/wrapper
				if failedPayload.Reason != expectedReason {
					t.Errorf("UserCreationFailedPayload Reason mismatch: expected %q, got %q", expectedReason, failedPayload.Reason)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey), receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// 3. Verify no UserCreated event was published
				unexpectedTopic := userevents.UserCreated
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup test case specific dependencies and state
			tc.setupFn(t, deps)

			receivedMsgs := make(map[string][]*message.Message)
			subscribers := make(map[string]message.Subscriber)
			subCtx, cancelSub := context.WithTimeout(deps.Ctx, 5*time.Second)
			defer cancelSub()

			// Subscribe to each expected outgoing topic
			for _, topic := range tc.expectedOutgoingTopics {
				consumerName := fmt.Sprintf("test-consumer-%s-%s", sanitizeTopicForNATS(topic), uuid.New().String()[:8])
				sub, err := deps.EventBus.Subscribe(subCtx, topic)
				if err != nil {
					t.Fatalf("Failed to subscribe to topic %q: %v", topic, err)
				}
				subscribers[topic] = deps.EventBus // Store the EventBus
				log.Printf("Subscribed to topic %q with consumer %q", topic, consumerName)

				// Start a goroutine to collect messages from this subscription
				go func(topic string, messages <-chan *message.Message) {
					for msg := range messages {
						log.Printf("Received message %s on topic %q", msg.UUID, topic)
						receivedMsgs[topic] = append(receivedMsgs[topic], msg)
						// Acknowledge the message so it's not redelivered
						msg.Ack()
					}
				}(topic, sub) // Pass the topic and the channel to the goroutine
			}

			// --- Publish the incoming message ---
			incomingMsg := tc.publishMsgFn(t, deps)

			time.Sleep(500 * time.Millisecond)

			// --- Validate the results ---
			tc.validateFn(t, deps, incomingMsg, receivedMsgs)
		})
	}
}

// sanitizeTopicForNATS is a helper to create a valid NATS consumer name from a topic.
func sanitizeTopicForNATS(topic string) string {
	sanitized := strings.ReplaceAll(topic, ".", "_")
	reg := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	sanitized = reg.ReplaceAllString(sanitized, "")
	sanitized = strings.TrimPrefix(sanitized, "-")
	sanitized = strings.TrimSuffix(sanitized, "-")
	return sanitized
}
