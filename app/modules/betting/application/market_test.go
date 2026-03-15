package bettingservice

import (
	"context"
	"errors"
	"testing"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// Additional market tests (F2, F3, F10, F11 code-review fixes)
// ---------------------------------------------------------------------------

// TestEnsureMarketsForGuild_AllRoundsFail verifies that EnsureMarketsForGuild
// returns an error when every eligible round fails market generation (F10).
func TestEnsureMarketsForGuild_AllRoundsFail(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-fail")
	roundID := sharedtypes.RoundID(uuid.New())
	futureStart := time.Now().Add(24 * time.Hour)

	round := &roundtypes.Round{
		ID:      roundID,
		GuildID: guildID,
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "a", Response: roundtypes.ResponseAccept},
			{UserID: "b", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()
	lbRepo := NewFakeLeaderboardRepository()
	roundRepo := NewFakeRoundRepository()

	userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
		return clubUUID, nil
	}
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return enabledEntitlements(), nil
	}
	lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
		return &leaderboarddb.Season{ID: "2026"}, nil
	}
	roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
		return []*roundtypes.Round{round}, nil
	}
	repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
		return nil, nil
	}
	// Force CreateMarket to always fail so all rounds fail.
	repo.CreateMarketFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.Market) error {
		return errors.New("DB unavailable")
	}

	svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
	_, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
	if err == nil {
		t.Fatal("expected error when all rounds fail, got nil")
	}
}

// TestEnsureMarketsForGuild_ContextCancelled verifies early exit when the
// context is cancelled mid-loop (F11).
func TestEnsureMarketsForGuild_ContextCancelled(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-cancel")
	futureStart := time.Now().Add(24 * time.Hour)

	rounds := make([]*roundtypes.Round, 3)
	for i := range rounds {
		rounds[i] = &roundtypes.Round{
			ID:      sharedtypes.RoundID(uuid.New()),
			GuildID: guildID,
			State:   roundtypes.RoundStateUpcoming,
			Participants: []roundtypes.Participant{
				{UserID: "a", Response: roundtypes.ResponseAccept},
				{UserID: "b", Response: roundtypes.ResponseAccept},
			},
			StartTime: (*sharedtypes.StartTime)(&futureStart),
		}
	}

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()
	lbRepo := NewFakeLeaderboardRepository()
	roundRepo := NewFakeRoundRepository()

	ctx, cancel := context.WithCancel(context.Background())

	userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
		return clubUUID, nil
	}
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return enabledEntitlements(), nil
	}
	lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
		return &leaderboarddb.Season{ID: "2026"}, nil
	}
	roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
		return rounds, nil
	}
	callCount := 0
	repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
		callCount++
		if callCount == 1 {
			cancel() // cancel after first round's processing begins
		}
		return nil, nil
	}

	svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
	_, err := svc.EnsureMarketsForGuild(ctx, guildID)
	if err == nil {
		t.Fatal("expected context.Canceled error, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestLockDueMarkets_Atomic verifies that a market update failure rolls back
// the entire transaction so no markets are partially locked (F3).
func TestLockDueMarkets_Atomic(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	m1 := bettingdb.Market{ID: 1, ClubUUID: clubUUID, Status: openMarketStatus}
	m2 := bettingdb.Market{ID: 2, ClubUUID: clubUUID, Status: openMarketStatus}

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()

	repo.ListOpenMarketsToLockFunc = func(_ context.Context, _ bun.IDB, _ time.Time) ([]bettingdb.Market, error) {
		return []bettingdb.Market{m1, m2}, nil
	}

	lockedIDs := make([]int64, 0)
	updateCalls := 0
	repo.UpdateMarketFunc = func(_ context.Context, _ bun.IDB, m *bettingdb.Market) error {
		updateCalls++
		if updateCalls == 2 {
			return errors.New("simulated DB error on second update")
		}
		lockedIDs = append(lockedIDs, m.ID)
		return nil
	}

	svc := newTestService(repo, userRepo, NewFakeGuildRepository(), NewFakeLeaderboardRepository(), nil)
	_, err := svc.LockDueMarkets(context.Background())
	if err == nil {
		t.Fatal("expected error from failed market update, got nil")
	}
	// Because the whole operation is wrapped in a transaction, the first
	// market's lock should have been rolled back. We can only verify the error
	// is returned; actual DB rollback is enforced at the DB level in integration
	// tests. Here we confirm LockDueMarkets does NOT swallow the error.
}

// TestEnsureMarketsForGuild_ConflictRetry verifies that a CreateMarket failure
// falls back to a re-select and returns the winner row (F2 TOCTOU fix).
func TestEnsureMarketsForGuild_ConflictRetry(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-conflict")
	roundID := sharedtypes.RoundID(uuid.New())
	futureStart := time.Now().Add(24 * time.Hour)
	existingMarketID := int64(42)

	round := &roundtypes.Round{
		ID:      roundID,
		GuildID: guildID,
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "a", Response: roundtypes.ResponseAccept},
			{UserID: "b", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()
	lbRepo := NewFakeLeaderboardRepository()
	roundRepo := NewFakeRoundRepository()

	userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
		return clubUUID, nil
	}
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return enabledEntitlements(), nil
	}
	lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
		return &leaderboarddb.Season{ID: "2026"}, nil
	}
	roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
		return []*roundtypes.Round{round}, nil
	}

	// First call: no market (our worker checks first)
	// After conflict: second call returns the winner's row.
	getCallCount := 0
	repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
		getCallCount++
		if getCallCount >= 2 {
			return &bettingdb.Market{ID: existingMarketID, MarketType: winnerMarketType, Status: openMarketStatus}, nil
		}
		return nil, nil
	}
	// Simulate a unique-constraint conflict on insert.
	repo.CreateMarketFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.Market) error {
		return errors.New("ERROR: duplicate key value violates unique constraint")
	}
	repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
		return []bettingdb.MarketOption{
			{MarketID: existingMarketID, OptionKey: "a"},
			{MarketID: existingMarketID, OptionKey: "b"},
		}, nil
	}
	userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
		return uuid.New(), nil
	}
	userRepo.GetUserByUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (*userdb.User, error) {
		id := sharedtypes.DiscordID("test-user")
		return &userdb.User{UserID: &id}, nil
	}

	svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
	results, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
	if err != nil {
		t.Fatalf("expected no error on conflict retry, got: %v", err)
	}
	if len(results) < 1 {
		t.Fatalf("expected at least 1 result after conflict retry, got %d", len(results))
	}
	// Find the winner market result specifically.
	var winnerResult *MarketGeneratedResult
	for i := range results {
		if results[i].MarketType == winnerMarketType {
			winnerResult = &results[i]
			break
		}
	}
	if winnerResult == nil {
		t.Fatalf("expected a winner market in results, got types: %v", func() []string {
			types := make([]string, len(results))
			for i, r := range results {
				types[i] = r.MarketType
			}
			return types
		}())
	}
	if winnerResult.MarketID != existingMarketID {
		t.Errorf("expected market ID %d from conflict winner, got %d", existingMarketID, winnerResult.MarketID)
	}
}

func TestEnsureMarketsForGuild(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())
	futureStart := time.Now().Add(24 * time.Hour)

	upcomingRound := &roundtypes.Round{
		ID:      roundID,
		GuildID: guildID,
		State:   roundtypes.RoundStateUpcoming,
		Participants: []roundtypes.Participant{
			{UserID: "player-a", Response: roundtypes.ResponseAccept},
			{UserID: "player-b", Response: roundtypes.ResponseAccept},
		},
		StartTime: (*sharedtypes.StartTime)(&futureStart),
	}

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository, *FakeLeaderboardRepository, *FakeRoundRepository)
		verify func(t *testing.T, results []MarketGeneratedResult, err error, repo *FakeBettingRepository)
	}{
		{
			name: "disabled guild returns empty results without creating markets",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return disabledEntitlements(), nil
				}
			},
			verify: func(t *testing.T, results []MarketGeneratedResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 0 {
					t.Errorf("expected no results for disabled guild, got %d", len(results))
				}
				for _, s := range repo.Trace() {
					if s == "CreateMarket" {
						t.Error("expected no CreateMarket call for disabled guild")
					}
				}
			},
		},
		{
			name: "frozen guild returns empty results without creating markets",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return frozenEntitlements(), nil
				}
			},
			verify: func(t *testing.T, results []MarketGeneratedResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 0 {
					t.Errorf("expected no results for frozen guild, got %d", len(results))
				}
			},
		},
		{
			name: "enabled guild creates winner market for upcoming round",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
					return []*roundtypes.Round{upcomingRound}, nil
				}
				// No existing market
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return nil, nil
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
					return uuid.New(), nil
				}
				userRepo.GetUserByUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (*userdb.User, error) {
					discordID := sharedtypes.DiscordID("player-test")
					return &userdb.User{UserID: &discordID}, nil
				}
			},
			verify: func(t *testing.T, results []MarketGeneratedResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// 2-player round generates winner + over_under.
				if len(results) < 1 {
					t.Fatalf("expected at least 1 result, got %d", len(results))
				}
				var winnerResult *MarketGeneratedResult
				for i := range results {
					if results[i].MarketType == winnerMarketType {
						winnerResult = &results[i]
						break
					}
				}
				if winnerResult == nil {
					t.Fatalf("expected a winner market in results")
				}
				if winnerResult.GuildID != guildID {
					t.Errorf("GuildID: want %s, got %s", guildID, winnerResult.GuildID)
				}
				trace := repo.Trace()
				foundCreate := false
				for _, s := range trace {
					if s == "CreateMarket" {
						foundCreate = true
					}
				}
				if !foundCreate {
					t.Errorf("expected CreateMarket in trace, got %v", trace)
				}
			},
		},
		{
			name: "existing market is not duplicated",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository, roundRepo *FakeRoundRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
					return []*roundtypes.Round{upcomingRound}, nil
				}
				// Existing market present
				repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
					return &bettingdb.Market{ID: 77, MarketType: winnerMarketType, Status: openMarketStatus}, nil
				}
				repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
					return []bettingdb.MarketOption{
						{MarketID: 77, OptionKey: "player-a"},
						{MarketID: 77, OptionKey: "player-b"},
					}, nil
				}
			},
			verify: func(t *testing.T, results []MarketGeneratedResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				for _, s := range repo.Trace() {
					if s == "CreateMarket" {
						t.Error("expected no CreateMarket call when market already exists")
					}
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
			guildRepo := NewFakeGuildRepository()
			lbRepo := NewFakeLeaderboardRepository()
			roundRepo := NewFakeRoundRepository()
			tt.setup(repo, userRepo, guildRepo, lbRepo, roundRepo)
			svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
			results, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
			tt.verify(t, results, err, repo)
		})
	}
}

// ---------------------------------------------------------------------------
// TestEnsureMarketsForGuild_FieldSizeGating
// Verifies that correct market type subsets are generated based on field size.
// ---------------------------------------------------------------------------

func TestEnsureMarketsForGuild_FieldSizeGating(t *testing.T) {
	t.Parallel()

	const guildID = sharedtypes.GuildID("guild-sizing")
	futureStart := time.Now().Add(24 * time.Hour)

	// wireBaseSetup handles the common repo wiring, leaving round and clubUUID as params.
	wireBaseSetup := func(
		repo *FakeBettingRepository,
		userRepo *FakeUserRepository,
		guildRepo *FakeGuildRepository,
		lbRepo *FakeLeaderboardRepository,
		roundRepo *FakeRoundRepository,
		clubUUID uuid.UUID,
		round *roundtypes.Round,
	) {
		userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
			return clubUUID, nil
		}
		guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
			return enabledEntitlements(), nil
		}
		lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
			return &leaderboarddb.Season{ID: "2026-spring"}, nil
		}
		roundRepo.GetAllUpcomingRoundsInWindowFunc = func(_ context.Context, _ bun.IDB, _ time.Duration) ([]*roundtypes.Round, error) {
			return []*roundtypes.Round{round}, nil
		}
		// No existing markets — force fresh creation for each type.
		repo.GetMarketByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ string, _ uuid.UUID, _ string) (*bettingdb.Market, error) {
			return nil, nil
		}
		// Return empty options on list (market was just created).
		repo.ListMarketOptionsFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.MarketOption, error) {
			return nil, nil
		}
		// No history needed — baseline odds are fine.
		roundRepo.GetFinalizedRoundsAfterFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID, _ time.Time) ([]*roundtypes.Round, error) {
			return nil, nil
		}
		userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
			return uuid.New(), nil
		}
	}

	makeRound := func(clubUUID uuid.UUID, ids ...sharedtypes.DiscordID) *roundtypes.Round {
		roundID := sharedtypes.RoundID(uuid.New())
		participants := make([]roundtypes.Participant, len(ids))
		for i, id := range ids {
			participants[i] = roundtypes.Participant{UserID: id, Response: roundtypes.ResponseAccept}
		}
		return &roundtypes.Round{
			ID:           roundID,
			GuildID:      guildID,
			State:        roundtypes.RoundStateUpcoming,
			Participants: participants,
			StartTime:    (*sharedtypes.StartTime)(&futureStart),
		}
	}

	collectMarketTypes := func(results []MarketGeneratedResult) map[string]int {
		counts := make(map[string]int)
		for _, r := range results {
			counts[r.MarketType]++
		}
		return counts
	}

	t.Run("2_players_creates_winner_and_over_under_only", func(t *testing.T) {
		t.Parallel()
		clubUUID := uuid.New()
		round := makeRound(clubUUID, "p1", "p2")

		repo := NewFakeBettingRepository()
		userRepo := NewFakeUserRepository()
		guildRepo := NewFakeGuildRepository()
		lbRepo := NewFakeLeaderboardRepository()
		roundRepo := NewFakeRoundRepository()
		wireBaseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo, clubUUID, round)

		svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
		results, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := collectMarketTypes(results)
		if counts[winnerMarketType] != 1 {
			t.Errorf("expected 1 winner market, got %d", counts[winnerMarketType])
		}
		if counts[overUnderMarketType] != 1 {
			t.Errorf("expected 1 over_under market, got %d", counts[overUnderMarketType])
		}
		if counts[placement2ndMarketType] != 0 {
			t.Errorf("expected 0 placement_2nd for 2-player field, got %d", counts[placement2ndMarketType])
		}
		if counts[placement3rdMarketType] != 0 {
			t.Errorf("expected 0 placement_3rd for 2-player field, got %d", counts[placement3rdMarketType])
		}
		if counts[placementLastMarketType] != 0 {
			t.Errorf("expected 0 placement_last for 2-player field, got %d", counts[placementLastMarketType])
		}
	})

	t.Run("3_players_creates_winner_2nd_last_and_over_under", func(t *testing.T) {
		t.Parallel()
		clubUUID := uuid.New()
		round := makeRound(clubUUID, "p1", "p2", "p3")

		repo := NewFakeBettingRepository()
		userRepo := NewFakeUserRepository()
		guildRepo := NewFakeGuildRepository()
		lbRepo := NewFakeLeaderboardRepository()
		roundRepo := NewFakeRoundRepository()
		wireBaseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo, clubUUID, round)

		svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
		results, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := collectMarketTypes(results)
		if counts[winnerMarketType] != 1 {
			t.Errorf("expected 1 winner market, got %d", counts[winnerMarketType])
		}
		if counts[placement2ndMarketType] != 1 {
			t.Errorf("expected 1 placement_2nd for 3-player field, got %d", counts[placement2ndMarketType])
		}
		if counts[placementLastMarketType] != 1 {
			t.Errorf("expected 1 placement_last for 3-player field, got %d", counts[placementLastMarketType])
		}
		if counts[overUnderMarketType] != 1 {
			t.Errorf("expected 1 over_under market, got %d", counts[overUnderMarketType])
		}
		if counts[placement3rdMarketType] != 0 {
			t.Errorf("expected 0 placement_3rd for 3-player field (need 4+), got %d", counts[placement3rdMarketType])
		}
	})

	t.Run("4_players_creates_all_5_market_types", func(t *testing.T) {
		t.Parallel()
		clubUUID := uuid.New()
		round := makeRound(clubUUID, "p1", "p2", "p3", "p4")

		repo := NewFakeBettingRepository()
		userRepo := NewFakeUserRepository()
		guildRepo := NewFakeGuildRepository()
		lbRepo := NewFakeLeaderboardRepository()
		roundRepo := NewFakeRoundRepository()
		wireBaseSetup(repo, userRepo, guildRepo, lbRepo, roundRepo, clubUUID, round)

		svc := newTestService(repo, userRepo, guildRepo, lbRepo, roundRepo)
		results, err := svc.EnsureMarketsForGuild(context.Background(), guildID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		counts := collectMarketTypes(results)
		for _, mt := range []string{winnerMarketType, placement2ndMarketType, placement3rdMarketType, placementLastMarketType, overUnderMarketType} {
			if counts[mt] != 1 {
				t.Errorf("expected 1 %s market for 4-player field, got %d", mt, counts[mt])
			}
		}
		if len(results) != 5 {
			t.Errorf("expected 5 total market results for 4-player field, got %d", len(results))
		}
	})
}
