package userhandler_integration_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
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
	// Define the test cases using an anonymous struct, similar to the leaderboard example
	tests := []struct {
		name string
		// Modified function signatures to explicitly accept deps
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration // Added timeout field for consistency
	}{
		{
			name: "Success - User Signup without Tag",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No specific setup needed for this case beyond what SetupTestUserHandler provides.
				return nil // Return initial state if any
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-notag-123")
				payload := userevents.UserSignupRequestPayload{
					GuildID:   "test-guild",
					UserID:    userID,
					TagNumber: nil, // No tag number
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage to publish the message.
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequest, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreated},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// 1. Verify user was created in the database (via service call, as per your strategy)
				userID := sharedtypes.DiscordID("testuser-notag-123")
				// Use WaitFor for eventual consistency in DB
				var createdUser *usertypes.UserData
				guildID := sharedtypes.GuildID("test-guild")
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
					if getUserErr != nil {
						return fmt.Errorf("service returned error: %w", getUserErr)
					}
					if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayload).User == nil {
						return errors.New("user not found in DB yet or success payload is nil")
					}
					createdUser = getUserResult.Success.(*userevents.GetUserResponsePayload).User
					return nil
				})
				if err != nil {
					t.Fatalf("User not found in database after waiting: %v", err)
				}

				if createdUser.UserID != userID {
					t.Errorf("Created user ID mismatch: expected %q, got %q", userID, createdUser.UserID)
				}
				// Removed createdUser.TagNumber check as usertypes.UserData does not contain it.

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
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil { // Use deps.UserModule.Helper
					t.Fatalf("Failed to unmarshal UserCreatedPayload: %v", err)
				}

				if successPayload.UserID != userID {
					t.Errorf("UserCreatedPayload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				// This check is correct for the event payload
				if successPayload.TagNumber != nil {
					t.Errorf("UserCreatedPayload TagNumber mismatch: expected nil, got %d", *successPayload.TagNumber)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey), receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second, // Default timeout for this test case
		},
		{
			name: "Success - User Signup with Tag (requests availability check)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No specific setup needed for this case beyond what SetupTestUserHandler provides.
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-withtag-456")
				tagNumber := sharedtypes.TagNumber(24)
				payload := userevents.UserSignupRequestPayload{
					GuildID:   "test-guild",
					UserID:    userID,
					TagNumber: &tagNumber,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequest, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.TagAvailabilityCheckRequested},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("testuser-withtag-456")
				tagNumber := sharedtypes.TagNumber(24)

				// Verify user is NOT created in the database (via service call)
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				// Expecting an error or a failure payload indicating "not found"
				if getUserErr == nil { // No technical error, now check business result
					if getUserResult.Success != nil {
						foundUser := getUserResult.Success.(*userevents.GetUserResponsePayload).User
						t.Fatalf("Expected user %q NOT to be created, but found: %+v", userID, foundUser)
					}
					if getUserResult.Failure == nil {
						t.Errorf("Expected GetUser to return 'user not found' failure or technical error, but got nil results")
					} else {
						failurePayload, ok := getUserResult.Failure.(*userevents.GetUserFailedPayload)
						if !ok || failurePayload.Reason != "user not found" { // Assuming service returns this specific reason
							t.Errorf("Expected GetUser to return 'user not found' failure, but got unexpected failure payload: %+v", getUserResult.Failure)
						}
					}
				} else if !errors.Is(getUserErr, errors.New("user not found")) { // Check if it's the expected "user not found" error
					t.Errorf("Expected GetUser to return 'user not found' error, but got unexpected error: %v", getUserErr)
				}

				// Verify the TagAvailabilityCheckRequested event was published
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
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &checkPayload); err != nil { // Use deps.UserModule.Helper
					t.Fatalf("Failed to unmarshal TagAvailabilityCheckRequestedPayload: %v", err)
				}

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
			expectHandlerError: false,
			timeout:            5 * time.Second, // Default timeout for this test case
		},
		{
			name: "Failure - User Already Exists (no tag)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Pre-create the user to simulate "already exists" scenario
				userID := sharedtypes.DiscordID("testuser-exists-789")
				tag := sharedtypes.TagNumber(23) // Dummy tag for pre-creation
				// Use the service from the user module to create the user
				guildID := sharedtypes.GuildID("test-guild")
				createResult, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, &tag) // Use env.Ctx
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create user for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				log.Printf("Pre-created user %q for test", userID)
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-exists-789") // Same user ID as pre-created
				payload := userevents.UserSignupRequestPayload{
					GuildID:   "test-guild",
					UserID:    userID,
					TagNumber: nil, // No tag number, will attempt creation
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequest, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// 1. Verify user still exists (no change expected from signup attempt)
				userID := sharedtypes.DiscordID("testuser-exists-789")
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				// Expect no technical error and a successful result (user was already there)
				if getUserErr != nil {
					t.Fatalf("Expected GetUser to succeed for existing user, but got error: %v", getUserErr)
				}
				if getUserResult.Success == nil || getUserResult.Success.(*userevents.GetUserResponsePayload).User == nil {
					t.Fatalf("Expected GetUser to return success payload for existing user, but got nil. Failure: %+v", getUserResult.Failure)
				}
				existingUser := getUserResult.Success.(*userevents.GetUserResponsePayload).User
				if existingUser.UserID != userID {
					t.Errorf("Existing user ID mismatch: expected %q, got %q", userID, existingUser.UserID)
				}
				// Removed existingUser.TagNumber check as usertypes.UserData does not contain it.

				// 2. Verify the UserCreationFailed event was published
				expectedTopic := userevents.UserCreationFailed
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var failedPayload userevents.UserCreationFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &failedPayload); err != nil { // Use deps.UserModule.Helper
					t.Fatalf("Failed to unmarshal UserCreationFailedPayload: %v", err)
				}

				if failedPayload.UserID != userID {
					t.Errorf("UserCreationFailedPayload UserID mismatch: expected %q, got %q", userID, failedPayload.UserID)
				}
				expectedReason := "user already exists" // Assuming this is the reason from your service/wrapper
				if failedPayload.Reason != expectedReason {
					t.Errorf("UserCreationFailedPayload Reason mismatch: expected %q, got %q", expectedReason, failedPayload.Reason)
				}
				// This check is correct for the event payload
				if failedPayload.TagNumber != nil {
					t.Errorf("UserCreationFailedPayload TagNumber mismatch: expected nil, got %v", failedPayload.TagNumber)
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
			expectHandlerError: false,
			timeout:            5 * time.Second, // Default timeout for this test case
		},
	}

	// Run the test cases using testutils.RunTest
	for _, tc := range tests {
		tc := tc // capture range variable for use in t.Run closure
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestUserHandler(t) // Setup dependencies for each subtest

			// Construct the testutils.TestCase from the anonymous struct fields
			genericCase := testutils.TestCase{
				Name: tc.name,
				// Pass deps to the inner function calls
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					// The tc.validateFn needs access to `deps`, which is captured by this outer closure
					tc.validateFn(t, deps, env, incomingMsg, receivedMsgs, initialState) // FIX: Pass 'env' here
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
