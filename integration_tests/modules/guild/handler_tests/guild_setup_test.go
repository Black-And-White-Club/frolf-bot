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

func TestHandleGuildSetup(t *testing.T) {
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
			name: "Success - Guild Setup creates config",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				guildID := sharedtypes.GuildID("823456789012345678")
				payload := guildevents.GuildConfigCreationRequestedPayloadV1{
					GuildID:              guildID,
					SignupChannelID:      "834567890123456789",
					EventChannelID:       "845678901234567890",
					LeaderboardChannelID: "856789012345678901",
					UserRoleID:           "867890123456789012",
					SignupEmoji:          "✅",
					AutoSetupCompleted:   true,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, guildevents.GuildSetupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{guildevents.GuildConfigCreatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				guildID := sharedtypes.GuildID("823456789012345678")

				// Verify config was created in database
				var config *guildtypes.GuildConfig
				err := testutils.WaitFor(5*time.Second, 100*time.Millisecond, func() error {
					getResult, getErr := deps.GuildModule.GuildService.GetGuildConfig(env.Ctx, guildID)
					if getErr != nil {
						return fmt.Errorf("service returned error: %w", getErr)
					}
					if getResult.Success == nil {
						return errors.New("config not found yet or success payload is nil")
					}
					
					successPayload, ok := getResult.Success.(*guildevents.GuildConfigRetrievedPayload)
					if !ok {
						return errors.New("success payload is not of type GuildConfigRetrievedPayload")
					}
					
					config = &successPayload.Config
					return nil
				})
				if err != nil {
					t.Fatalf("Config not found in database after waiting: %v", err)
				}

				if config.SignupChannelID != "834567890123456789" {
					t.Errorf("Expected SignupChannelID %s, got %s", "834567890123456789", config.SignupChannelID)
				}

				// Verify GuildConfigCreated event was published
				expectedTopic := guildevents.GuildConfigCreatedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var successPayload guildevents.GuildConfigCreatedPayload
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &successPayload); err != nil {
					t.Fatalf("Failed to unmarshal success payload: %v", err)
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Expected GuildID %s, got %s", guildID, successPayload.GuildID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q", 
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
		{
			name: "Failure - Guild Setup with existing config",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				guildID := sharedtypes.GuildID("923456789012345678")
				existingConfig := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "934567890123456789",
					EventChannelID:       "945678901234567890",
					LeaderboardChannelID: "956789012345678901",
					UserRoleID:           "967890123456789012",
					SignupEmoji:          "👍",
				}
				createResult, createErr := deps.GuildModule.GuildService.CreateGuildConfig(env.Ctx, existingConfig)
				if createErr != nil || createResult.Success == nil {
					t.Fatalf("Failed to pre-create config for test setup: %v, result: %+v", createErr, createResult.Failure)
				}
				return guildID
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				guildID := sharedtypes.GuildID("923456789012345678")
				payload := guildevents.GuildConfigCreationRequestedPayloadV1{
					GuildID:              guildID,
					SignupChannelID:      "944567890123456789",
					EventChannelID:       "955678901234567890",
					LeaderboardChannelID: "966789012345678901",
					UserRoleID:           "977890123456789012",
					SignupEmoji:          "🎯",
					AutoSetupCompleted:   true,
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, guildevents.GuildSetupRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{guildevents.GuildConfigCreationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, incomingMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				guildID := initialState.(sharedtypes.GuildID)

				// Verify failure event was published
				expectedTopic := guildevents.GuildConfigCreationFailedV1
				msgs := receivedMsgs[expectedTopic]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", expectedTopic)
				}
				if len(msgs) > 1 {
					t.Errorf("Expected exactly one message on topic %q, but received %d", expectedTopic, len(msgs))
				}

				receivedMsg := msgs[0]
				var failurePayload guildevents.GuildConfigCreationFailedPayload
				if err := deps.TestHelpers.UnmarshalPayload(receivedMsg, &failurePayload); err != nil {
					t.Fatalf("Failed to unmarshal failure payload: %v", err)
				}

				if failurePayload.GuildID != guildID {
					t.Errorf("Expected GuildID %s, got %s", guildID, failurePayload.GuildID)
				}

				// Verify original config is unchanged
				getResult, getErr := deps.GuildModule.GuildService.GetGuildConfig(env.Ctx, guildID)
				if getErr != nil {
					t.Fatalf("Expected GetGuildConfig to succeed for existing config, but got error: %v", getErr)
				}
				if getResult.Success == nil {
					t.Fatalf("Expected GetGuildConfig to return success payload, but got nil. Failure: %+v", getResult.Failure)
				}
				
				successPayload, ok := getResult.Success.(*guildevents.GuildConfigRetrievedPayload)
				if !ok {
					t.Fatalf("Success payload is not of type GuildConfigRetrievedPayload")
				}
				
				existingConfig := &successPayload.Config
				if existingConfig.SignupChannelID != "934567890123456789" {
					t.Errorf("Original config was modified. Expected SignupChannelID %s, got %s", "934567890123456789", existingConfig.SignupChannelID)
				}

				// Verify correlation ID is propagated
				if receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) != incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey) {
					t.Errorf("Correlation ID mismatch: expected %q, got %q",
						incomingMsg.Metadata.Get(middleware.CorrelationIDMetadataKey),
						receivedMsg.Metadata.Get(middleware.CorrelationIDMetadataKey))
				}

				// Verify no success event was published
				unexpectedTopic := guildevents.GuildConfigCreatedV1
				if len(receivedMsgs[unexpectedTopic]) > 0 {
					t.Errorf("Expected no messages on topic %q, but received %d", unexpectedTopic, len(receivedMsgs[unexpectedTopic]))
				}
			},
			expectHandlerError: false,
			timeout:            5 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, cleanup := SetupTestGuildHandler(t)
			defer cleanup()

			env := deps.TestEnvironment

			// Convert to testutils.TestCase
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
