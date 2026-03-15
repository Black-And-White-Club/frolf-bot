package bettingservice

import (
	"context"
	"errors"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// ---------------------------------------------------------------------------
// TestMirrorPointsToWallet
// ---------------------------------------------------------------------------

func TestMirrorPointsToWallet(t *testing.T) {
	t.Parallel()

	guildID := sharedtypes.GuildID("guild-mirror-test")
	roundID := sharedtypes.RoundID(uuid.New())
	clubUUID := uuid.New()
	user1UUID := uuid.New()
	user2UUID := uuid.New()
	discordID1 := sharedtypes.DiscordID("user-1")
	discordID2 := sharedtypes.DiscordID("user-2")

	tests := []struct {
		name   string
		points map[sharedtypes.DiscordID]int
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository, *FakeLeaderboardRepository)
		verify func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository)
	}{
		{
			name: "success: two users get journal entries and balance deltas",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 100,
				discordID2: 50,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, id sharedtypes.DiscordID) (uuid.UUID, error) {
					if id == discordID1 {
						return user1UUID, nil
					}
					return user2UUID, nil
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				trace := repo.Trace()
				journalCount := 0
				deltaCount := 0
				for _, step := range trace {
					switch step {
					case "CreateWalletJournalEntry":
						journalCount++
					case "ApplyWalletBalanceDelta":
						deltaCount++
					}
				}
				if journalCount != 2 {
					t.Errorf("expected 2 CreateWalletJournalEntry calls, got %d", journalCount)
				}
				if deltaCount != 2 {
					t.Errorf("expected 2 ApplyWalletBalanceDelta calls, got %d", deltaCount)
				}
			},
		},
		{
			name: "no club found: GetClubUUIDByDiscordGuildID returns ErrNotFound → nil (no-op)",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 100,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return uuid.Nil, userdb.ErrNotFound
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error for no-club guild, got %v", err)
				}
				for _, step := range repo.Trace() {
					if step == "CreateWalletJournalEntry" || step == "ApplyWalletBalanceDelta" {
						t.Errorf("expected no repo calls for missing club, but got: %s", step)
					}
				}
			},
		},
		{
			name: "user not found: one user errors, other user still processed",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 100,
				discordID2: 50,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, id sharedtypes.DiscordID) (uuid.UUID, error) {
					if id == discordID1 {
						return uuid.Nil, errors.New("user lookup failed")
					}
					return user2UUID, nil
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error (partial success), got %v", err)
				}
				journalCount := 0
				for _, step := range repo.Trace() {
					if step == "CreateWalletJournalEntry" {
						journalCount++
					}
				}
				if journalCount != 1 {
					t.Errorf("expected 1 CreateWalletJournalEntry call (skipped failed user), got %d", journalCount)
				}
			},
		},
		{
			name: "idempotent replay: duplicate key error is silently skipped",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 100,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, id sharedtypes.DiscordID) (uuid.UUID, error) {
					return user1UUID, nil
				}
				repo.CreateWalletJournalEntryFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.WalletJournalEntry) error {
					return errors.New("ERROR: duplicate key value violates unique constraint \"idx_betting_wallet_journal_dedup\"")
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error on idempotent replay, got %v", err)
				}
			},
		},
		{
			name: "zero points skipped: GetUUIDByDiscordID not called for zero-point users",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 0,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error for zero-points map, got %v", err)
				}
				for _, step := range userRepo.Trace() {
					if step == "GetUUIDByDiscordID" {
						t.Error("expected GetUUIDByDiscordID NOT to be called for zero-points users")
					}
				}
				for _, step := range repo.Trace() {
					if step == "CreateWalletJournalEntry" || step == "ApplyWalletBalanceDelta" {
						t.Errorf("expected no repo write calls for zero-points users, but got: %s", step)
					}
				}
			},
		},
		{
			name: "no active season: uses defaultSeasonID",
			points: map[sharedtypes.DiscordID]int{
				discordID1: 75,
			},
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
					return clubUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return nil, nil // no active season
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
					return user1UUID, nil
				}
				repo.CreateWalletJournalEntryFunc = func(_ context.Context, _ bun.IDB, entry *bettingdb.WalletJournalEntry) error {
					if entry.SeasonID != defaultSeasonID {
						t.Errorf("expected defaultSeasonID %q, got %q", defaultSeasonID, entry.SeasonID)
					}
					return nil
				}
			},
			verify: func(t *testing.T, err error, repo *FakeBettingRepository, userRepo *FakeUserRepository) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected nil error with no active season, got %v", err)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := NewFakeBettingRepository()
			userRepo := NewFakeUserRepository()
			guildRepo := NewFakeGuildRepository()
			lbRepo := NewFakeLeaderboardRepository()

			// Default: club exists and active season is set.
			userRepo.GetClubUUIDByDiscordGuildIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (uuid.UUID, error) {
				return clubUUID, nil
			}
			lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
				return &leaderboarddb.Season{ID: "2026-spring"}, nil
			}

			if tc.setup != nil {
				tc.setup(repo, userRepo, guildRepo, lbRepo)
			}

			svc := newTestService(repo, userRepo, guildRepo, lbRepo, nil)

			err := svc.MirrorPointsToWallet(context.Background(), guildID, roundID, tc.points)

			tc.verify(t, err, repo, userRepo)
		})
	}
}
