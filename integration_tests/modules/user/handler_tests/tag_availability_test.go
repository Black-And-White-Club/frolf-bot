package userhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time" // Import time for timeout field

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared" // Import sharedtypes for UserRoleEnum
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// --- Test Cases ---

func TestHandleTagAvailable(t *testing.T) {
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
			name: "Success - user created from tag available",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No specific setup needed for this case, DB and NATS cleanup are handled by testutils.RunTest
				return nil // Return initial state if any
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := userevents.TagAvailablePayload{
					GuildID:   "test-guild",
					UserID:    "test-tag-user-available",
					TagNumber: 21,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				// RETAINED: Use testutils.PublishMessage as RunTest does not publish the message
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.TagAvailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreated},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[userevents.UserCreated]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreated message, got %d", len(msgs))
				}
				var payload userevents.UserCreatedPayload
				// Access Helper via the passed deps argument
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "test-tag-user-available" {
					t.Errorf("Expected UserID 'test-tag-user-available', got %q", payload.UserID)
				}
				if payload.TagNumber == nil || *payload.TagNumber != 21 {
					t.Errorf("Expected TagNumber 21, got %v", payload.TagNumber)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
		},
		{
			name: "Failure - user already exists",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Create the user that will cause the "already exists" error
				// Use testutils.InsertUser to directly insert the user into the database
				guildID := sharedtypes.GuildID("test-guild")
				err := testutils.InsertUser(t, env.DB, "existing-tag-user", guildID, sharedtypes.UserRoleUser)
				if err != nil {
					t.Fatalf("Failed to insert pre-existing user via testutils.InsertUser: %v", err)
				}
				return nil // Return initial state if any
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := userevents.TagAvailablePayload{
					GuildID:   "test-guild",
					UserID:    "existing-tag-user",
					TagNumber: 22,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				// RETAINED: Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.TagAvailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[userevents.UserCreationFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreationFailed message, got %d", len(msgs))
				}
				var payload userevents.UserCreationFailedPayload
				// Access Helper via the passed deps argument
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "existing-tag-user" {
					t.Errorf("Expected UserID 'existing-tag-user', got %q", payload.UserID)
				}
				if payload.TagNumber == nil || *payload.TagNumber != 22 {
					t.Errorf("Expected TagNumber 22, got %v", payload.TagNumber)
				}
				// Check the reason field
				if payload.Reason == "" {
					t.Errorf("Expected non-empty reason, got empty")
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false, // This should be false if an error event is published
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

func TestHandleTagUnavailable(t *testing.T) {
	// Define the test cases using an anonymous struct
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
			name: "Always fails with 'tag not available'",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No specific setup needed for this case
				return nil // Return initial state if any
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := userevents.TagUnavailablePayload{
					UserID:    "tag-unavail-user",
					TagNumber: 77,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				// RETAINED: Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.TagUnavailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[userevents.UserCreationFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreationFailed message, got %d", len(msgs))
				}
				var payload userevents.UserCreationFailedPayload
				// Access Helper via the passed deps argument
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "tag-unavail-user" {
					t.Errorf("Expected UserID 'tag-unavail-user', got %q", payload.UserID)
				}
				if payload.TagNumber == nil || *payload.TagNumber != 77 {
					t.Errorf("Expected TagNumber 77, got %v", payload.TagNumber)
				}
				if payload.Reason != "tag not available" {
					t.Errorf("Expected reason 'tag not available', got %q", payload.Reason)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false, // This should be false if an error event is published
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
