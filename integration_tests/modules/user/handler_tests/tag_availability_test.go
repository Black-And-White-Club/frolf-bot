package handler_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleTagAvailable(t *testing.T) {
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
			name: "Success - user created from tag available",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				if err := testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB); err != nil {
					t.Fatalf("Failed to clean DB: %v", err)
				}
				if err := deps.ResetJetStreamState(deps.Ctx, "user"); err != nil {
					t.Fatalf("Failed to clean NATS: %v", err)
				}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.TagAvailablePayload{
					UserID:    "test-tag-user-available",
					TagNumber: 21,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.TagAvailable)
				if err := deps.EventBus.Publish(userevents.TagAvailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.UserCreated]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreated message, got %d", len(msgs))
				}
				var payload userevents.UserCreatedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "test-tag-user-available" {
					t.Errorf("Expected UserID 'test-tag-user-available', got %q", payload.UserID)
				}
				if payload.TagNumber == nil || *payload.TagNumber != 21 {
					t.Errorf("Expected TagNumber 21, got %v", payload.TagNumber)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.UserCreated},
		},
		{
			name: "Failure - user already exists",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB)
				deps.ResetJetStreamState(deps.Ctx, "user")
				_, _ = deps.UserModule.UserService.CreateUser(deps.Ctx, "existing-tag-user", nil)
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
				payload := userevents.TagAvailablePayload{
					UserID:    "existing-tag-user",
					TagNumber: 22,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Marshal error: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), data)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("topic", userevents.TagAvailable)
				if err := deps.EventBus.Publish(userevents.TagAvailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.UserCreationFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreationFailed message, got %d", len(msgs))
				}
				var payload userevents.UserCreationFailedPayload
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Unmarshal error: %v", err)
				}
				if payload.UserID != "existing-tag-user" {
					t.Errorf("Expected UserID 'existing-tag-user', got %q", payload.UserID)
				}
				if payload.TagNumber == nil || *payload.TagNumber != 22 {
					t.Errorf("Expected TagNumber 22, got %v", payload.TagNumber)
				}
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
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

func TestHandleTagUnavailable(t *testing.T) {
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
			name: "Always fails with 'tag not available'",
			setupFn: func(t *testing.T, deps HandlerTestDeps) {
				testutils.CleanUserIntegrationTables(deps.Ctx, deps.TestEnvironment.DB)
				deps.ResetJetStreamState(deps.Ctx, "user")
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps) *message.Message {
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
				msg.Metadata.Set("topic", userevents.TagUnavailable)
				if err := deps.EventBus.Publish(userevents.TagUnavailable, msg); err != nil {
					t.Fatalf("Publish error: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message) {
				msgs := receivedMsgs[userevents.UserCreationFailed]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UserCreationFailed message, got %d", len(msgs))
				}
				var payload userevents.UserCreationFailedPayload
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
				if msgs[0].Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectedOutgoingTopics: []string{userevents.UserCreationFailed},
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
