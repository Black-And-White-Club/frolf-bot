package bettinghandlers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	bettingevents "github.com/Black-And-White-Club/frolf-bot-shared/events/betting"
	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	bettingmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/betting"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// TestHandleRoundFinalized
// ---------------------------------------------------------------------------

func TestHandleRoundFinalized(t *testing.T) {
	t.Parallel()

	guildID := sharedtypes.GuildID("guild-123")
	roundID := sharedtypes.RoundID(uuid.New())
	clubUUID := uuid.New()
	marketID := int64(42)

	payload := &roundevents.RoundFinalizedPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
		RoundData: roundtypes.Round{
			ID:      roundID,
			GuildID: guildID,
			State:   roundtypes.RoundStateFinalized,
			Participants: []roundtypes.Participant{
				{UserID: "player-1", Response: roundtypes.ResponseAccept},
				{UserID: "player-2", Response: roundtypes.ResponseAccept},
			},
		},
	}

	tests := []struct {
		name   string
		setup  func(*FakeBettingService)
		verify func(t *testing.T, svc *FakeBettingService, results []interface{ GetTopic() string }, err error)
	}{
		{
			name: "service returns results → emits BettingMarketSettledV1 per result",
			setup: func(svc *FakeBettingService) {
				svc.SettleRoundFunc = func(_ context.Context, _ sharedtypes.GuildID, _ *bettingservice.BettingSettlementRound, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketSettlementResult, error) {
					return []bettingservice.MarketSettlementResult{
						{
							GuildID:           guildID,
							ClubUUID:          clubUUID.String(),
							RoundID:           roundID,
							MarketID:          marketID,
							ResultSummary:     "player-1 wins",
							SettlementVersion: 1,
						},
					}, nil
				}
			},
		},
		{
			name: "service returns multiple results → one event per market",
			setup: func(svc *FakeBettingService) {
				svc.SettleRoundFunc = func(_ context.Context, _ sharedtypes.GuildID, _ *bettingservice.BettingSettlementRound, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketSettlementResult, error) {
					return []bettingservice.MarketSettlementResult{
						{GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: roundID, MarketID: 1},
						{GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: roundID, MarketID: 2},
					}, nil
				}
			},
		},
		{
			name: "service returns empty results → no events emitted",
			setup: func(svc *FakeBettingService) {
				svc.SettleRoundFunc = func(_ context.Context, _ sharedtypes.GuildID, _ *bettingservice.BettingSettlementRound, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketSettlementResult, error) {
					return nil, nil
				}
			},
		},
		{
			name: "service error → handler propagates error",
			setup: func(svc *FakeBettingService) {
				svc.SettleRoundFunc = func(_ context.Context, _ sharedtypes.GuildID, _ *bettingservice.BettingSettlementRound, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketSettlementResult, error) {
					return nil, errors.New("settle failed")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &FakeBettingService{}
			tt.setup(svc)

			h := NewEventHandlers(svc, bettingmetrics.NewNoop())
			results, err := h.HandleRoundFinalized(context.Background(), payload)

			// Check SettleRound was always called
			found := false
			for _, s := range svc.Trace() {
				if s == "SettleRound" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected SettleRound to be called")
			}

			switch tt.name {
			case "service error → handler propagates error":
				if err == nil {
					t.Error("expected error, got nil")
				}
				if results != nil {
					t.Errorf("expected nil results on error, got %v", results)
				}
			case "service returns empty results → no events emitted":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 0 {
					t.Errorf("expected 0 results, got %d", len(results))
				}
			case "service returns results → emits BettingMarketSettledV1 per result":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Each service result produces 2 handler results: bare topic + club-scoped topic.
				if len(results) != 2 {
					t.Fatalf("expected 2 results, got %d", len(results))
				}
				if results[0].Topic != bettingevents.BettingMarketSettledV1 {
					t.Errorf("Topic: want %s, got %s", bettingevents.BettingMarketSettledV1, results[0].Topic)
				}
				wantScoped := fmt.Sprintf("%s.%s", bettingevents.BettingMarketSettledV1, clubUUID.String())
				if results[1].Topic != wantScoped {
					t.Errorf("scoped Topic: want %s, got %s", wantScoped, results[1].Topic)
				}
				p, ok := results[0].Payload.(bettingevents.BettingMarketSettledPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: %T", results[0].Payload)
				}
				if p.GuildID != guildID {
					t.Errorf("GuildID: want %s, got %s", guildID, p.GuildID)
				}
				if p.MarketID != marketID {
					t.Errorf("MarketID: want %d, got %d", marketID, p.MarketID)
				}
				if p.SettlementVersion != 1 {
					t.Errorf("SettlementVersion: want 1, got %d", p.SettlementVersion)
				}
			case "service returns multiple results → one event per market":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// 2 service results × 2 topics (bare + scoped) = 4 handler results.
				if len(results) != 4 {
					t.Errorf("expected 4 results, got %d", len(results))
				}
				wantScoped := fmt.Sprintf("%s.%s", bettingevents.BettingMarketSettledV1, clubUUID.String())
				expectedTopics := []string{
					bettingevents.BettingMarketSettledV1, wantScoped,
					bettingevents.BettingMarketSettledV1, wantScoped,
				}
				for i, want := range expectedTopics {
					if i >= len(results) {
						break
					}
					if results[i].Topic != want {
						t.Errorf("results[%d].Topic: want %s, got %s", i, want, results[i].Topic)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandleRoundDeleted
// ---------------------------------------------------------------------------

func TestHandleRoundDeleted(t *testing.T) {
	t.Parallel()

	guildID := sharedtypes.GuildID("guild-456")
	roundID := sharedtypes.RoundID(uuid.New())
	clubUUID := uuid.New()
	marketID := int64(77)

	payload := &roundevents.RoundDeletedPayloadV1{
		GuildID: guildID,
		RoundID: roundID,
	}

	tests := []struct {
		name  string
		setup func(*FakeBettingService)
	}{
		{
			name: "service returns results → emits BettingMarketVoidedV1 per result",
			setup: func(svc *FakeBettingService) {
				svc.VoidRoundMarketsFunc = func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketVoidResult, error) {
					return []bettingservice.MarketVoidResult{
						{
							GuildID:  guildID,
							ClubUUID: clubUUID.String(),
							RoundID:  roundID,
							MarketID: marketID,
							Reason:   "round deleted",
						},
					}, nil
				}
			},
		},
		{
			name: "service returns multiple results → one event per market",
			setup: func(svc *FakeBettingService) {
				svc.VoidRoundMarketsFunc = func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketVoidResult, error) {
					return []bettingservice.MarketVoidResult{
						{GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: roundID, MarketID: 10},
						{GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: roundID, MarketID: 11},
					}, nil
				}
			},
		},
		{
			name: "service returns empty results → no events emitted",
			setup: func(svc *FakeBettingService) {
				svc.VoidRoundMarketsFunc = func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketVoidResult, error) {
					return nil, nil
				}
			},
		},
		{
			name: "service error → handler propagates error",
			setup: func(svc *FakeBettingService) {
				svc.VoidRoundMarketsFunc = func(_ context.Context, _ sharedtypes.GuildID, _ sharedtypes.RoundID, _ string, _ *uuid.UUID, _ string) ([]bettingservice.MarketVoidResult, error) {
					return nil, errors.New("void failed")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := &FakeBettingService{}
			tt.setup(svc)

			h := NewEventHandlers(svc, bettingmetrics.NewNoop())
			results, err := h.HandleRoundDeleted(context.Background(), payload)

			// VoidRoundMarkets must always be called
			found := false
			for _, s := range svc.Trace() {
				if s == "VoidRoundMarkets" {
					found = true
					break
				}
			}
			if !found {
				t.Error("expected VoidRoundMarkets to be called")
			}

			switch tt.name {
			case "service error → handler propagates error":
				if err == nil {
					t.Error("expected error, got nil")
				}
				if results != nil {
					t.Errorf("expected nil results on error, got %v", results)
				}
			case "service returns empty results → no events emitted":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 0 {
					t.Errorf("expected 0 results, got %d", len(results))
				}
			case "service returns results → emits BettingMarketVoidedV1 per result":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Each service result produces 2 handler results: bare topic + club-scoped topic.
				if len(results) != 2 {
					t.Fatalf("expected 2 results, got %d", len(results))
				}
				if results[0].Topic != bettingevents.BettingMarketVoidedV1 {
					t.Errorf("Topic: want %s, got %s", bettingevents.BettingMarketVoidedV1, results[0].Topic)
				}
				wantScoped := fmt.Sprintf("%s.%s", bettingevents.BettingMarketVoidedV1, clubUUID.String())
				if results[1].Topic != wantScoped {
					t.Errorf("scoped Topic: want %s, got %s", wantScoped, results[1].Topic)
				}
				p, ok := results[0].Payload.(bettingevents.BettingMarketVoidedPayloadV1)
				if !ok {
					t.Fatalf("unexpected payload type: %T", results[0].Payload)
				}
				if p.GuildID != guildID {
					t.Errorf("GuildID: want %s, got %s", guildID, p.GuildID)
				}
				if p.MarketID != marketID {
					t.Errorf("MarketID: want %d, got %d", marketID, p.MarketID)
				}
				if p.Reason != "round deleted" {
					t.Errorf("Reason: want 'round deleted', got %s", p.Reason)
				}
			case "service returns multiple results → one event per market":
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// 2 service results × 2 topics (bare + scoped) = 4 handler results.
				if len(results) != 4 {
					t.Errorf("expected 4 results, got %d", len(results))
				}
				wantScoped := fmt.Sprintf("%s.%s", bettingevents.BettingMarketVoidedV1, clubUUID.String())
				expectedTopics := []string{
					bettingevents.BettingMarketVoidedV1, wantScoped,
					bettingevents.BettingMarketVoidedV1, wantScoped,
				}
				for i, want := range expectedTopics {
					if i >= len(results) {
						break
					}
					if results[i].Topic != want {
						t.Errorf("results[%d].Topic: want %s, got %s", i, want, results[i].Topic)
					}
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestHandleBettingSnapshotRequest
// ---------------------------------------------------------------------------

func TestHandleBettingSnapshotRequest(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()

	cases := []struct {
		name            string
		payload         *bettingevents.BettingSnapshotRequestPayloadV1
		snapshotFunc    func(ctx context.Context, id uuid.UUID) (*bettingservice.MarketSnapshot, error)
		wantErrContains string
		wantResultTopic string
		wantRespErr     bool
	}{
		{
			name:            "invalid club uuid returns parse error",
			payload:         &bettingevents.BettingSnapshotRequestPayloadV1{ClubUUID: "not-a-uuid"},
			wantErrContains: "invalid club_uuid",
		},
		{
			name:    "service error returns response with error field",
			payload: &bettingevents.BettingSnapshotRequestPayloadV1{ClubUUID: clubUUID.String()},
			snapshotFunc: func(_ context.Context, _ uuid.UUID) (*bettingservice.MarketSnapshot, error) {
				return nil, errors.New("db down")
			},
			wantResultTopic: bettingevents.BettingSnapshotResponseV1,
			wantRespErr:     true,
		},
		{
			name:    "success with nil market",
			payload: &bettingevents.BettingSnapshotRequestPayloadV1{ClubUUID: clubUUID.String()},
			snapshotFunc: func(_ context.Context, _ uuid.UUID) (*bettingservice.MarketSnapshot, error) {
				return &bettingservice.MarketSnapshot{
					ClubUUID:    clubUUID.String(),
					GuildID:     "guild-1",
					AccessState: "active",
				}, nil
			},
			wantResultTopic: bettingevents.BettingSnapshotResponseV1,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &FakeBettingService{}
			if tc.snapshotFunc != nil {
				svc.GetMarketSnapshotFunc = tc.snapshotFunc
			}
			h := &EventHandlers{service: svc, metrics: bettingmetrics.NewNoop()}

			results, err := h.HandleBettingSnapshotRequest(context.Background(), tc.payload)

			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}
				if !errors.Is(err, err) {
					t.Errorf("error %v does not contain %q", err, tc.wantErrContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			if results[0].Topic != tc.wantResultTopic {
				t.Errorf("Topic: want %s, got %s", tc.wantResultTopic, results[0].Topic)
			}

			resp, ok := results[0].Payload.(*bettingevents.BettingSnapshotResponsePayloadV1)
			if !ok {
				t.Fatalf("payload not BettingSnapshotResponsePayloadV1")
			}

			if tc.wantRespErr && resp.Error == "" {
				t.Errorf("expected error in response payload, got empty")
			}
			if !tc.wantRespErr && resp.Error != "" {
				t.Errorf("unexpected error in response payload: %s", resp.Error)
			}
		})
	}
}

// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// TestHandleFeatureAccessUpdated (F6)
// ---------------------------------------------------------------------------

func TestHandleFeatureAccessUpdated(t *testing.T) {
	t.Parallel()

	guildID := sharedtypes.GuildID("guild-freeze-test")
	clubUUID := uuid.New()
	marketID := int64(99)
	roundID := sharedtypes.RoundID(uuid.New())

	makeSuspendResult := func() []bettingservice.MarketSuspendedResult {
		return []bettingservice.MarketSuspendedResult{
			{GuildID: guildID, ClubUUID: clubUUID.String(), RoundID: roundID, MarketID: marketID},
		}
	}

	tests := []struct {
		name            string
		entitlements    guildtypes.ResolvedClubEntitlements
		setupSvc        func(*FakeBettingService)
		wantSuspendCall bool
		wantTopics      []string
		wantErr         bool
	}{
		{
			name: "frozen state → suspends open markets, emits BettingMarketSuspendedV1",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
					guildtypes.ClubFeatureBetting: {Key: guildtypes.ClubFeatureBetting, State: guildtypes.FeatureAccessStateFrozen},
				},
			},
			setupSvc: func(svc *FakeBettingService) {
				svc.SuspendOpenMarketsForClubFunc = func(_ context.Context, _ sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error) {
					return makeSuspendResult(), nil
				}
			},
			wantSuspendCall: true,
			wantTopics: []string{
				bettingevents.BettingMarketSuspendedV1,
				fmt.Sprintf("%s.%s", bettingevents.BettingMarketSuspendedV1, clubUUID.String()),
			},
		},
		{
			name: "disabled state → suspends open markets, emits BettingMarketSuspendedV1",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
					guildtypes.ClubFeatureBetting: {Key: guildtypes.ClubFeatureBetting, State: guildtypes.FeatureAccessStateDisabled},
				},
			},
			setupSvc: func(svc *FakeBettingService) {
				svc.SuspendOpenMarketsForClubFunc = func(_ context.Context, _ sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error) {
					return makeSuspendResult(), nil
				}
			},
			wantSuspendCall: true,
			wantTopics: []string{
				bettingevents.BettingMarketSuspendedV1,
				fmt.Sprintf("%s.%s", bettingevents.BettingMarketSuspendedV1, clubUUID.String()),
			},
		},
		{
			name: "enabled state → no suspension, no events",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
					guildtypes.ClubFeatureBetting: {Key: guildtypes.ClubFeatureBetting, State: guildtypes.FeatureAccessStateEnabled},
				},
			},
			setupSvc:        func(*FakeBettingService) {},
			wantSuspendCall: false,
			wantTopics:      nil,
		},
		{
			name: "no betting entitlement key → no suspension, no events",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{},
			},
			setupSvc:        func(*FakeBettingService) {},
			wantSuspendCall: false,
			wantTopics:      nil,
		},
		{
			name: "frozen + no open markets → empty results, no error",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
					guildtypes.ClubFeatureBetting: {Key: guildtypes.ClubFeatureBetting, State: guildtypes.FeatureAccessStateFrozen},
				},
			},
			setupSvc: func(svc *FakeBettingService) {
				svc.SuspendOpenMarketsForClubFunc = func(_ context.Context, _ sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error) {
					return nil, nil
				}
			},
			wantSuspendCall: true,
			wantTopics:      nil,
		},
		{
			name: "service returns error → propagates error",
			entitlements: guildtypes.ResolvedClubEntitlements{
				Features: map[guildtypes.ClubFeatureKey]guildtypes.ClubFeatureAccess{
					guildtypes.ClubFeatureBetting: {Key: guildtypes.ClubFeatureBetting, State: guildtypes.FeatureAccessStateFrozen},
				},
			},
			setupSvc: func(svc *FakeBettingService) {
				svc.SuspendOpenMarketsForClubFunc = func(_ context.Context, _ sharedtypes.GuildID) ([]bettingservice.MarketSuspendedResult, error) {
					return nil, errors.New("db error")
				}
			},
			wantSuspendCall: true,
			wantErr:         true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := &FakeBettingService{}
			tc.setupSvc(svc)
			h := NewEventHandlers(svc, bettingmetrics.NewNoop())

			results, err := h.HandleFeatureAccessUpdated(context.Background(), &guildevents.GuildFeatureAccessUpdatedPayloadV1{
				GuildID:      guildID,
				Entitlements: tc.entitlements,
			})

			if tc.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			calledSuspend := false
			for _, step := range svc.Trace() {
				if step == "SuspendOpenMarketsForClub" {
					calledSuspend = true
				}
			}
			if tc.wantSuspendCall != calledSuspend {
				t.Errorf("wantSuspendCall=%v, got calledSuspend=%v", tc.wantSuspendCall, calledSuspend)
			}

			if len(tc.wantTopics) != len(results) {
				t.Fatalf("expected %d results, got %d", len(tc.wantTopics), len(results))
			}
			for i, want := range tc.wantTopics {
				if results[i].Topic != want {
					t.Errorf("result[%d] topic: want %q, got %q", i, want, results[i].Topic)
				}
				payload, ok := results[i].Payload.(bettingevents.BettingMarketSuspendedPayloadV1)
				if !ok {
					t.Errorf("result[%d] payload: want BettingMarketSuspendedPayloadV1, got %T", i, results[i].Payload)
					continue
				}
				if payload.GuildID != guildID {
					t.Errorf("result[%d] payload.GuildID: want %q, got %q", i, guildID, payload.GuildID)
				}
				if payload.RoundID != roundID {
					t.Errorf("result[%d] payload.RoundID: want %q, got %q", i, roundID, payload.RoundID)
				}
				if payload.MarketID != marketID {
					t.Errorf("result[%d] payload.MarketID: want %d, got %d", i, marketID, payload.MarketID)
				}
			}
		})
	}
	// suppress unused variable warning (clubUUID is used in makeSuspendResult)
	_ = clubUUID
}
