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

func TestHandleUserProfileUpdated(t *testing.T) {
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
			name: "Success - User Profile Updated",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				// Pre-create user
				userID := sharedtypes.DiscordID("testuser-profile-123")
				guildID := sharedtypes.GuildID("test-guild")
				_, err := deps.UserModule.UserService.CreateUser(env.Ctx, guildID, userID, nil, nil, nil)
				if err != nil {
					t.Fatalf("Failed to pre-create user: %v", err)
				}
				return userID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				userID := sharedtypes.DiscordID("testuser-profile-123")
				payload := userevents.UserProfileUpdatedPayloadV1{
					UserID:      userID,
					DisplayName: "New Display Name",
					AvatarHash:  "new-avatar-hash",
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, userevents.UserProfileUpdatedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				userID := initialState.(sharedtypes.DiscordID)
				guildID := sharedtypes.GuildID("test-guild")

				// Wait for async processing
				time.Sleep(500 * time.Millisecond)

				// Verify DB state
				getUserResult, err := deps.UserModule.UserService.GetUser(env.Ctx, guildID, userID)
				if err != nil {
					t.Fatalf("Failed to get user: %v", err)
				}
				if !getUserResult.IsSuccess() {
					t.Fatalf("Expected user to exist, but got failure: %+v", getUserResult.Failure)
				}

				user := *getUserResult.Success
				if user.DisplayName != "New Display Name" {
					t.Errorf("Display name mismatch: expected 'New Display Name', got %q", user.DisplayName)
				}
				if user.AvatarHash == nil || *user.AvatarHash != "new-avatar-hash" {
					t.Errorf("Avatar hash mismatch: expected 'new-avatar-hash', got %v", user.AvatarHash)
				}
			},
			expectedOutgoingTopics: []string{}, // No outgoing events for this handler
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
