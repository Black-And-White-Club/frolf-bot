package roundhandler_integration_tests

import (
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
)

func TestHandleScoreUpdateRequest(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Valid Score Update Request (Under Par)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				score := sharedtypes.Score(-2)
				payload := createScoreUpdateRequestPayload(roundID, data2.UserID, &score)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateValidatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateValidatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", roundevents.RoundScoreUpdateValidatedV1)
				}
				var payload roundevents.ScoreUpdateValidatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.ScoreUpdateRequestPayload.Score == nil || *payload.ScoreUpdateRequestPayload.Score != sharedtypes.Score(-2) {
					t.Errorf("Expected score -2, got %v", payload.ScoreUpdateRequestPayload.Score)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Valid Score Update Request (Over Par)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				score := sharedtypes.Score(5)
				payload := createScoreUpdateRequestPayload(roundID, data.UserID, &score)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateValidatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateValidatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", roundevents.RoundScoreUpdateValidatedV1)
				}
				var payload roundevents.ScoreUpdateValidatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.ScoreUpdateRequestPayload.Score == nil || *payload.ScoreUpdateRequestPayload.Score != sharedtypes.Score(5) {
					t.Errorf("Expected score 5, got %v", payload.ScoreUpdateRequestPayload.Score)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Success - Valid Score Update Request (Even Par)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				score := sharedtypes.Score(0)
				payload := createScoreUpdateRequestPayload(roundID, data2.UserID, &score)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateValidatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateValidatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", roundevents.RoundScoreUpdateValidatedV1)
				}
				var payload roundevents.ScoreUpdateValidatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.ScoreUpdateRequestPayload.Score == nil || *payload.ScoreUpdateRequestPayload.Score != sharedtypes.Score(0) {
					t.Errorf("Expected score 0, got %v", payload.ScoreUpdateRequestPayload.Score)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Empty Round ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				score := sharedtypes.Score(-1)
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), data.UserID, &score)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one error message on topic %q", roundevents.RoundScoreUpdateErrorV1)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Empty Participant ID",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				score := sharedtypes.Score(2)
				payload := createScoreUpdateRequestPayload(roundID, "", &score)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one error message on topic %q", roundevents.RoundScoreUpdateErrorV1)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Nil Score",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				helper := testutils.NewRoundTestHelper(env.EventBus, nil)
				roundID := helper.CreateRoundInDBWithState(t, deps.DB, data.UserID, roundtypes.RoundStateInProgress)

				// Create payload with nil score
				payload := createScoreUpdateRequestPayload(roundID, data.UserID, nil)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q, but received none", roundevents.RoundScoreUpdateErrorV1)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Multiple Validation Errors",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				payload := createScoreUpdateRequestPayload(sharedtypes.RoundID(uuid.Nil), "", nil)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("Failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one error message on topic %q", roundevents.RoundScoreUpdateErrorV1)
				}
			},
			timeout: 5 * time.Second,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				return nil
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidJSON := []byte(`invalid json`)
				msg := message.NewMessage(uuid.New().String(), invalidJSON)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateRequestedV1, msg); err != nil {
					t.Fatalf("Failed to publish message: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				successMsgs := receivedMsgs[roundevents.RoundScoreUpdateValidatedV1]
				errorMsgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(successMsgs) > 0 || len(errorMsgs) > 0 {
					t.Errorf("Expected no messages for invalid JSON, got success=%d, error=%d", len(successMsgs), len(errorMsgs))
				}
			},
			timeout: 5 * time.Second,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			deps := SetupTestRoundHandler(t)
			// Ensure streams are created
			ensureStreams(t, deps.TestEnvironment)

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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE TESTS
func createScoreUpdateRequestPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateRequestPayloadV1 {
	return roundevents.ScoreUpdateRequestPayloadV1{
		GuildID:   sharedtypes.GuildID("test-guild"),
		RoundID:   roundID,
		UserID:    participant,
		Score:     score,
		ChannelID: "test-channel",
		MessageID: "test-message",
	}
}
