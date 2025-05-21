package userhandler_integration_tests

import (
	"encoding/json"
	"log"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// TestHandleGetUserRequest is an integration test for the HandleGetUserRequest handler.
func TestHandleGetUserRequest(t *testing.T) {
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
			name: "Success - user found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Create a test user directly in the database for setup
				userID := sharedtypes.DiscordID("test-get-user")
				tagNum := sharedtypes.TagNumber(42)
				err := testutils.InsertUser(t, env.DB, userID, sharedtypes.UserRoleRattler) // Assuming a default role
				if err != nil {
					t.Fatalf("Failed to insert pre-existing user via testutils.InsertUser: %v", err)
				}
				// Optionally, set the tag number if needed for the user data
				// Note: InsertUser might not directly support tag number, adjust as per your testutils.InsertUser
				// If your UserData model requires a tag number at creation, you might need to use UserService.CreateUser
				// For this test, we'll assume InsertUser is sufficient for existence.
				_, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, userID, &tagNum)
				if createErr != nil {
					log.Printf("Warning: Could not create user with tag number in setup, assuming user exists: %v", createErr)
				}

				return nil // Return initial state if any
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("test-get-user")
				payload := userevents.GetUserRequestPayload{
					UserID: userID,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.GetUserRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.GetUserResponse},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("test-get-user")
				msgs := receivedMsgs[userevents.GetUserResponse]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserResponse message, got %d", len(msgs))
				}
				var payload userevents.GetUserResponsePayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.User == nil {
					t.Fatalf("Expected user data in payload, but got nil")
				}
				if payload.User.UserID != userID {
					t.Errorf("Expected UserID %q, got %q", userID, payload.User.UserID)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Failure - user not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No user created, ensuring user is not found
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("non-existent-user")
				payload := userevents.GetUserRequestPayload{
					UserID: userID,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.GetUserRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.GetUserFailed},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("non-existent-user")
				msgs := receivedMsgs[userevents.GetUserFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserFailed message, got %d", len(msgs))
				}
				var payload userevents.GetUserFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != userID {
					t.Errorf("Expected UserID %q, got %q", userID, payload.UserID)
				}
				if payload.Reason != "user not found" {
					t.Errorf("Expected reason 'user not found', got %q", payload.Reason)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
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
					tc.validateFn(t, deps, env, incomingMsg, receivedMsgs, initialState)
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}

// TestHandleGetUserRoleRequest is an integration test for the HandleGetUserRoleRequest handler.
func TestHandleGetUserRoleRequest(t *testing.T) {
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
			name: "Success - role found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Create a test user with a role
				userID := sharedtypes.DiscordID("test-role-user")
				tagNum := sharedtypes.TagNumber(55)
				// Create the user first
				createResult, createErr := deps.UserModule.UserService.CreateUser(env.Ctx, userID, &tagNum)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to create test user for role test: %v, result: %+v", createErr, createResult.Failure)
				}

				// Set user role (assuming UpdateUserRoleInDatabase exists and works)
				roleResult, err := deps.UserModule.UserService.UpdateUserRoleInDatabase(env.Ctx, userID, sharedtypes.UserRoleAdmin)
				if err != nil {
					t.Fatalf("Failed to set user role: %v", err)
				}
				if roleResult.Failure != nil {
					t.Fatalf("Failed to set user role: %v", roleResult.Failure)
				}
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("test-role-user")
				payload := userevents.GetUserRoleRequestPayload{
					UserID: userID,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.GetUserRoleRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.GetUserRoleResponse},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("test-role-user")
				msgs := receivedMsgs[userevents.GetUserRoleResponse]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserRoleResponse message, got %d", len(msgs))
				}
				var payload userevents.GetUserRoleResponsePayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != userID {
					t.Errorf("Expected UserID %q, got %q", userID, payload.UserID)
				}
				if payload.Role != sharedtypes.UserRoleAdmin {
					t.Errorf("Expected Role %q, got %q", sharedtypes.UserRoleAdmin, payload.Role)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Failure - user for role not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// No user created, ensuring user is not found
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("non-existent-role-user")
				payload := userevents.GetUserRoleRequestPayload{
					UserID: userID,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				// Use testutils.PublishMessage
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.GetUserRoleRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{userevents.GetUserRoleFailed},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := sharedtypes.DiscordID("non-existent-role-user")
				msgs := receivedMsgs[userevents.GetUserRoleFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserRoleFailed message, got %d", len(msgs))
				}
				var payload userevents.GetUserRoleFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != userID {
					t.Errorf("Expected UserID %q, got %q", userID, payload.UserID)
				}
				if payload.Reason != "user not found" {
					t.Errorf("Expected reason 'user not found', got %q", payload.Reason)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != triggerMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
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
					tc.validateFn(t, deps, env, incomingMsg, receivedMsgs, initialState)
				},
				ExpectError:    tc.expectHandlerError,
				MessageTimeout: tc.timeout,
			}
			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
