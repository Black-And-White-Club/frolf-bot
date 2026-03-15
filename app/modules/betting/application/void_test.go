package bettingservice

import (
	"context"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"testing"
)

// ---------------------------------------------------------------------------
// TestVoidRoundMarkets
// ---------------------------------------------------------------------------

func TestVoidRoundMarkets(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	const guildID = sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())
	marketID := int64(66)

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository)
		verify func(t *testing.T, results []MarketVoidResult, err error, repo *FakeBettingRepository)
	}{
		{
			name: "all accepted bets are refunded and market is voided",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				repo.ListMarketsByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ uuid.UUID) ([]bettingdb.Market, error) {
					return []bettingdb.Market{
						{ID: marketID, ClubUUID: clubUUID, RoundID: roundID.UUID(), MarketType: winnerMarketType, Status: openMarketStatus},
					}, nil
				}
				repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
					return []bettingdb.Bet{
						{ID: 10, MarketID: marketID, ClubUUID: clubUUID, Stake: 100, Status: acceptedBetStatus},
						{ID: 11, MarketID: marketID, ClubUUID: clubUUID, Stake: 50, Status: acceptedBetStatus},
					}, nil
				}
			},
			verify: func(t *testing.T, results []MarketVoidResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 1 {
					t.Fatalf("expected 1 void result, got %d", len(results))
				}
				if results[0].MarketID != marketID {
					t.Errorf("MarketID: want %d, got %d", marketID, results[0].MarketID)
				}
				trace := repo.Trace()
				// Accepted bets do not create journal entries on void — their stake is tracked
				// solely via reserved (GetWalletJournalBalance excludes stake_reserved entries),
				// so no bettingBalance credit is needed. Only previously-settled bets trigger
				// a correction journal entry.
				journalCount := 0
				for _, s := range trace {
					if s == "CreateWalletJournalEntry" {
						journalCount++
					}
				}
				if journalCount != 0 {
					t.Errorf("expected 0 CreateWalletJournalEntry calls for accepted bets, got %d — trace: %v", journalCount, trace)
				}
			},
		},
		{
			name: "no markets returns empty results",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				repo.ListMarketsByRoundFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ uuid.UUID) ([]bettingdb.Market, error) {
					return nil, nil
				}
			},
			verify: func(t *testing.T, results []MarketVoidResult, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(results) != 0 {
					t.Errorf("expected no results, got %d", len(results))
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
			results, err := svc.VoidRoundMarkets(context.Background(), guildID, roundID, "test", nil, "test void")
			tt.verify(t, results, err, repo)
		})
	}
}
