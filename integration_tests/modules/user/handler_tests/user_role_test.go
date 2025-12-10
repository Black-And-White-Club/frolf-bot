package userhandler_integration_tests

import (
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

// TestHandleUserRoleUpdateRequest is an integration test for the HandleUserRoleUpdateRequest handler.
func TestHandleUserRoleUpdateRequest(t *testing.T) {
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
			name: "Success - role updated",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Create a user that will have their role updated
				userID := sharedtypes.DiscordID("user-to-update-role")
				tagNum := sharedtypes.TagNumber(101)
				// Use UserService.CreateUser to ensure the user exists in the database
				guildID := sharedtypes.GuildID("test-guild")
				createResult, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, &tagNum, nil, nil)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to create test user for role update: %v, result: %+v", createErr, createResult.Failure)
				}
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// Create the payload for the role update request
				payload := userevents.UserRoleUpdateRequestPayload{
					GuildID:     "test-guild",
					UserID:      "user-to-update-role",
					Role:        sharedtypes.UserRoleAdmin,
					RequesterID: "requester-123",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}

				// Create and publish the message using testutils.PublishMessage
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserRoleUpdateRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
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
			timeout:                5 * time.Second,
		},
		{
			name: "Failure - invalid role",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No specific setup needed for this test case, as the role is invalid
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// Create payload with an invalid role string
				payload := userevents.UserRoleUpdateRequestPayload{
					GuildID:     "test-guild",
					UserID:      "any-user-id",
					Role:        "invalid-role", // This will cause the failure
					RequesterID: "requester-456",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}

				// Create and publish the message using testutils.PublishMessage
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserRoleUpdateRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
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
				if payload.Error != "invalid role" { // Assuming this is the expected error message from the handler
					t.Errorf("Expected Error 'invalid role', got %q", payload.Error)
				}

				// Assert Correlation ID
				if resultMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.DiscordUserRoleUpdateFailed},
			timeout:                5 * time.Second,
		},
		{
			name: "Failure - user not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Ensure the user does NOT exist in the database for this test
				// No user creation needed here.
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// Create payload for a non-existent user
				payload := userevents.UserRoleUpdateRequestPayload{
					GuildID:     "test-guild",
					UserID:      "non-existent-user-for-role-update",
					Role:        sharedtypes.UserRoleAdmin,
					RequesterID: "requester-789",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}

				// Create and publish the message using testutils.PublishMessage
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserRoleUpdateRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
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
				if payload.UserID != "non-existent-user-for-role-update" {
					t.Errorf("Expected UserID 'non-existent-user-for-role-update', got %q", payload.UserID)
				}
				if payload.Role != sharedtypes.UserRoleAdmin {
					t.Errorf("Expected Role '%s', got '%s'", sharedtypes.UserRoleAdmin, payload.Role)
				}
				if payload.Success {
					t.Errorf("Expected Success to be false, got true")
				}
				if payload.Error != "user not found" { // Assuming this is the expected error message from the handler
					t.Errorf("Expected Error 'user not found', got %q", payload.Error)
				}

				// Assert Correlation ID
				if resultMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.DiscordUserRoleUpdateFailed},
			timeout:                5 * time.Second,
		},
	}

	// Run each test as a subtest
	for _, tc := range tests {
		tc := tc // capture range variable for use in t.Run closure
		t.Run(tc.name, func(t *testing.T) {
			// Setup dependencies for each subtest
			deps := SetupTestUserHandler(t)

			// Construct the testutils.TestCase from the anonymous struct fields
			genericCase := testutils.TestCase{
				Name: tc.name,
				// Pass deps and env to the inner function calls
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, env, incomingMsg, receivedMsgs, initialState)
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
