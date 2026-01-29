package roundhandler_integration_tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// createValidAllScoresSubmittedPayload creates a valid AllScoresSubmittedPayloadV1 for testing
func createValidAllScoresSubmittedPayload(roundID sharedtypes.RoundID, participants []roundtypes.Participant, round roundtypes.Round) roundevents.AllScoresSubmittedPayloadV1 {
	return roundevents.AllScoresSubmittedPayloadV1{
		GuildID:        "test-guild",
		RoundID:        roundID,
		EventMessageID: round.EventMessageID,
		RoundData:      round,
		Participants:   participants,
		Config:         nil,
	}
}

// createValidRoundFinalizedPayload creates a valid RoundFinalizedPayload for testing
func createValidRoundFinalizedPayload(roundID sharedtypes.RoundID, roundData roundtypes.Round) roundevents.RoundFinalizedPayloadV1 {
	return roundevents.RoundFinalizedPayloadV1{
		GuildID:   "test-guild",
		RoundID:   roundID,
		RoundData: roundData,
		Config:    nil,
	}
}

// createExistingRoundForFinalization creates and stores a round that can be finalized
func createExistingRoundForFinalization(t *testing.T, userID sharedtypes.DiscordID, db bun.IDB) (sharedtypes.RoundID, []roundtypes.Participant, roundtypes.Round) {
	t.Helper()

	generator := testutils.NewTestDataGenerator(time.Now().UnixNano())
	roundData := generator.GenerateRound(testutils.DiscordID(userID), 0, []testutils.User{})

	tagNumber1 := sharedtypes.TagNumber(1)
	tagNumber2 := sharedtypes.TagNumber(2)
	score1 := sharedtypes.Score(-3)
	score2 := sharedtypes.Score(2)

	participants := []roundtypes.Participant{
		{
			UserID:    userID,
			TagNumber: &tagNumber1,
			Response:  roundtypes.ResponseAccept,
			Score:     &score1,
		},
		{
			UserID:    sharedtypes.DiscordID("123456789012345678"),
			TagNumber: &tagNumber2,
			Response:  roundtypes.ResponseAccept,
			Score:     &score2,
		},
	}

	roundData.Participants = participants

	roundDBRec := &rounddb.Round{
		ID:           roundData.ID,
		Title:        roundData.Title,
		Description:  roundData.Description,
		Location:     roundData.Location,
		EventType:    roundData.EventType,
		StartTime:    *roundData.StartTime,
		Finalized:    roundData.Finalized,
		CreatedBy:    roundData.CreatedBy,
		State:        roundData.State,
		Participants: roundData.Participants,
		GuildID:      "test-guild",
	}

	_, err := db.NewInsert().Model(roundDBRec).Exec(context.Background())
	if err != nil {
		t.Fatalf("Failed to insert test round for finalization: %v", err)
	}

	return roundData.ID, participants, roundData
}

func TestHandleAllScoresSubmitted(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Valid All Scores Submitted",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				roundID, participants, round := createExistingRoundForFinalization(t, data.UserID, env.DB)
				return struct {
					id           sharedtypes.RoundID
					participants []roundtypes.Participant
					round        roundtypes.Round
				}{roundID, participants, round}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				roundID, participants, round := createExistingRoundForFinalization(t, data.UserID, env.DB)
				payload := createValidAllScoresSubmittedPayload(roundID, participants, round)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundAllScoresSubmittedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{roundevents.RoundFinalizedDiscordV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[roundevents.RoundFinalizedDiscordV1]
				if len(msgs) == 0 {
					t.Fatalf("expected discord finalized message, got none")
				}
			},
			timeout: 5 * time.Second,
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

func TestHandleRoundFinalized(t *testing.T) {
	tests := []struct {
		name                   string
		setupFn                func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{}
		publishMsgFn           func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message
		validateFn             func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
		expectedOutgoingTopics []string
		timeout                time.Duration
	}{
		{
			name: "Success - Valid Round Finalized",
			setupFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) interface{} {
				data := NewTestData()
				roundID, _, existingRound := createExistingRoundForFinalization(t, data.UserID, env.DB)
				return struct {
					id sharedtypes.RoundID
					r  roundtypes.Round
				}{roundID, existingRound}
			},
			publishMsgFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment) *message.Message {
				data := NewTestData()
				roundID, _, existingRound := createExistingRoundForFinalization(t, data.UserID, env.DB)
				payload := createValidRoundFinalizedPayload(roundID, existingRound)
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					t.Fatalf("failed to marshal payload: %v", err)
				}
				msg := message.NewMessage(uuid.New().String(), payloadBytes)
				msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
				if err := testutils.PublishMessage(t, env.EventBus, env.Ctx, roundevents.RoundFinalizedV1, msg); err != nil {
					t.Fatalf("Publish failed: %v", err)
				}
				return msg
			},
			expectedOutgoingTopics: []string{sharedevents.ProcessRoundScoresRequestedV1},
			validateFn: func(t *testing.T, deps HandlerTestDeps, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[sharedevents.ProcessRoundScoresRequestedV1]
				if len(msgs) == 0 {
					t.Fatalf("expected process round scores request, got none")
				}
			},
			timeout: 5 * time.Second,
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
