package bettingservice

import (
	"context"
	"errors"
	"strings"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"testing"
)

// ---------------------------------------------------------------------------
// TestSettleRound
// ---------------------------------------------------------------------------

func TestSettleRound(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-1")
	roundUUID := uuid.New()
	roundID := sharedtypes.RoundID(roundUUID)
	marketID := int64(55)

	baseRound := &BettingSettlementRound{
		ID:        roundID,
		Title:     "Test Round",
		GuildID:   guildID,
		Finalized: true,
		Participants: []BettingSettlementParticipant{
			{MemberID: "player-a", Response: string(roundtypes.ResponseAccept), Score: ptr(72)},
			{MemberID: "player-b", Response: string(roundtypes.ResponseAccept), Score: ptr(80)},
		},
	}

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository)
		round  *BettingSettlementRound
		verify func(t *testing.T, results []MarketSettlementResult, err error, repo *FakeBettingRepository)
	}{
		{
			name:  "nil round returns nil results",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository) {},
			round: nil,
			verify: func(t *testing.T, results []MarketSettlementResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if results != nil {
					t.Errorf("expected nil results for nil round, got %v", results)
				}
			},
		},
		{
			name: "single winner bet gets payout journal entry",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				repo.ListMarketsByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ uuid.UUID) ([]bettingdb.Market, error) {
					return []bettingdb.Market{
						{ID: marketID, ClubUUID: clubUUID, RoundID: roundUUID, MarketType: winnerMarketType, Status: openMarketStatus, SettlementVersion: 0},
					}, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return []bettingdb.MarketOption{
						{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a", ProbabilityBps: 5000, DecimalOddsCents: 200},
						{MarketID: marketID, OptionKey: "player-b", ParticipantMemberID: "player-b", ProbabilityBps: 5000, DecimalOddsCents: 200},
					}, nil
				}
				repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
					return []bettingdb.Bet{
						{ID: 1, MarketID: marketID, ClubUUID: clubUUID, SelectionKey: "player-a", Stake: 100, DecimalOddsCents: 200, PotentialPayout: 200, Status: acceptedBetStatus},
					}, nil
				}
			},
			round: baseRound,
			verify: func(t *testing.T, results []MarketSettlementResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 1 {
					t.Fatalf("expected 1 settlement result, got %d", len(results))
				}
				if results[0].MarketID != marketID {
					t.Errorf("MarketID: want %d, got %d", marketID, results[0].MarketID)
				}
				trace := repo.Trace()
				foundJournal := false
				foundBetUpdate := false
				for _, s := range trace {
					if s == "CreateWalletJournalEntry" {
						foundJournal = true
					}
					if s == "UpdateBet" {
						foundBetUpdate = true
					}
				}
				if !foundJournal {
					t.Errorf("expected CreateWalletJournalEntry in trace, got %v", trace)
				}
				if !foundBetUpdate {
					t.Errorf("expected UpdateBet in trace, got %v", trace)
				}
			},
		},
		{
			name: "no-scorer round voids all bets",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				repo.ListMarketsByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ uuid.UUID) ([]bettingdb.Market, error) {
					return []bettingdb.Market{
						{ID: marketID, ClubUUID: clubUUID, RoundID: roundUUID, MarketType: winnerMarketType, Status: openMarketStatus},
					}, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					// Options exist but no participant has a score
					return []bettingdb.MarketOption{
						{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a"},
					}, nil
				}
				repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
					return []bettingdb.Bet{
						{ID: 1, MarketID: marketID, ClubUUID: clubUUID, SelectionKey: "player-a", Stake: 50, Status: acceptedBetStatus},
					}, nil
				}
			},
			round: &BettingSettlementRound{
				ID:        roundID,
				Title:     "No Score Round",
				GuildID:   guildID,
				Finalized: true,
				Participants: []BettingSettlementParticipant{
					{MemberID: "player-a", Response: string(roundtypes.ResponseAccept), Score: nil}, // no score
				},
			},
			verify: func(t *testing.T, results []MarketSettlementResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Refund journal entry should be present
				trace := repo.Trace()
				foundJournal := false
				for _, s := range trace {
					if s == "CreateWalletJournalEntry" {
						foundJournal = true
					}
				}
				if !foundJournal {
					t.Errorf("expected refund CreateWalletJournalEntry for voided market, got %v", trace)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			tt.setup(repo, userRepo)
			svc := newTestService(repo, userRepo, NewFakeGuildRepository(), NewFakeLeaderboardRepository(), nil)
			results, err := svc.SettleRound(context.Background(), guildID, tt.round, "test", nil, "")
			tt.verify(t, results, err, repo)
		})
	}
}

// ---------------------------------------------------------------------------
// F8 tests: settlement entitlement behaviour
// ---------------------------------------------------------------------------

func TestSettleRound_AllowedWhenFrozen(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-1")
	roundUUID := uuid.New()
	roundID := sharedtypes.RoundID(roundUUID)
	marketID := int64(55)

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()

	// Guild is frozen (read-only) — settlement MUST continue.
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return frozenEntitlements(), nil
	}
	userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
		return clubUUID, nil
	}
	repo.ListMarketsByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ uuid.UUID) ([]bettingdb.Market, error) {
		return []bettingdb.Market{
			{ID: marketID, ClubUUID: clubUUID, RoundID: roundUUID, MarketType: winnerMarketType, Status: openMarketStatus},
		}, nil
	}
	repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
		return []bettingdb.MarketOption{
			{MarketID: marketID, OptionKey: "player-a", ParticipantMemberID: "player-a", ProbabilityBps: 5000, DecimalOddsCents: 200},
		}, nil
	}
	repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
		return []bettingdb.Bet{
			{ID: 1, MarketID: marketID, ClubUUID: clubUUID, SelectionKey: "player-a", Stake: 100, DecimalOddsCents: 200, PotentialPayout: 200, Status: acceptedBetStatus},
		}, nil
	}

	round := &BettingSettlementRound{
		ID: roundID, GuildID: guildID, Finalized: true,
		Participants: []BettingSettlementParticipant{
			{MemberID: "player-a", Response: string(roundtypes.ResponseAccept), Score: ptr(72)},
		},
	}
	svc := newTestService(repo, userRepo, guildRepo, NewFakeLeaderboardRepository(), nil)
	results, err := svc.SettleRound(context.Background(), guildID, round, "test", nil, "")
	if err != nil {
		t.Fatalf("expected settlement to proceed when frozen, got error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 settlement result, got %d", len(results))
	}
}

func TestSettleRound_BlockedWhenDisabled(t *testing.T) {
	t.Parallel()

	const guildID = sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()

	clubUUID := uuid.New()
	userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
		return clubUUID, nil
	}
	// Guild is fully disabled — settlement must be blocked.
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return disabledEntitlements(), nil
	}

	round := &BettingSettlementRound{
		ID: roundID, GuildID: guildID, Finalized: true,
		Participants: []BettingSettlementParticipant{
			{MemberID: "player-a", Response: string(roundtypes.ResponseAccept), Score: ptr(72)},
		},
	}
	svc := newTestService(repo, userRepo, guildRepo, NewFakeLeaderboardRepository(), nil)
	_, err := svc.SettleRound(context.Background(), guildID, round, "test", nil, "")
	if err == nil {
		t.Fatal("expected ErrFeatureDisabled when feature is disabled, got nil")
	}
	if !errors.Is(err, ErrFeatureDisabled) {
		t.Errorf("expected ErrFeatureDisabled, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Placement outcome derivation tests
// ---------------------------------------------------------------------------

func TestDerivePlacementOutcome(t *testing.T) {
	t.Parallel()

	accept := string(roundtypes.ResponseAccept)

	// helper to build options for a set of member IDs
	makeOpts := func(ids ...string) []bettingdb.MarketOption {
		opts := make([]bettingdb.MarketOption, len(ids))
		for i, id := range ids {
			opts[i] = bettingdb.MarketOption{OptionKey: id, ParticipantMemberID: id, Label: id}
		}
		return opts
	}

	tests := []struct {
		name           string
		participants   []BettingSettlementParticipant
		options        []bettingdb.MarketOption
		targetPosition int
		wantStatus     string
		wantWinners    []string // option keys that must be in winners
		wantVoided     []string // option keys that must be in scratched
	}{
		{
			name: "2nd_place_clear_winner",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, Score: ptr(70)},
				{MemberID: "carol", Response: accept, Score: ptr(80)},
			},
			options:        makeOpts("alice", "bob", "carol"),
			targetPosition: 2,
			wantStatus:     settledMarketStatus,
			wantWinners:    []string{"bob"},
		},
		{
			name: "2nd_place_tie_at_boundary",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, Score: ptr(70)},
				{MemberID: "carol", Response: accept, Score: ptr(70)},
				{MemberID: "dave", Response: accept, Score: ptr(80)},
			},
			options:        makeOpts("alice", "bob", "carol", "dave"),
			targetPosition: 2,
			wantStatus:     settledMarketStatus,
			wantWinners:    []string{"bob", "carol"}, // both tied at 70
		},
		{
			name: "last_place_dynamic",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, Score: ptr(70)},
				{MemberID: "carol", Response: accept, Score: ptr(80)},
			},
			options:        makeOpts("alice", "bob", "carol"),
			targetPosition: -1, // dynamic last
			wantStatus:     settledMarketStatus,
			wantWinners:    []string{"carol"},
		},
		{
			name: "last_place_tie",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, Score: ptr(80)},
				{MemberID: "carol", Response: accept, Score: ptr(80)},
			},
			options:        makeOpts("alice", "bob", "carol"),
			targetPosition: -1,
			wantStatus:     settledMarketStatus,
			wantWinners:    []string{"bob", "carol"},
		},
		{
			name: "dnf_player_voided",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, Score: ptr(70)},
				{MemberID: "carol", Response: accept, IsDNF: true},
			},
			options:        makeOpts("alice", "bob", "carol"),
			targetPosition: 2,
			wantStatus:     settledMarketStatus,
			wantWinners:    []string{"bob"},
			wantVoided:     []string{"carol"},
		},
		{
			name: "too_few_finishers_voids_market",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(60)},
				{MemberID: "bob", Response: accept, IsDNF: true},
			},
			options:        makeOpts("alice", "bob"),
			targetPosition: 2,
			wantStatus:     voidedMarketStatus,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			round := &BettingSettlementRound{
				ID:           sharedtypes.RoundID(uuid.New()),
				GuildID:      "guild-test",
				Finalized:    true,
				Participants: tc.participants,
			}
			outcome := derivePlacementOutcome(round, tc.options, tc.targetPosition)

			if outcome.status != tc.wantStatus {
				t.Errorf("status: want %q, got %q (voidReason=%q)", tc.wantStatus, outcome.status, outcome.voidReason)
			}

			for _, key := range tc.wantWinners {
				if _, ok := outcome.winners[key]; !ok {
					t.Errorf("expected key %q in winners, got %v", key, outcome.winners)
				}
			}

			for _, key := range tc.wantVoided {
				if _, ok := outcome.scratched[key]; !ok {
					t.Errorf("expected key %q in scratched, got %v", key, outcome.scratched)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Over/Under outcome derivation tests
// ---------------------------------------------------------------------------

func TestDeriveOverUnderOutcome(t *testing.T) {
	t.Parallel()

	accept := string(roundtypes.ResponseAccept)

	makeOUOptions := func(memberID string, line int) []bettingdb.MarketOption {
		meta := `{"line":` + itoa(line) + `}`
		return []bettingdb.MarketOption{
			{OptionKey: memberID + "_over", ParticipantMemberID: memberID, Label: memberID + " Over", Metadata: meta},
			{OptionKey: memberID + "_under", ParticipantMemberID: memberID, Label: memberID + " Under", Metadata: meta},
		}
	}

	tests := []struct {
		name         string
		participants []BettingSettlementParticipant
		options      []bettingdb.MarketOption
		wantStatus   string
		wantWinners  []string
		wantVoided   []string
	}{
		{
			name: "over_wins",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(55)},
				{MemberID: "bob", Response: accept, Score: ptr(48)},
			},
			options:     append(makeOUOptions("alice", 50), makeOUOptions("bob", 50)...),
			wantStatus:  settledMarketStatus,
			wantWinners: []string{"alice_over", "bob_under"},
		},
		{
			name: "under_wins",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(45)},
			},
			options:     makeOUOptions("alice", 50),
			wantStatus:  settledMarketStatus,
			wantWinners: []string{"alice_under"},
		},
		{
			name: "push_goes_to_under",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, Score: ptr(50)},
			},
			options:     makeOUOptions("alice", 50),
			wantStatus:  settledMarketStatus,
			wantWinners: []string{"alice_under"}, // push: under wins
		},
		{
			name: "dnf_voids_both_options",
			participants: []BettingSettlementParticipant{
				{MemberID: "alice", Response: accept, IsDNF: true},
			},
			options:    makeOUOptions("alice", 50),
			wantStatus: voidedMarketStatus,
			wantVoided: []string{"alice_over", "alice_under"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			round := &BettingSettlementRound{
				ID:           sharedtypes.RoundID(uuid.New()),
				GuildID:      "guild-ou",
				Finalized:    true,
				Participants: tc.participants,
			}
			outcome := deriveOverUnderOutcome(round, tc.options)

			if outcome.status != tc.wantStatus {
				t.Errorf("status: want %q, got %q (voidReason=%q)", tc.wantStatus, outcome.status, outcome.voidReason)
			}

			for _, key := range tc.wantWinners {
				if _, ok := outcome.winners[key]; !ok {
					t.Errorf("expected key %q in winners, got %v", key, outcome.winners)
				}
			}

			for _, key := range tc.wantVoided {
				if _, ok := outcome.scratched[key]; !ok {
					t.Errorf("expected key %q in scratched, got %v", key, outcome.scratched)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// parseLineFromMetadata tests
// ---------------------------------------------------------------------------

func TestParseLineFromMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		wantLine int
	}{
		{`{"line":52}`, 52},
		{`{"line":0}`, 0},
		{`{"line":100}`, 100},
		{``, 0},
		{`{"other":"field"}`, 0},
		{`notjson`, 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := parseLineFromMetadata(tc.input)
			if got != tc.wantLine {
				t.Errorf("parseLineFromMetadata(%q): want %d, got %d", tc.input, tc.wantLine, got)
			}
		})
	}
}

// itoa is a minimal integer-to-string helper for test metadata construction.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// Ensure strings import is used (needed for strings.Contains in test helpers).
var _ = strings.Contains
