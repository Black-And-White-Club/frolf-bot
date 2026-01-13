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

func TestHandleScoreUpdateValidated(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Update Participant Score (Single Participant)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil}})

				score := sharedtypes.Score(-1)
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &score)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundParticipantScoreUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundParticipantScoreUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundParticipantScoreUpdatedV1)
				}
				var result roundevents.ParticipantScoreUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - Update Participant Score (Multiple Participants)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: scorePtr(2)},
					{UserID: data4.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				score := sharedtypes.Score(3)
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &score)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundParticipantScoreUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundParticipantScoreUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundParticipantScoreUpdatedV1)
				}
				var result roundevents.ParticipantScoreUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - Update Score for Participant with Existing Score",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				existingScore := sharedtypes.Score(1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &existingScore}})

				newScore := sharedtypes.Score(-2)
				payload := createScoreUpdateValidatedPayload(roundID, data2.UserID, &newScore)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundParticipantScoreUpdatedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundParticipantScoreUpdatedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundParticipantScoreUpdatedV1)
				}
				var result roundevents.ParticipantScoreUpdatedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &result); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if result.Score == 0 && result.Participants == nil {
					t.Error("Expected updated score and participants to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Failure - Round Not Found",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				nonExistentRoundID := sharedtypes.RoundID(uuid.New())
				score := sharedtypes.Score(0)
				payload := createScoreUpdateValidatedPayload(nonExistentRoundID, data.UserID, &score)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundScoreUpdateErrorV1)
				}
				var payload roundevents.RoundScoreUpdateErrorPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.Error == "" {
					t.Error("Expected error message to be populated")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Failure - Participant Not Found in Round",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: nil}})
				score := sharedtypes.Score(1)
				payload := createScoreUpdateValidatedPayload(roundID, data3.UserID, &score)
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoreUpdateErrorV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoreUpdateErrorV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundScoreUpdateErrorV1)
				}
				var payload roundevents.RoundScoreUpdateErrorPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.Error == "" {
					t.Error("Expected error message to be populated")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
				invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundScoreUpdateValidatedV1, invalidMsg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return invalidMsg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				if len(receivedMsgs) != 0 {
					t.Errorf("Expected no messages for invalid JSON, got %d topics", len(receivedMsgs))
				}
			},
			timeout: 500 * time.Millisecond,
		},
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

// Helper functions for creating payloads - UNIQUE TO SCORE UPDATE VALIDATED TESTS
func createScoreUpdateValidatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score *sharedtypes.Score) roundevents.ScoreUpdateValidatedPayloadV1 {
	return roundevents.ScoreUpdateValidatedPayloadV1{
		GuildID: sharedtypes.GuildID("test-guild"),
		ScoreUpdateRequestPayload: roundevents.ScoreUpdateRequestPayloadV1{
			GuildID:   sharedtypes.GuildID("test-guild"),
			RoundID:   roundID,
			UserID:    participant,
			Score:     score,
			ChannelID: "test-channel",
			MessageID: "test-message",
		},
	}
}

// NOTE: MessageCapture-based helpers removed. Use the per-test
// ValidateFn with the receivedMsgs map and deps.TestHelpers.UnmarshalPayload
// to inspect and validate outgoing messages.

// Validation functions - UNIQUE TO SCORE UPDATE VALIDATED TESTS
// (Use deps.TestHelpers.UnmarshalPayload within ValidateFn)

// (No direct replacements provided here; per-test ValidateFn should perform
// any message inspections using receivedMsgs and deps.TestHelpers.UnmarshalPayload.)

// Helper utility functions
func scorePtr(score int) *sharedtypes.Score {
	s := sharedtypes.Score(score)
	return &s
}
