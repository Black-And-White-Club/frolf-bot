package bettingservice

import (
	"context"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---- fake round repo for odds tests ----

type fakeOddsRoundRepo struct {
	rounds []*roundtypes.Round
}

func (f *fakeOddsRoundRepo) GetRound(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID, _ sharedtypes.RoundID) (*roundtypes.Round, error) {
	return nil, nil
}

func (f *fakeOddsRoundRepo) GetUpcomingRounds(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) ([]*roundtypes.Round, error) {
	return nil, nil
}

func (f *fakeOddsRoundRepo) GetFinalizedRoundsAfter(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID, _ time.Time) ([]*roundtypes.Round, error) {
	return f.rounds, nil
}

func (f *fakeOddsRoundRepo) GetAllUpcomingRoundsInWindow(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
	return nil, nil
}

// ---- fake leaderboard repo for odds tests ----

type fakeOddsLeaderboardRepo struct{}

func (f *fakeOddsLeaderboardRepo) GetLeaderboardByGuild(ctx context.Context, guildID sharedtypes.GuildID) (interface{}, error) {
	return nil, nil
}

// buildRound creates a finalized round with the given scores for a fixed set
// of players. scoresByUser maps DiscordID → raw score (lower is better).
func buildFinalizedRound(guildID sharedtypes.GuildID, finishedAt time.Time, scoresByUser map[sharedtypes.DiscordID]int) *roundtypes.Round {
	participants := make([]roundtypes.Participant, 0, len(scoresByUser))
	for uid, s := range scoresByUser {
		score := sharedtypes.Score(s)
		participants = append(participants, roundtypes.Participant{
			UserID: uid,
			Score:  &score,
		})
	}
	return &roundtypes.Round{
		ID:           sharedtypes.RoundID(uuid.New()),
		GuildID:      guildID,
		Finalized:    roundtypes.Finalized(true),
		StartTime:    (*sharedtypes.StartTime)(func() *time.Time { t := finishedAt; return &t }()),
		Participants: participants,
	}
}

// helpers to build targetParticipant slices for test cases
func makeParticipants(ids ...sharedtypes.DiscordID) []targetParticipant {
	out := make([]targetParticipant, len(ids))
	for i, id := range ids {
		out[i] = targetParticipant{
			participant: roundtypes.Participant{UserID: id},
			userUUID:    uuid.New(),
			label:       string(id),
		}
	}
	return out
}

func TestOddsEngine_NoHistory_BaselineFallback(t *testing.T) {
	// With no history every player gets the same baseline rating, so
	// probabilities should be roughly equal — all within [min, max].
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	participants := makeParticipants("playerA", "playerB", "playerC", "playerD")
	opts, err := engine.priceWinnerOptions(context.Background(), nil, "guild1", participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}
	for _, o := range opts {
		// probabilityBps is the raw Monte Carlo rate (not clamped); we verify the
		// priced output via decimalOddsCents, which is computed from the clamped prob.
		if o.decimalOddsCents < minDecimalOddsCents {
			t.Errorf("decimalOddsCents %d below minimum %d for %s", o.decimalOddsCents, minDecimalOddsCents, o.optionKey)
		}
	}
}

func TestOddsEngine_WithHistory_BestPlayerFavoured(t *testing.T) {
	// playerA consistently scores ~50 (good), playerB ~70, playerC ~90 (bad).
	// After enough simulations playerA should have the highest win probability.
	const guild = sharedtypes.GuildID("guild1")
	playerA := sharedtypes.DiscordID("playerA")
	playerB := sharedtypes.DiscordID("playerB")
	playerC := sharedtypes.DiscordID("playerC")

	now := time.Now()
	rounds := make([]*roundtypes.Round, 0, 10)
	for i := 0; i < 10; i++ {
		rounds = append(rounds, buildFinalizedRound(guild, now.Add(-time.Duration(i)*24*time.Hour), map[sharedtypes.DiscordID]int{
			playerA: 50,
			playerB: 70,
			playerC: 90,
		}))
	}

	engine := newOddsEngine(&fakeOddsRoundRepo{rounds: rounds}, nil)
	participants := makeParticipants(playerA, playerB, playerC)
	opts, err := engine.priceWinnerOptions(context.Background(), nil, guild, participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find playerA's probability.
	var probA int
	for _, o := range opts {
		if o.optionKey == string(playerA) {
			probA = o.probabilityBps
		}
	}

	// All decimal odds must be at the minimum.
	for _, o := range opts {
		if o.decimalOddsCents < minDecimalOddsCents {
			t.Errorf("decimalOddsCents %d below minimum %d for %s", o.decimalOddsCents, minDecimalOddsCents, o.optionKey)
		}
	}

	// playerA should have the highest win probability (>40%).
	if probA < 4000 {
		t.Errorf("expected playerA win probability > 40%%, got %d bps (%.2f%%)", probA, float64(probA)/100)
	}
}

func TestOddsEngine_ProbabilitiesWithinBounds(t *testing.T) {
	// Sanity check: even with wildly different players, no probability exceeds
	// the configured ceiling or floor.
	const guild = sharedtypes.GuildID("guild2")
	playerA := sharedtypes.DiscordID("ace")
	playerB := sharedtypes.DiscordID("novice")

	rounds := []*roundtypes.Round{
		buildFinalizedRound(guild, time.Now().Add(-24*time.Hour), map[sharedtypes.DiscordID]int{
			playerA: 40,
			playerB: 120,
		}),
	}

	engine := newOddsEngine(&fakeOddsRoundRepo{rounds: rounds}, nil)
	opts, err := engine.priceWinnerOptions(context.Background(), nil, guild, makeParticipants(playerA, playerB))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// probabilityBps is raw (not clamped). Verify the priced output via
	// decimalOddsCents: valid odds must be >= minimum.
	for _, o := range opts {
		if o.decimalOddsCents < minDecimalOddsCents {
			t.Errorf("decimalOddsCents %d below minimum %d for %s", o.decimalOddsCents, minDecimalOddsCents, o.optionKey)
		}
		// Odds above the ceiling (i.e., pricedProb > maxMarketProbability) cannot
		// happen because the engine clamps pricedProb before computing odds.
		maxOddsCentsForCeiling := int(float64(100) / minMarketProbability * 100)
		if o.decimalOddsCents > maxOddsCentsForCeiling {
			t.Errorf("decimalOddsCents %d exceeds max expected %d for %s", o.decimalOddsCents, maxOddsCentsForCeiling, o.optionKey)
		}
	}
}

func TestOddsEngine_TooFewParticipants(t *testing.T) {
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)
	_, err := engine.priceWinnerOptions(context.Background(), nil, "guild1", makeParticipants("onlyOne"))
	if err == nil {
		t.Fatal("expected error for < 2 participants, got nil")
	}
}

// ---------------------------------------------------------------------------
// Placement pricing tests
// ---------------------------------------------------------------------------

func TestOddsEngine_PricePlacementOptions_BaselineFallback(t *testing.T) {
	// With no history all players get baseline ratings, so placement options
	// should be priced without error and respect the minimum odds floor.
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	participants := makeParticipants("alice", "bob", "carol")
	opts, err := engine.pricePlacementOptions(context.Background(), nil, "guild1", participants, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 3 {
		t.Fatalf("expected 3 options (one per participant), got %d", len(opts))
	}
	for _, o := range opts {
		if o.decimalOddsCents < minDecimalOddsCents {
			t.Errorf("decimalOddsCents %d below minimum %d for option %s", o.decimalOddsCents, minDecimalOddsCents, o.optionKey)
		}
	}
}

func TestOddsEngine_PricePlacementOptions_BestPlayerLeastLikelyLast(t *testing.T) {
	// playerA consistently scores low (good); playerC scores high (bad).
	// For "last place", playerA should have lower probability than playerC.
	const guild = sharedtypes.GuildID("guild-placement")
	playerA := sharedtypes.DiscordID("aces")
	playerB := sharedtypes.DiscordID("mid")
	playerC := sharedtypes.DiscordID("champ-reversed") // worst player

	now := time.Now()
	rounds := make([]*roundtypes.Round, 0, 8)
	for i := 0; i < 8; i++ {
		rounds = append(rounds, buildFinalizedRound(guild, now.Add(-time.Duration(i)*24*time.Hour), map[sharedtypes.DiscordID]int{
			playerA: 45, // best
			playerB: 65,
			playerC: 90, // worst
		}))
	}

	engine := newOddsEngine(&fakeOddsRoundRepo{rounds: rounds}, nil)
	participants := makeParticipants(playerA, playerB, playerC)

	// last place = position 3
	opts, err := engine.pricePlacementOptions(context.Background(), nil, guild, participants, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	probByKey := make(map[string]int, len(opts))
	for _, o := range opts {
		probByKey[o.optionKey] = o.probabilityBps
	}

	if probByKey[string(playerA)] >= probByKey[string(playerC)] {
		t.Errorf("expected playerA (good player) to have lower P(last) than playerC (bad player): playerA=%d bps, playerC=%d bps",
			probByKey[string(playerA)], probByKey[string(playerC)])
	}
}

func TestOddsEngine_PricePlacementOptions_TooFewParticipants(t *testing.T) {
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	_, err := engine.pricePlacementOptions(context.Background(), nil, "guild1", makeParticipants("solo"), 2)
	if err == nil {
		t.Fatal("expected error for single participant, got nil")
	}
}

// ---------------------------------------------------------------------------
// Over/Under pricing tests
// ---------------------------------------------------------------------------

func TestOddsEngine_PriceOverUnderOptions_GeneratesPairs(t *testing.T) {
	// Each participant should get exactly one _over and one _under option,
	// both with ParticipantMemberID matching the player's Discord ID.
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	playerA := sharedtypes.DiscordID("alice")
	playerB := sharedtypes.DiscordID("bob")
	playerC := sharedtypes.DiscordID("carol")
	participants := makeParticipants(playerA, playerB, playerC)

	opts, err := engine.priceOverUnderOptions(context.Background(), nil, "guild1", participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 6 {
		t.Fatalf("expected 6 options (2 per participant), got %d", len(opts))
	}

	overKeys := make(map[string]bool)
	underKeys := make(map[string]bool)
	for _, o := range opts {
		switch {
		case len(o.optionKey) > 5 && o.optionKey[len(o.optionKey)-5:] == "_over":
			pid := o.optionKey[:len(o.optionKey)-5]
			overKeys[pid] = true
			if string(o.memberID) != pid {
				t.Errorf("option %s: memberID %s should match player ID %s", o.optionKey, o.memberID, pid)
			}
		case len(o.optionKey) > 6 && o.optionKey[len(o.optionKey)-6:] == "_under":
			pid := o.optionKey[:len(o.optionKey)-6]
			underKeys[pid] = true
			if string(o.memberID) != pid {
				t.Errorf("option %s: memberID %s should match player ID %s", o.optionKey, o.memberID, pid)
			}
		default:
			t.Errorf("unexpected option key format: %s", o.optionKey)
		}
	}

	for _, p := range participants {
		pid := string(p.participant.UserID)
		if !overKeys[pid] {
			t.Errorf("missing _over option for player %s", pid)
		}
		if !underKeys[pid] {
			t.Errorf("missing _under option for player %s", pid)
		}
	}
}

func TestOddsEngine_PriceOverUnderOptions_LineFromHistory(t *testing.T) {
	// A player with consistent history of score=55 should get a line of 55.
	const guild = sharedtypes.GuildID("guild-ou")
	playerA := sharedtypes.DiscordID("lineplayer")
	playerB := sharedtypes.DiscordID("other")

	now := time.Now()
	rounds := make([]*roundtypes.Round, 0, 5)
	for i := 0; i < 5; i++ {
		rounds = append(rounds, buildFinalizedRound(guild, now.Add(-time.Duration(i)*24*time.Hour), map[sharedtypes.DiscordID]int{
			playerA: 55,
			playerB: 70,
		}))
	}

	engine := newOddsEngine(&fakeOddsRoundRepo{rounds: rounds}, nil)
	opts, err := engine.priceOverUnderOptions(context.Background(), nil, guild, makeParticipants(playerA, playerB))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Find playerA's over option and check metadata contains "line":55.
	found := false
	for _, o := range opts {
		if o.optionKey == string(playerA)+"_over" {
			found = true
			wantMeta := `{"line":55}`
			if o.metadata != wantMeta {
				t.Errorf("playerA over option metadata: want %s, got %s", wantMeta, o.metadata)
			}
		}
	}
	if !found {
		t.Error("playerA over option not found")
	}
}

func TestOddsEngine_PriceOverUnderOptions_ComplementaryProbabilities(t *testing.T) {
	// For each player, over probabilityBps + under probabilityBps should equal
	// 10000 (100% in basis points) within rounding tolerance of ±1.
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	players := []sharedtypes.DiscordID{"p1", "p2", "p3"}
	participants := makeParticipants(players...)

	opts, err := engine.priceOverUnderOptions(context.Background(), nil, "guild1", participants)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	overBps := make(map[string]int)
	underBps := make(map[string]int)
	for _, o := range opts {
		switch {
		case len(o.optionKey) > 5 && o.optionKey[len(o.optionKey)-5:] == "_over":
			overBps[o.optionKey[:len(o.optionKey)-5]] = o.probabilityBps
		case len(o.optionKey) > 6 && o.optionKey[len(o.optionKey)-6:] == "_under":
			underBps[o.optionKey[:len(o.optionKey)-6]] = o.probabilityBps
		}
	}

	for _, pid := range players {
		total := overBps[string(pid)] + underBps[string(pid)]
		if total < 9999 || total > 10001 {
			t.Errorf("player %s: over(%d) + under(%d) = %d, want ~10000", pid, overBps[string(pid)], underBps[string(pid)], total)
		}
	}
}

func TestOddsEngine_PriceOverUnderOptions_TooFewParticipants(t *testing.T) {
	engine := newOddsEngine(&fakeOddsRoundRepo{}, nil)

	_, err := engine.priceOverUnderOptions(context.Background(), nil, "guild1", makeParticipants("solo"))
	if err == nil {
		t.Fatal("expected error for single participant, got nil")
	}
}
