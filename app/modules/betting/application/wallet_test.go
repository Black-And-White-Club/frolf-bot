package bettingservice

import (
	"context"
	"errors"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"testing"
)

// ---------------------------------------------------------------------------
// TestUpdateSettings
// ---------------------------------------------------------------------------

func TestUpdateSettings(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	userUUID := uuid.New()

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository)
		verify func(t *testing.T, result *MemberSettings, err error, repo *FakeBettingRepository)
	}{
		{
			name: "frozen feature returns error",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(userUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return frozenEntitlements(), nil
				}
			},
			verify: func(t *testing.T, result *MemberSettings, err error, repo *FakeBettingRepository) {
				if !errors.Is(err, ErrFeatureFrozen) {
					t.Errorf("expected ErrFeatureFrozen, got %v", err)
				}
			},
		},
		{
			name: "enabled guild persists updated settings",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(userUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
			},
			verify: func(t *testing.T, result *MemberSettings, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				trace := repo.Trace()
				found := false
				for _, s := range trace {
					if s == "UpsertMemberSettings" {
						found = true
					}
				}
				if !found {
					t.Errorf("expected UpsertMemberSettings in trace, got %v", trace)
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
			result, err := svc.UpdateSettings(context.Background(), UpdateSettingsRequest{
				ClubUUID:        clubUUID,
				UserUUID:        userUUID,
				OptOutTargeting: true,
			})
			tt.verify(t, result, err, repo)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAdjustWallet
// ---------------------------------------------------------------------------

func TestAdjustWallet(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	adminUUID := uuid.New()
	memberID := sharedtypes.DiscordID("member-discord-1")
	memberUUID := uuid.New()

	tests := []struct {
		name   string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository, *FakeLeaderboardRepository)
		verify func(t *testing.T, result *WalletJournal, err error, repo *FakeBettingRepository)
	}{
		{
			name: "non-admin user is rejected",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(adminUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
			},
			verify: func(t *testing.T, result *WalletJournal, err error, repo *FakeBettingRepository) {
				if !errors.Is(err, ErrAdminRequired) {
					t.Errorf("expected ErrAdminRequired, got %v", err)
				}
			},
		},
		{
			name: "admin creates adjustment journal entry with correct season",
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository, lbRepo *FakeLeaderboardRepository) {
				callCount := 0
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, uUID, _ uuid.UUID) (*userdb.ClubMembership, error) {
					callCount++
					// First call is resolveAccess, second is resolveAdminAccess check, third resolves target
					if uUID == adminUUID {
						return adminMembership(adminUUID, clubUUID), nil
					}
					return memberMembership(memberUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
					return memberUUID, nil
				}
				lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
					return &leaderboarddb.Season{ID: "2026-spring"}, nil
				}
				var capturedEntry *bettingdb.WalletJournalEntry
				repo.CreateWalletJournalEntryFunc = func(_ context.Context, _ bun.IDB, entry *bettingdb.WalletJournalEntry) error {
					capturedEntry = entry
					return nil
				}
				repo.CreateAuditLogFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.AuditLog) error { return nil }
				_ = capturedEntry
			},
			verify: func(t *testing.T, result *WalletJournal, err error, repo *FakeBettingRepository) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				trace := repo.Trace()
				foundEntry := false
				foundAudit := false
				for _, s := range trace {
					if s == "CreateWalletJournalEntry" {
						foundEntry = true
					}
					if s == "CreateAuditLog" {
						foundAudit = true
					}
				}
				if !foundEntry {
					t.Error("expected CreateWalletJournalEntry in trace")
				}
				if !foundAudit {
					t.Error("expected CreateAuditLog in trace")
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
			result, err := svc.AdjustWallet(context.Background(), AdjustWalletRequest{
				ClubUUID:  clubUUID,
				AdminUUID: adminUUID,
				MemberID:  memberID,
				Amount:    -50,
				Reason:    "correction",
			})
			tt.verify(t, result, err, repo)
		})
	}
}

// ---------------------------------------------------------------------------
// F9 tests: admin wallet frozen allowance / disabled block
// ---------------------------------------------------------------------------

func TestAdjustWallet_AllowedWhenFrozen(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	adminUUID := uuid.New()
	memberUUID := uuid.New()
	memberID := sharedtypes.DiscordID("member-1")

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()
	lbRepo := NewFakeLeaderboardRepository()

	// Feature is frozen — admin wallet adjustments must still be allowed.
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return frozenEntitlements(), nil
	}
	userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, uID, _ uuid.UUID) (*userdb.ClubMembership, error) {
		if uID == adminUUID {
			return adminMembership(adminUUID, clubUUID), nil
		}
		return memberMembership(memberUUID, clubUUID), nil
	}
	userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
		return "guild-1", nil
	}
	userRepo.GetUUIDByDiscordIDFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.DiscordID) (uuid.UUID, error) {
		return memberUUID, nil
	}
	lbRepo.GetActiveSeasonFunc = func(_ context.Context, _ bun.IDB, _ string) (*leaderboarddb.Season, error) {
		return &leaderboarddb.Season{ID: "2026"}, nil
	}
	repo.CreateWalletJournalEntryFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.WalletJournalEntry) error { return nil }
	repo.CreateAuditLogFunc = func(_ context.Context, _ bun.IDB, _ *bettingdb.AuditLog) error { return nil }

	svc := newTestService(repo, userRepo, guildRepo, lbRepo, nil)
	_, err := svc.AdjustWallet(context.Background(), AdjustWalletRequest{
		ClubUUID:  clubUUID,
		AdminUUID: adminUUID,
		MemberID:  memberID,
		Amount:    50,
		Reason:    "correction during freeze",
	})
	if err != nil {
		t.Fatalf("expected AdjustWallet to be allowed when frozen, got error: %v", err)
	}
}

func TestAdjustWallet_BlockedWhenDisabled(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	adminUUID := uuid.New()

	repo := NewFakeBettingRepository()
	userRepo := NewFakeUserRepository()
	guildRepo := NewFakeGuildRepository()

	// Feature is fully disabled — admin wallet adjustments must be blocked.
	guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
		return disabledEntitlements(), nil
	}
	userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
		return adminMembership(adminUUID, clubUUID), nil
	}
	userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
		return "guild-1", nil
	}

	svc := newTestService(repo, userRepo, guildRepo, NewFakeLeaderboardRepository(), nil)
	_, err := svc.AdjustWallet(context.Background(), AdjustWalletRequest{
		ClubUUID:  clubUUID,
		AdminUUID: adminUUID,
		MemberID:  "member-x",
		Amount:    50,
		Reason:    "should be blocked",
	})
	if err == nil {
		t.Fatal("expected ErrFeatureDisabled when feature is disabled, got nil")
	}
	if !errors.Is(err, ErrFeatureDisabled) {
		t.Errorf("expected ErrFeatureDisabled, got %v", err)
	}
}
