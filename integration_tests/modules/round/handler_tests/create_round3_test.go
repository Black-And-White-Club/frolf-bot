package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

// createValidRoundMessageIDUpdatePayload creates a valid RoundMessageIDUpdatePayload for testing
func createValidRoundMessageIDUpdatePayload(roundID sharedtypes.RoundID) roundevents.RoundMessageIDUpdatePayloadV1 {
	return roundevents.RoundMessageIDUpdatePayloadV1{
		RoundID: roundID,
		GuildID: "test-guild",
	}
}

// TestHandleRoundEventMessageIDUpdate runs integration tests for the round event message ID update handler
func TestHandleRoundEventMessageIDUpdate(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name:    "Success - Update Valid Round Message ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				// Create a round in DB for this test
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundInDB(t, deps.DB, data.UserID)

				payload := createValidRoundMessageIDUpdatePayload(roundID)
				discordMessageID := "discord_msg_123456"

				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}

				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				msg.Metadata.Set("discord_message_id", discordMessageID)

				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundEventMessageIDUpdateV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundEventMessageIDUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundEventMessageIDUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundEventMessageIDUpdatedV1)
				}
				var payload roundevents.RoundScheduledPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.EventMessageID != "discord_msg_123456" {
					t.Errorf("Discord message ID mismatch: expected %s, got %s", "discord_msg_123456", payload.EventMessageID)
				}
			},
			timeout: 500 * time.Millisecond,
		},
		// Other test cases (failures and correlation id preservation) can be added following the same pattern.
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			genericCase := testutils.TestCase{
				Name: tc.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return tc.setupFn(t, deps, env)
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					return tc.publishMsgFn(t, deps, env)
				},
				ExpectedTopics: tc.expectedOutgoingTopics,
				ValidateFn: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
					tc.validateFn(t, deps, env, triggerMsg, receivedMsgs, initialState)
				},
				ExpectError:    false,
				MessageTimeout: tc.timeout,
			}

			testutils.RunTest(t, genericCase, deps.TestEnvironment)
		})
	}
}
