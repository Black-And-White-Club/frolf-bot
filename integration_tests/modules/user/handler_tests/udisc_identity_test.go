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

func TestHandleUpdateUDiscIdentityRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		expectHandlerError     bool
		timeout                time.Duration
	}{
		{
			name: "Success - Update UDisc Identity",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Pre-create user
				userID := sharedtypes.DiscordID("user-udisc-123")
				guildID := sharedtypes.GuildID("test-guild")
				_, err := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, nil, nil, nil)
				if err != nil {
					t.Fatalf("Failed to pre-create user: %v", err)
				}
				return userID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("user-udisc-123")
				username := "test_udisc_user"
				name := "Test UDisc Name"
				payload := userevents.UpdateUDiscIdentityRequestedPayloadV1{
					GuildID:  "test-guild",
					UserID:   userID,
					Username: &username,
					Name:     &name,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UpdateUDiscIdentityRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := initialState.(sharedtypes.DiscordID)

				// Verify successful response
				msgs := receivedMsgs[userevents.UDiscIdentityUpdatedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UDiscIdentityUpdated message, got %d", len(msgs))
				}

				var payload userevents.UDiscIdentityUpdatedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}

				if payload.UserID != userID {
					t.Errorf("UserID mismatch: expected %q, got %q", userID, payload.UserID)
				}
				if payload.Username == nil || *payload.Username != "test_udisc_user" {
					t.Errorf("Username mismatch: expected 'test_udisc_user', got %v", payload.Username)
				}

				// Verify DB state
				guildID := sharedtypes.GuildID("test-guild")
				getUserResult, err := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				if err != nil {
					t.Fatalf("Failed to get user: %v", err)
				}
				if !getUserResult.IsSuccess() {
					t.Fatalf("Expected user to exist, but got failure: %+v", getUserResult.Failure)
				}

				user := *getUserResult.Success
				if user.UDiscUsername == nil || *user.UDiscUsername != "test_udisc_user" {
					t.Errorf("DB UDisc username mismatch: expected 'test_udisc_user', got %v", user.UDiscUsername)
				}
			},
			expectedOutgoingTopics: []string{userevents.UDiscIdentityUpdatedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name: "Success - Update UDisc Name Only",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				userID := sharedtypes.DiscordID("user-udisc-name-only")
				guildID := sharedtypes.GuildID("test-guild")
				_, err := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, nil, nil, nil)
				if err != nil {
					t.Fatalf("Failed to pre-create user: %v", err)
				}
				return userID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("user-udisc-name-only")
				name := "Only Name"
				payload := userevents.UpdateUDiscIdentityRequestedPayloadV1{
					GuildID: "test-guild",
					UserID:  userID,
					Name:    &name,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UpdateUDiscIdentityRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[userevents.UDiscIdentityUpdatedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UDiscIdentityUpdated message, got %d", len(msgs))
				}
				var payload userevents.UDiscIdentityUpdatedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.Name == nil || *payload.Name != "Only Name" {
					t.Errorf("Name mismatch: expected 'Only Name', got %v", payload.Name)
				}
				if payload.Username != nil {
					t.Errorf("Expected nil username, got %v", payload.Username)
				}
			},
			expectedOutgoingTopics: []string{userevents.UDiscIdentityUpdatedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
		{
			name: "Failure - User Not Found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("non-existent-user")
				username := "test_udisc_user"
				payload := userevents.UpdateUDiscIdentityRequestedPayloadV1{
					GuildID:  "test-guild",
					UserID:   userID,
					Username: &username,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UpdateUDiscIdentityRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify failure response
				msgs := receivedMsgs[userevents.UDiscIdentityUpdateFailedV1]
				if len(msgs) != 1 {
					t.Fatalf("Expected 1 UDiscIdentityUpdateFailed message, got %d", len(msgs))
				}

				var payload userevents.UDiscIdentityUpdateFailedPayloadV1
				if err := deps.UserModule.Helper.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal failure payload: %v", err)
				}

				if payload.Reason == "" {
					t.Error("Expected non-empty failure reason")
				}
			},
			expectedOutgoingTopics: []string{userevents.UDiscIdentityUpdateFailedV1},
			expectHandlerError:     false,
			timeout:                5 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestUserHandler(t)
			genericCase := testutils.TestCase{
				Name: tc.name,
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
