package guildhandlerintegrationtests

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleRetrieveGuildConfig(t *testing.T) {
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
			name: "Success - Retrieve existing config",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				guildID := sharedtypes.GuildID("723456789012345678")
				config := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "734567890123456789",
					EventChannelID:       "745678901234567890",
					LeaderboardChannelID: "756789012345678901",
					UserRoleID:           "767890123456789012",
					SignupEmoji:          "✅",
				}
				createResult, createErr := deps.GuildModule.GuildService.CreateGuildConfig(env.Ctx, config)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create config for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				return guildID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				guildID := sharedtypes.GuildID("723456789012345678")
				payload := guildevents.GuildConfigRetrievalRequestedPayloadV1{
					GuildID: guildID,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, guildevents.GuildConfigRetrievalRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{guildevents.GuildConfigRetrievedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				guildID := initialState.(sharedtypes.GuildID)

				// Verify GuildConfigRetrieved event was published
				expectedTopic := guildevents.GuildConfigRetrievedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var successPayload guildevents.GuildConfigRetrievedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal success payload: %v", err)
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Expected GuildID %s, got %s", guildID, successPayload.GuildID)
				}

				if successPayload.Config.SignupChannelID != "734567890123456789" {
					t.Errorf("Expected SignupChannelID %s, got %s", "734567890123456789", successPayload.Config.SignupChannelID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Failure - Config not found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				guildID := sharedtypes.GuildID("623456789012345678")
				payload := guildevents.GuildConfigRetrievalRequestedPayloadV1{
					GuildID: guildID,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, guildevents.GuildConfigRetrievalRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{guildevents.GuildConfigRetrievalFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// Verify failure event was published
				expectedTopic := guildevents.GuildConfigRetrievalFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var failurePayload guildevents.GuildConfigRetrievalFailedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &failurePayload); err != nil {
					t.Fatalf("Failed to unmarshal failure payload: %v", err)
				}

				guildID := sharedtypes.GuildID("623456789012345678")
				if failurePayload.GuildID != guildID {
					t.Errorf("Expected GuildID %s, got %s", guildID, failurePayload.GuildID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestGuildHandler(t)

			env := deps.TestEnvironment

			genericCase := testutils.TestCase{
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tt.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tt.publishMsgFn(t, deps, env)
				},
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tt.validateFn(t, deps, env, triggerMsg, receivedMsgs, initialState)
				},
				ExpectedTopics: tt.expectedOutgoingTopics,
				ExpectError:    tt.expectHandlerError,
				MessageTimeout: tt.timeout,
			}

			testutils.RunTest(t, genericCase, env)
		})
	}
}

func TestHandleUpdateGuildConfig(t *testing.T) {
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
			name: "Success - Update existing config",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				guildID := sharedtypes.GuildID("523456789012345678")
				config := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "534567890123456789",
					EventChannelID:       "545678901234567890",
					LeaderboardChannelID: "556789012345678901",
					UserRoleID:           "567890123456789012",
					SignupEmoji:          "✅",
				}
				createResult, createErr := deps.GuildModule.GuildService.CreateGuildConfig(env.Ctx, config)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create config for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				return guildID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				guildID := sharedtypes.GuildID("523456789012345678")
				newEmoji := "🎯"
				payload := guildevents.GuildConfigUpdateRequestedPayloadV1{
					GuildID:     guildID,
					SignupEmoji: newEmoji,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, guildevents.GuildConfigUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{guildevents.GuildConfigUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				guildID := initialState.(sharedtypes.GuildID)

				// Verify config was updated in database
				var updatedConfig *guildtypes.GuildConfig
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					getResult, getErr := deps.GuildModule.GuildService.GetGuildConfig(env.Ctx, guildID)
					if getErr != nil {
						return fmt.Errorf("service returned error: %w", getErr)
					}
					if getResult.Success == nil {
						return errors.New("config not found yet or success payload is nil")
					}

					successPayload, ok := getResult.Success.(*guildevents.GuildConfigRetrievedPayloadV1)
					if !ok {
						return errors.New("success payload is not of type GuildConfigRetrievedPayloadV1")
					}

					updatedConfig = &successPayload.Config
					if updatedConfig.SignupEmoji != "🎯" {
						return errors.New("config not updated yet")
					}
					return nil
				})
				if err != nil {
					t.Fatalf("Config not updated in database after waiting: %v", err)
				}

				// Verify GuildConfigUpdated event was published
				expectedTopic := guildevents.GuildConfigUpdatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var successPayload guildevents.GuildConfigUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal success payload: %v", err)
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Expected GuildID %s, got %s", guildID, successPayload.GuildID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch")
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestGuildHandler(t)

			env := deps.TestEnvironment

			genericCase := testutils.TestCase{
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tt.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tt.publishMsgFn(t, deps, env)
				},
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tt.validateFn(t, deps, env, triggerMsg, receivedMsgs, initialState)
				},
				ExpectedTopics: tt.expectedOutgoingTopics,
				ExpectError:    tt.expectHandlerError,
				MessageTimeout: tt.timeout,
			}

			testutils.RunTest(t, genericCase, env)
		})
	}
}
