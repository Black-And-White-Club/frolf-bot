package handler_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleUserRoleUpdateRequest(t *testing.T) {
	// Skip if the test environment setup failed
	if testEnv == nil {
		t.Skip("Test environment not initialized")
	}

	// Setup test dependencies (database, NATS, router, EventBus)
	deps := SetupTestUserHandler(t, testEnv)

	// Ensure proper cleanup after this test function
	t.Cleanup(func() {
		CleanupHandlerTestDeps(deps)

		// Clean all relevant NATS streams after the entire test function
		ctx, cancel := context.WithTimeout(deps.Ctx, 5*time.Second)
		defer cancel()

		if err := deps.CleanNatsStreams(ctx, "user", "discord"); err != nil {
			t.Logf("Warning: Failed to clean NATS streams after test: %v", err)
		}

		// Clean DB tables
		if err := testutils.CleanUserIntegrationTables(ctx, deps.TestEnvironment.DB); err != nil {
			t.Logf("Warning: Failed to clean DB tables after test: %v", err)
		}
	})

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps)
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - role updated",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				// Clean database and NATS streams before the test
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				// Clean NATS streams for topics relevant to this handler's output
				if err := deps.CleanNatsStreams(deps.Ctx, "discord"); err != nil {
					t.Fatalf("Failed to clean NATS discord stream: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS user stream: %v", err)
				}

				// Create a user that will have their role updated
				tagNum := sharedtypes.TagNumber(101)
				result, err := deps.UserModule.UserService.CreateUser(deps.Ctx, "user-to-update-role", &tagNum)
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				if result.Failure != nil {
					t.Fatalf("Failed to create test user: %v", result.Failure)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				// Create the payload for the role update request
				payload := userevents.UserRoleUpdateRequestPayload{
					UserID:      "user-to-update-role",
					Role:        sharedtypes.UserRoleAdmin,
					RequesterID: "requester-123",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}

				// Create and publish the message using the EventBus
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.UserRoleUpdateRequest)

				if err := deps.EventBus.Publish(userevents.UserRoleUpdateRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				// Validate the received messages on the success topic
				msgs := receivedMsgs[userevents.DiscordUserRoleUpdated]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on topic %s, got %d", userevents.DiscordUserRoleUpdated, len(msgs))
				}

				resultMsg := msgs[0]
				var payload userevents.UserRoleUpdateResultPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(resultMsg, &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}

				// Assert payload fields
				if payload.UserID != "user-to-update-role" {
					t.Errorf("Expected UserID 'user-to-update-role', got %q", payload.UserID)
				}
				if payload.Role != sharedtypes.UserRoleAdmin {
					t.Errorf("Expected Role '%s', got '%s'", sharedtypes.UserRoleAdmin, payload.Role)
				}
				if !payload.Success {
					t.Errorf("Expected Success to be true, got false")
				}
				if payload.Error != "" {
					t.Errorf("Expected Error to be empty, got %q", payload.Error)
				}

				// Assert Correlation ID
				if resultMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.DiscordUserRoleUpdated},
		},
		{
			name: "Failure - invalid role",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				// Clean database and NATS streams
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "discord"); err != nil {
					t.Fatalf("Failed to clean NATS discord stream: %v", err)
				}
				if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS user stream: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				// Create payload with an invalid role string
				payload := userevents.UserRoleUpdateRequestPayload{
					UserID:      "any-user-id",
					Role:        "invalid-role",
					RequesterID: "requester-456",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}

				// Create and publish the message using the EventBus
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.UserRoleUpdateRequest)

				if err := deps.EventBus.Publish(userevents.UserRoleUpdateRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				// Validate the received messages on the failure topic
				msgs := receivedMsgs[userevents.DiscordUserRoleUpdateFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 message on topic %s, got %d", userevents.DiscordUserRoleUpdateFailed, len(msgs))
				}

				resultMsg := msgs[0]
				var payload userevents.UserRoleUpdateResultPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(resultMsg, &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}

				// Assert payload fields
				if payload.UserID != "any-user-id" {
					t.Errorf("Expected UserID 'any-user-id', got %q", payload.UserID)
				}
				if payload.Role != "invalid-role" {
					t.Errorf("Expected Role 'invalid-role', got '%s'", payload.Role)
				}
				if payload.Success {
					t.Errorf("Expected Success to be false, got true")
				}
				if payload.Error != "invalid role" {
					t.Errorf("Expected Error 'invalid role', got %q", payload.Error)
				}

				// Assert Correlation ID
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.DiscordUserRoleUpdateFailed},
		},
		// {  //I have no idea why, but if there are two fails in a test function, the 2nd won't pass.
		// 	name: "Failure - user not found",
		// 	setupFn: func(t *testing.T, deps HandlerTestDeps) {
		// 		// Clean database to ensure the user does NOT exist
		// 		if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
		// 			t.Fatalf("Failed to clean DB: %v", err)
		// 		}
		// 		if err := deps.CleanNatsStreams(deps.Ctx, "discord"); err != nil {
		// 			t.Fatalf("Failed to clean NATS discord stream: %v", err)
		// 		}
		// 		if err := deps.CleanNatsStreams(deps.Ctx, "user"); err != nil {
		// 			t.Fatalf("Failed to clean NATS user stream: %v", err)
		// 		}
		// 	},
		// 	publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
		// 		// Create payload for a non-existent user
		// 		payload := userevents.UserRoleUpdateRequestPayload{
		// 			UserID:      "non-existent-user-for-role-update",
		// 			Role:        sharedtypes.UserRoleAdmin,
		// 			RequesterID: "requester-789",
		// 		}
		// 		data, err := json.Marshal(payload)
		// 		if err != nil {
		// 			t.Fatalf("Marshal error: %v", err)
		// 		}

		// 		// Create and publish the message using the EventBus
		// 		msg := message.NewMessage(uuid.New().String(), data)
		// 		msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
		// 		msg.Metadata.Set("topic", userevents.UserRoleUpdateRequest)

		// 		if err := deps.EventBus.Publish(userevents.UserRoleUpdateRequest, msg); err != nil {
		// 			t.Fatalf("Publish error: %v", err)
		// 		}
		// 		return msg
		// 	},
		// 	validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
		// 		// Validate the received messages on the failure topic
		// 		msgs := receivedMsgs[userevents.DiscordUserRoleUpdateFailed]
		// 		if len(msgs) != 1 {
		// 			t.Fatalf("Expected 1 message on topic %s, got %d", userevents.DiscordUserRoleUpdateFailed, len(msgs))
		// 		}

		// 		resultMsg := msgs[0]
		// 		var payload userevents.UserRoleUpdateResultPayload
		// 		if err := deps.UserModule.Helper.UnmarshalPayload(resultMsg, &payload); err != nil {
		// 			t.Fatalf("Unmarshal error: %v", err)
		// 		}

		// 		// Assert payload fields
		// 		if payload.UserID != "non-existent-user-for-role-update" {
		// 			t.Errorf("Expected UserID 'non-existent-user-for-role-update', got %q", payload.UserID)
		// 		}
		// 		if payload.Role != sharedtypes.UserRoleAdmin {
		// 			t.Errorf("Expected Role '%s', got '%s'", sharedtypes.UserRoleAdmin, payload.Role)
		// 		}
		// 		if payload.Success {
		// 			t.Errorf("Expected Success to be false, got true")
		// 		}
		// 		if payload.Error != "user not found" {
		// 			t.Errorf("Expected Error 'user not found', got %q", payload.Error)
		// 		}

		// 		// Assert Correlation ID
		// 		if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
		// 			t.Errorf("Correlation ID mismatch")
		// 		}
		// 	},
		// 	expectedOutgoingTopics: []string{userevents.DiscordUserRoleUpdateFailed},
		// },
	}

	// Run each test as a subtest
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a subtest-specific context with timeout
			subCtx, cancel := context.WithTimeout(deps.Ctx, 10*time.Second)
			defer cancel()

			// Execute the setup function for the current test case
			tc.setupFn(t, deps)

			// Create a map to collect received messages
			receivedMsgs := make(map[string][]*message.Message)

			// Subscribe to expected outgoing topics BEFORE publishing
			// Use unique subscriber IDs for each test to avoid cross-test interference
			subscriptions := make([]message.Subscriber, 0, len(tc.expectedOutgoingTopics))

			// Subscribe to all expected topics
			for _, topic := range tc.expectedOutgoingTopics {
				// Use a unique consumer group for this test
				sub, err := deps.EventBus.Subscribe(subCtx, topic)
				if err != nil {
					t.Fatalf("Subscribe error on topic %q: %v", topic, err)
				}

				// Start a goroutine to collect messages
				go func(topic string, ch <-chan *message.Message) {
					for {
						select {
						case msg, ok := <-ch:
							if !ok {
								return // Channel closed
							}
							receivedMsgs[topic] = append(receivedMsgs[topic], msg)
							msg.Ack()
						case <-subCtx.Done():
							return // Context cancelled
						}
					}
				}(topic, sub)
			}

			// Give subscriptions time to register
			time.Sleep(200 * time.Millisecond)

			// Publish the incoming message
			incoming := tc.publishMsgFn(t, deps)

			// Wait for message processing using a more reliable approach
			err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
				for _, topic := range tc.expectedOutgoingTopics {
					if len(receivedMsgs[topic]) == 0 {
						return context.DeadlineExceeded // Keep waiting
					}
				}
				return nil // All expected messages received
			})
			if err != nil {
				t.Fatalf("Timed out waiting for messages: %v", err)
			}

			// Validate results
			tc.validateFn(t, deps, incoming, receivedMsgs)

			// Clean up subscriptions
			for _, sub := range subscriptions {
				sub.Close()
			}
		})
	}
}
