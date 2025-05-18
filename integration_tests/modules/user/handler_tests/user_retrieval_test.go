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

func TestHandleGetUserRequest(t *testing.T) {
	deps := SetupTestUserHandler(t, testEnv)
	defer CleanupHandlerTestDeps(deps)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps)
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - user found",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.ResetJetStreamState(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS: %v", err)
				}

				// Create a test user
				tagNum := sharedtypes.TagNumber(42)
				result, err := deps.UserModule.UserService.CreateUser(deps.Ctx, "test-get-user", &tagNum)
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				if result.Failure != nil {
					t.Fatalf("Failed to create test user: %s", result.Failure.(*userevents.UserCreationFailedPayload).Reason)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.GetUserRequestPayload{
					UserID: "test-get-user",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.GetUserRequest)
				if err := deps.EventBus.Publish(userevents.GetUserRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.GetUserResponse]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserResponse message, got %d", len(msgs))
				}
				var payload userevents.GetUserResponsePayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.User.UserID != "test-get-user" {
					t.Errorf("Expected UserID 'test-get-user', got %q", payload.User.UserID)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.GetUserResponse},
		},
		{
			name: "Failure - user not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.ResetJetStreamState(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS: %v", err)
				}
				// No user created
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.GetUserRequestPayload{
					UserID: "non-existent-user",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.GetUserRequest)
				if err := deps.EventBus.Publish(userevents.GetUserRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.GetUserFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserFailed message, got %d", len(msgs))
				}
				var payload userevents.GetUserFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "non-existent-user" {
					t.Errorf("Expected UserID 'non-existent-user', got %q", payload.UserID)
				}
				if payload.Reason != "user not found" {
					t.Errorf("Expected reason 'user not found', got %q", payload.Reason)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.GetUserFailed},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFn(t, deps)

			receivedMsgs := make(map[string][]*message.Message)
			subCtx, cancel := context.WithTimeout(deps.Ctx, 5*time.Second)
			defer cancel()

			for _, topic := range tc.expectedOutgoingTopics {
				sub, err := deps.EventBus.Subscribe(subCtx, topic)
				if err != nil {
					t.Fatalf("Subscribe error: %v", err)
				}
				go func(topic string, ch <-chan *message.Message) {
					for msg := range ch {
						receivedMsgs[topic] = append(receivedMsgs[topic], msg)
						msg.Ack()
					}
				}(topic, sub)
			}

			incoming := tc.publishMsgFn(t, deps)
			time.Sleep(500 * time.Millisecond)
			tc.validateFn(t, deps, incoming, receivedMsgs)
		})
	}
}

func TestHandleGetUserRoleRequest(t *testing.T) {
	deps := SetupTestUserHandler(t, testEnv)
	defer CleanupHandlerTestDeps(deps)

	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps)
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message)
		expectedOutgoingTopics []string
	}{
		{
			name: "Success - role found",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.ResetJetStreamState(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS: %v", err)
				}

				// Create a test user with a role
				tagNum := sharedtypes.TagNumber(55)
				result, err := deps.UserModule.UserService.CreateUser(deps.Ctx, "test-role-user", &tagNum)
				if err != nil {
					t.Fatalf("Failed to create test user: %v", err)
				}
				if result.Failure != nil {
					t.Fatalf("Failed to create test user: %s", result.Failure.(*userevents.UserCreationFailedPayload).Reason)
				}

				// Set user role (assuming you have a method to do this)
				roleResult, err := deps.UserModule.UserService.UpdateUserRoleInDatabase(deps.Ctx, "test-role-user", sharedtypes.UserRoleAdmin)
				if err != nil {
					t.Fatalf("Failed to set user role: %v", err)
				}
				if roleResult.Failure != nil {
					t.Fatalf("Failed to set user role: %v", roleResult.Failure)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.GetUserRoleRequestPayload{
					UserID: "test-role-user",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.GetUserRoleRequest)
				if err := deps.EventBus.Publish(userevents.GetUserRoleRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.GetUserRoleResponse]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserRoleResponse message, got %d", len(msgs))
				}
				var payload userevents.GetUserRoleResponsePayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "test-role-user" {
					t.Errorf("Expected UserID 'test-role-user', got %q", payload.UserID)
				}
				if payload.Role != sharedtypes.UserRoleAdmin {
					t.Errorf("Expected Role 'admin', got %q", payload.Role)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.GetUserRoleResponse},
		},
		{
			name: "Failure - user for role not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.ResetJetStreamState(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.GetUserRoleRequestPayload{
					UserID: "non-existent-role-user",
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.GetUserRoleRequest)
				if err := deps.EventBus.Publish(userevents.GetUserRoleRequest, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.GetUserRoleFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 GetUserRoleFailed message, got %d", len(msgs))
				}
				var payload userevents.GetUserRoleFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "non-existent-role-user" {
					t.Errorf("Expected UserID 'non-existent-role-user', got %q", payload.UserID)
				}
				if payload.Reason != "user not found" {
					t.Errorf("Expected reason 'user not found', got %q", payload.Reason)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.GetUserRoleFailed},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupFn(t, deps)

			receivedMsgs := make(map[string][]*message.Message)
			subCtx, cancel := context.WithTimeout(deps.Ctx, 5*time.Second)
			defer cancel()

			for _, topic := range tc.expectedOutgoingTopics {
				sub, err := deps.EventBus.Subscribe(subCtx, topic)
				if err != nil {
					t.Fatalf("Subscribe error: %v", err)
				}
				go func(topic string, ch <-chan *message.Message) {
					for msg := range ch {
						receivedMsgs[topic] = append(receivedMsgs[topic], msg)
						msg.Ack()
					}
				}(topic, sub)
			}

			incoming := tc.publishMsgFn(t, deps)
			time.Sleep(500 * time.Millisecond)
			tc.validateFn(t, deps, incoming, receivedMsgs)
		})
	}
}
