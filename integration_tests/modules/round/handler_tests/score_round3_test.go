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

func TestHandleParticipantScoreUpdated(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - All Scores Submitted (Single Participant)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
				})

				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundAllScoresSubmittedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundAllScoresSubmittedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundAllScoresSubmittedV1)
				}
				var payload roundevents.AllScoresSubmittedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - All Scores Submitted (Multiple Participants)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(-2)
				score2 := sharedtypes.Score(1)
				score3 := sharedtypes.Score(0)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				payload := createParticipantScoreUpdatedPayload(roundID, data3.UserID, score2, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
				})
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundAllScoresSubmittedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundAllScoresSubmittedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundAllScoresSubmittedV1)
				}
				var payload roundevents.AllScoresSubmittedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - Not All Scores Submitted (Single Participant Without Score)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(2)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})

				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
				})
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoresPartiallySubmittedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoresPartiallySubmittedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundScoresPartiallySubmittedV1)
				}
				var payload roundevents.ScoresPartiallySubmittedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.Participant == "" {
					t.Error("Expected Participant to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - Not All Scores Submitted (Multiple Missing Scores)",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				data5 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(-1)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data5.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})

				payload := createParticipantScoreUpdatedPayload(roundID, data2.UserID, score1, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: nil},
					{UserID: data5.UserID, Response: roundtypes.ResponseTentative, Score: nil},
				})
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundScoresPartiallySubmittedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundScoresPartiallySubmittedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundScoresPartiallySubmittedV1)
				}
				var payload roundevents.ScoresPartiallySubmittedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 500 * time.Millisecond,
		},
		{
			name: "Success - Last Score Submitted Triggers All Scores Submitted",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				data2 := NewTestData()
				data3 := NewTestData()
				data4 := NewTestData()
				helper := testutils.NewRoundTestHelper(nil, nil)
				score1 := sharedtypes.Score(1)
				score2 := sharedtypes.Score(-1)
				score3 := sharedtypes.Score(0)
				roundID := helper.CreateRoundWithParticipants(t, deps.DB, data.UserID, []testutils.ParticipantData{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
				})

				payload := createParticipantScoreUpdatedPayload(roundID, data4.UserID, score3, []roundtypes.Participant{
					{UserID: data2.UserID, Response: roundtypes.ResponseAccept, Score: &score1},
					{UserID: data3.UserID, Response: roundtypes.ResponseAccept, Score: &score2},
					{UserID: data4.UserID, Response: roundtypes.ResponseAccept, Score: &score3},
				})
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundAllScoresSubmittedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundAllScoresSubmittedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundAllScoresSubmittedV1)
				}
				var payload roundevents.AllScoresSubmittedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
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
				payload := createParticipantScoreUpdatedPayload(nonExistentRoundID, data.UserID, score, []roundtypes.Participant{})
				b, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), b)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundFinalizationFailedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundFinalizationFailedV1]
				if len(msgs) == 0 {
					t.Fatalf("Expected at least one message on topic %q", roundevents.RoundFinalizationFailedV1)
				}
				var payload roundevents.RoundFinalizationFailedPayloadV1
				if err := deps.TestHelpers.UnmarshalPayload(msgs[0], &payload); err != nil {
					t.Fatalf("Failed to unmarshal payload: %v", err)
				}
				if payload.RoundID == sharedtypes.RoundID(uuid.Nil) {
					t.Error("Expected RoundID to be set")
				}
			},
			timeout: 700 * time.Millisecond,
		},
		{
			name: "Failure - Invalid JSON Message",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} { return nil },
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				invalidMsg := message.NewMessage(uuid.New().String(), []byte("invalid json"))
				invalidMsg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundParticipantScoreUpdatedV1, invalidMsg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return invalidMsg
			},
			expectedOutgoingTopics: []string{},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				// No outgoing topics expected for invalid JSON
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

// Helper functions for creating payloads - UNIQUE TO PARTICIPANT SCORE UPDATED TESTS
func createParticipantScoreUpdatedPayload(roundID sharedtypes.RoundID, participant sharedtypes.DiscordID, score sharedtypes.Score, participants []roundtypes.Participant) roundevents.ParticipantScoreUpdatedPayloadV1 {
	return roundevents.ParticipantScoreUpdatedPayloadV1{
		GuildID:        "test-guild",
		RoundID:        roundID,
		Participant:    participant,
		Score:          score,
		ChannelID:      "test_channel_123",
		EventMessageID: "test-event-message-id",
		Participants:   participants,
		Config:         nil,
	}
}

// NOTE: MessageCapture- and context.Background()-dependent helpers removed.
// Use per-test PublishMsgFn (which should call testutils.PublishMessage with
// env.EventBus and env.Ctx) and perform outgoing message inspection inside the
// TestCase ValidateFn using the receivedMsgs map and
// deps.TestHelpers.UnmarshalPayload(msg, &payload).
