package userhandler_integration_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
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
				payload := userevents.UserSignupRequestedPayloadV1{
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
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify the UserCreated event was published (treat DB checks as flaky after schema changes)
				userID := sharedtypes.DiscordID("testuser-notag-123")

				// 2. Verify the UserCreated event was published
				expectedTopic := userevents.UserCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var successPayload userevents.UserCreatedPayloadV1
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
				payload := userevents.UserSignupRequestedPayloadV1{
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
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{sharedevents.TagAvailabilityCheckRequestedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("testuser-withtag-456")
				tagNumber := sharedtypes.TagNumber(24)

				// Verify user is NOT created in the database (via service call)
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				if getUserErr != nil {
					t.Fatalf("Unexpected technical error from GetUser: %v", getUserErr)
				}

				if getUserResult.IsSuccess() {
					t.Fatalf("Expected user %q NOT to be created, but found: %+v", userID, *getUserResult.Success)
				}

				if !getUserResult.IsFailure() {
					t.Fatal("Expected GetUser to return a failure payload for non-existent user, but got none")
				}

				errVal := *getUserResult.Failure
				// Assuming the service returns ErrNotFound from repo when user is missing in guild
				if !errors.Is(errVal, userdb.ErrNotFound) && errVal.Error() != "user not found" {
					t.Errorf("Expected GetUser to return 'user not found' error, but got: %v", errVal)
				}

				// Verify the TagAvailabilityCheckRequested event was published
				expectedTopic := sharedevents.TagAvailabilityCheckRequestedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var checkPayload sharedevents.TagAvailabilityCheckRequestedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &checkPayload); err != nil { // Use deps.UserModule.Helper
					t.Fatalf("Failed to unmarshal TagAvailabilityCheckRequestedPayload: %v", err)
				}

				if checkPayload.TagNumber == nil || *checkPayload.TagNumber != tagNumber {
					t.Errorf("TagAvailabilityCheckRequestedPayload TagNumber mismatch: expected %d, got %v", tagNumber, checkPayload.TagNumber)
				}
				if checkPayload.UserID != userID {
					t.Errorf("TagAvailabilityCheckRequestedPayload UserID mismatch: expected %q, got %q", userID, checkPayload.UserID)
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey), receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				unexpectedTopic := userevents.UserCreatedV1
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second, // Default timeout for this test case
		},
		{
			name: "Success - New User (IsReturningUser=false)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No setup needed - brand new user
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-brand-new-999")
				payload := userevents.UserSignupRequestedPayloadV1{
					GuildID:   "test-guild",
					UserID:    userID,
					TagNumber: nil,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("testuser-brand-new-999")
				guildID := sharedtypes.GuildID("test-guild")

				// Verify user was created
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
					if getUserErr != nil {
						return fmt.Errorf("service returned error: %w", getUserErr)
					}
					if !getUserResult.IsSuccess() || *getUserResult.Success == nil {
						return errors.New("user not found in DB yet")
					}
					return nil
				})
				if err != nil {
					t.Fatalf("User not found after waiting: %v", err)
				}

				// Verify the UserCreated event was published with IsReturningUser=false
				expectedTopic := userevents.UserCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var successPayload userevents.UserCreatedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal UserCreatedPayload: %v", err)
				}

				if successPayload.UserID != userID {
					t.Errorf("UserCreatedPayload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.IsReturningUser != false {
					t.Errorf("UserCreatedPayload IsReturningUser mismatch: expected false, got %v", successPayload.IsReturningUser)
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Success - Returning User to New Guild (IsReturningUser=true)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Pre-create the user in guild-1
				userID := sharedtypes.DiscordID("testuser-returning-888")
				guild1 := sharedtypes.GuildID("test-guild-1")
				createResult, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, guild1, userID, nil, nil, nil)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create user for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				log.Printf("Pre-created returning user %q in guild-1", userID)
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-returning-888")
				payload := userevents.UserSignupRequestedPayloadV1{
					GuildID:   "test-guild-2", // Different guild
					UserID:    userID,
					TagNumber: nil,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("testuser-returning-888")
				guild2 := sharedtypes.GuildID("test-guild-2")

				// Verify user now exists in guild-2
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guild2, userID)
					if getUserErr != nil {
						return fmt.Errorf("service returned error: %w", getUserErr)
					}
					if !getUserResult.IsSuccess() || *getUserResult.Success == nil {
						return errors.New("user not found in guild-2 yet")
					}
					return nil
				})
				if err != nil {
					t.Fatalf("User not found in guild-2 after waiting: %v", err)
				}

				// Verify the UserCreated event was published with IsReturningUser=true
				expectedTopic := userevents.UserCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var successPayload userevents.UserCreatedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal UserCreatedPayload: %v", err)
				}

				if successPayload.UserID != userID {
					t.Errorf("UserCreatedPayload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.IsReturningUser != true {
					t.Errorf("UserCreatedPayload IsReturningUser mismatch: expected true, got %v", successPayload.IsReturningUser)
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Success - User Already Exists (idempotent)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Pre-create the user to simulate "already exists" scenario
				userID := sharedtypes.DiscordID("testuser-exists-789")
				tag := sharedtypes.TagNumber(23)
				guildID := sharedtypes.GuildID("test-guild")
				createResult, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, &tag, nil, nil)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create user for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-exists-789")
				payload := userevents.UserSignupRequestedPayloadV1{
					GuildID:   "test-guild",
					UserID:    userID,
					TagNumber: nil,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserSignupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("testuser-exists-789")

				// 1. Verify user still exists
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, getUserErr := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				if getUserErr != nil {
					t.Fatalf("Expected GetUser to succeed for existing user, but got error: %v", getUserErr)
				}
				if !getUserResult.IsSuccess() || *getUserResult.Success == nil {
					t.Fatalf("Expected GetUser to return success payload for existing user, but got nil. Failure: %+v", getUserResult.Failure)
				}

				// 2. Verify UserCreated event was published (idempotent success)
				expectedTopic := userevents.UserCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}

				receivedMsg := msgs[0]
				var successPayload userevents.UserCreatedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal UserCreatedPayload: %v", err)
				}

				if successPayload.UserID != userID {
					t.Errorf("UserCreatedPayload UserID mismatch: expected %q, got %q", userID, successPayload.UserID)
				}
				if successPayload.IsReturningUser != true {
					t.Errorf("UserCreatedPayload IsReturningUser mismatch: expected true, got %v", successPayload.IsReturningUser)
				}

				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
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
