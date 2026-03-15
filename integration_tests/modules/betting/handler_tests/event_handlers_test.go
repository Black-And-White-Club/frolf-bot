package bettinghandlerintegrationtests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// seedHandlerRound inserts an upcoming round into the DB for handler tests.
func seedHandlerRound(t *testing.T, db *bun.DB, roundRepo rounddb.Repository, guildID sharedtypes.GuildID, participants ...roundtypes.Participant) sharedtypes.RoundID {
	t.Helper()
	roundID := sharedtypes.RoundID(uuid.New())
	startTime := sharedtypes.StartTime(time.Now().Add(48 * time.Hour).UTC())
	round := &roundtypes.Round{
		ID:           roundID,
		GuildID:      guildID,
		Title:        "Handler Test Round",
		StartTime:    &startTime,
		State:        roundtypes.RoundStateUpcoming,
		Participants: participants,
	}
	if err := roundRepo.CreateRound(context.Background(), db, guildID, round); err != nil {
		t.Fatalf("seedHandlerRound: %v", err)
	}
	return roundID
}

func TestHandleRoundFinalized(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundFinalizedPayloadV1
		expectedTopics []string
		validate       func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
	}{
		{
			name: "no_markets_for_round_no_output_event",
			setup: func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundFinalizedPayloadV1 {
				world := SeedBettingHandlerWorld(t, deps.DB)
				memberScore := sharedtypes.Score(36)
				adminScore := sharedtypes.Score(40)
				return roundevents.RoundFinalizedPayloadV1{
					GuildID: world.GuildID,
					RoundID: sharedtypes.RoundID(uuid.New()),
					RoundData: roundtypes.Round{
						GuildID: world.GuildID,
						State:   roundtypes.RoundStateFinalized,
						Participants: []roundtypes.Participant{
							{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept, Score: &memberScore},
							{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept, Score: &adminScore},
						},
					},
				}
			},
			expectedTopics: []string{},
			validate: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				if len(receivedMsgs[bettingevents.BettingMarketSettledV1]) > 0 {
					t.Errorf("expected no settled events for round with no markets, got %d", len(receivedMsgs[bettingevents.BettingMarketSettledV1]))
				}
			},
		},
		{
			name: "settles_market_emits_betting_market_settled_event",
			setup: func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundFinalizedPayloadV1 {
				world := SeedBettingHandlerWorld(t, deps.DB)
				roundRepo := rounddb.NewRepository(deps.DB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				roundID := seedHandlerRound(t, deps.DB, roundRepo, world.GuildID, participants...)
				if _, err := deps.BettingModule.BettingService.EnsureMarketsForGuild(context.Background(), world.GuildID); err != nil {
					t.Fatalf("EnsureMarketsForGuild: %v", err)
				}
				memberScore := sharedtypes.Score(36)
				adminScore := sharedtypes.Score(40)
				return roundevents.RoundFinalizedPayloadV1{
					GuildID: world.GuildID,
					RoundID: roundID,
					RoundData: roundtypes.Round{
						ID:      roundID,
						GuildID: world.GuildID,
						State:   roundtypes.RoundStateFinalized,
						Participants: []roundtypes.Participant{
							{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept, Score: &memberScore},
							{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept, Score: &adminScore},
						},
					},
				}
			},
			expectedTopics: []string{bettingevents.BettingMarketSettledV1},
			validate: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[bettingevents.BettingMarketSettledV1]
				if len(msgs) == 0 {
					t.Fatal("expected at least one BettingMarketSettledV1 event")
				}
				var payload bettingevents.BettingMarketSettledPayloadV1
				if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
					t.Fatalf("unmarshal BettingMarketSettledV1 payload: %v", err)
				}
				if payload.MarketID == 0 {
					t.Error("expected non-zero MarketID in settled event")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingHandler(t)
			payload := tt.setup(t, deps)

			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			msg := message.NewMessage(uuid.New().String(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
			msg.Metadata.Set("topic", roundevents.RoundFinalizedV2)

			testutils.RunTest(t, testutils.TestCase{
				Name: tt.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return nil
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundFinalizedV2, msg); err != nil {
						t.Fatalf("publish RoundFinalizedV2: %v", err)
					}
					return msg
				},
				ExpectedTopics: tt.expectedTopics,
				ValidateFn:     tt.validate,
			}, deps.TestEnvironment)
		})
	}
}

func TestHandleRoundDeleted(t *testing.T) {
	tests := []struct {
		name           string
		setup          func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundDeletedPayloadV1
		expectedTopics []string
		validate       func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{})
	}{
		{
			name: "no_markets_for_round_no_output_event",
			setup: func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundDeletedPayloadV1 {
				world := SeedBettingHandlerWorld(t, deps.DB)
				return roundevents.RoundDeletedPayloadV1{
					GuildID: world.GuildID,
					RoundID: sharedtypes.RoundID(uuid.New()),
				}
			},
			expectedTopics: []string{},
			validate: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				if len(receivedMsgs[bettingevents.BettingMarketVoidedV1]) > 0 {
					t.Errorf("expected no void events for round with no markets, got %d", len(receivedMsgs[bettingevents.BettingMarketVoidedV1]))
				}
			},
		},
		{
			name: "voids_market_emits_betting_market_voided_event",
			setup: func(t *testing.T, deps BettingHandlerTestDeps) roundevents.RoundDeletedPayloadV1 {
				world := SeedBettingHandlerWorld(t, deps.DB)
				roundRepo := rounddb.NewRepository(deps.DB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				roundID := seedHandlerRound(t, deps.DB, roundRepo, world.GuildID, participants...)
				if _, err := deps.BettingModule.BettingService.EnsureMarketsForGuild(context.Background(), world.GuildID); err != nil {
					t.Fatalf("EnsureMarketsForGuild: %v", err)
				}
				return roundevents.RoundDeletedPayloadV1{
					GuildID: world.GuildID,
					RoundID: roundID,
				}
			},
			expectedTopics: []string{bettingevents.BettingMarketVoidedV1},
			validate: func(t *testing.T, env *testutils.TestEnvironment, triggerMsg *message.Message, receivedMsgs map[string][]*message.Message, initialState interface{}) {
				msgs := receivedMsgs[bettingevents.BettingMarketVoidedV1]
				if len(msgs) == 0 {
					t.Fatal("expected at least one BettingMarketVoidedV1 event")
				}
				var payload bettingevents.BettingMarketVoidedPayloadV1
				if err := json.Unmarshal(msgs[0].Payload, &payload); err != nil {
					t.Fatalf("unmarshal BettingMarketVoidedV1 payload: %v", err)
				}
				if payload.MarketID == 0 {
					t.Error("expected non-zero MarketID in voided event")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingHandler(t)
			payload := tt.setup(t, deps)

			payloadBytes, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("marshal payload: %v", err)
			}
			msg := message.NewMessage(uuid.New().String(), payloadBytes)
			msg.Metadata.Set(middleware.CorrelationIDMetadataKey, uuid.New().String())
			msg.Metadata.Set("topic", roundevents.RoundDeletedV2)

			testutils.RunTest(t, testutils.TestCase{
				Name: tt.name,
				SetupFn: func(t *testing.T, env *testutils.TestEnvironment) interface{} {
					return nil
				},
				PublishMsgFn: func(t *testing.T, env *testutils.TestEnvironment) *message.Message {
					if err := testutils.PublishMessage(t, deps.EventBus, context.Background(), roundevents.RoundDeletedV2, msg); err != nil {
						t.Fatalf("publish RoundDeletedV2: %v", err)
					}
					return msg
				},
				ExpectedTopics: tt.expectedTopics,
				ValidateFn:     tt.validate,
			}, deps.TestEnvironment)
		})
	}
}
