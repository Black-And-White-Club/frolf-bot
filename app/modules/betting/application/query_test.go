package bettingservice

import (
	"context"
	"errors"
	"testing"
	"time"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// TestGetOverview
// ---------------------------------------------------------------------------

func TestGetOverview(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	userUUID := uuid.New()
	updatedAt := time.Date(2026, 3, 12, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository, *FakeLeaderboardRepository)
		verify func(t *testing.T, result *Overview, err error)
	}{
		{
			name: "aggregates season points, adjustment balance, and journal",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(userUUID, clubUUID), nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring", Name: "Spring 2026", GuildID: "guild-1"}, nil
				}
				userRepo.GetUserByUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (*userdb.User, error) {
					discordID := sharedtypes.DiscordID("player-overview")
					return &userdb.User{UserID: &discordID}, nil
				}
				lbRepo.GetSeasonStandingFunc = func(_ context.Context, _ bun.IDB, _ string, _ sharedtypes.DiscordID) (*leaderboarddb.SeasonStanding, error) {
					return &leaderboarddb.SeasonStanding{TotalPoints: 300}, nil
				}
				repo.GetWalletJournalBalanceFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, seasonID string) (int, error) {
					if seasonID != "2026-spring" {
						t.Errorf("unexpected seasonID: %s", seasonID)
					}
					return -120, nil
				}
				repo.GetReservedStakeTotalFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string) (int, error) {
					return 50, nil
				}
				repo.GetMemberSettingsFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*bettingdb.MemberSetting, error) {
					return &bettingdb.MemberSetting{OptOutTargeting: true, UpdatedAt: updatedAt}, nil
				}
				repo.ListWalletJournalFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID, _ string, limit int) ([]bettingdb.WalletJournalEntry, error) {
					if limit != walletHistorySize {
						t.Errorf("unexpected limit: %d", limit)
					}
					return []bettingdb.WalletJournalEntry{
						{ID: 7, EntryType: "manual_adjustment", Amount: -120, Reason: "correction", CreatedAt: updatedAt},
					}, nil
				}
			},
			verify: func(t *testing.T, result *Overview, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.Wallet.SeasonPoints != 300 {
					t.Errorf("SeasonPoints: want 300, got %d", result.Wallet.SeasonPoints)
				}
				if result.Wallet.AdjustmentBalance != -120 {
					t.Errorf("AdjustmentBalance: want -120, got %d", result.Wallet.AdjustmentBalance)
				}
				if result.Wallet.Available != -120-50 {
					t.Errorf("Available: want %d, got %d", -120-50, result.Wallet.Available)
				}
				if result.Wallet.Reserved != 50 {
					t.Errorf("Reserved: want 50, got %d", result.Wallet.Reserved)
				}
				if result.Settings.OptOutTargeting != true {
					t.Error("expected OptOutTargeting=true")
				}
				if len(result.Journal) != 1 {
					t.Errorf("expected 1 journal entry, got %d", len(result.Journal))
				}
			},
		},
		{
			name: "disabled feature returns error",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return disabledEntitlements(), nil
				}
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(userUUID, clubUUID), nil
				}
			},
			verify: func(t *testing.T, result *Overview, err error) {
				if !errors.Is(err, ErrFeatureDisabled) {
					t.Errorf("expected ErrFeatureDisabled, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			guildRepo := NewFakeGuildRepository()
			lbRepo := NewFakeLeaderboardRepository()
			tt.setup(repo, userRepo, guildRepo, lbRepo)
			svc := newTestService(repo, userRepo, guildRepo, lbRepo, nil)
			result, err := svc.GetOverview(context.Background(), clubUUID, userUUID)
			tt.verify(t, result, err)
		})
	}
}

// ---------------------------------------------------------------------------
// TestGetAdminMarkets
// ---------------------------------------------------------------------------

func TestGetAdminMarkets(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	adminUUID := uuid.New()
	memberUUID := uuid.New()

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository)
		verify func(t *testing.T, result *AdminMarketBoard, err error)
	}{
		{
			name: "non-admin user is rejected",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(memberUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
			},
			verify: func(t *testing.T, result *AdminMarketBoard, err error) {
				if !errors.Is(err, ErrAdminRequired) {
					t.Errorf("expected ErrAdminRequired, got %v", err)
				}
			},
		},
		{
			name: "admin receives market board with summaries",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				marketID := int64(42)
				roundID := uuid.New()
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return adminMembership(adminUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				repo.ListMarketsForClubFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, limit int) ([]bettingdb.Market, error) {
					if limit != adminMarketListSize {
						t.Errorf("unexpected limit: %d", limit)
					}
					return []bettingdb.Market{
						{ID: marketID, RoundID: roundID, MarketType: winnerMarketType, Status: settledMarketStatus, Title: "Test Market"},
					}, nil
				}
				repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
					return []bettingdb.Bet{
						{ID: 1, MarketID: marketID, Status: wonBetStatus, Stake: 100, PotentialPayout: 200},
						{ID: 2, MarketID: marketID, Status: lostBetStatus, Stake: 50, PotentialPayout: 75},
					}, nil
				}
			},
			verify: func(t *testing.T, result *AdminMarketBoard, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(result.Markets) != 1 {
					t.Fatalf("expected 1 market, got %d", len(result.Markets))
				}
				m := result.Markets[0]
				if m.TicketCount != 2 {
					t.Errorf("TicketCount: want 2, got %d", m.TicketCount)
				}
				if m.WonTickets != 1 {
					t.Errorf("WonTickets: want 1, got %d", m.WonTickets)
				}
				if m.LostTickets != 1 {
					t.Errorf("LostTickets: want 1, got %d", m.LostTickets)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			guildRepo := NewFakeGuildRepository()
			tt.setup(repo, userRepo, guildRepo)
			svc := newTestService(repo, userRepo, guildRepo, NewFakeLeaderboardRepository(), nil)
			result, err := svc.GetAdminMarkets(context.Background(), clubUUID, adminUUID)
			tt.verify(t, result, err)
		})
	}
}
