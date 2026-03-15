package bettingservice

import (
	"context"
	"errors"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	bettingdb "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/infrastructure/repositories"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"testing"
)

// ---------------------------------------------------------------------------
// TestAdminMarketAction
// ---------------------------------------------------------------------------

func TestAdminMarketAction(t *testing.T) {
	t.Parallel()

	clubUUID := uuid.New()
	adminUUID := uuid.New()
	memberUUID := uuid.New()
	marketID := int64(77)
	roundUUID := uuid.New()

	tests := []struct {
		name   string
		action string
		setup  func(*FakeBettingRepository, *FakeUserRepository, *FakeGuildRepository)
		verify func(t *testing.T, result *AdminMarketActionResult, err error)
	}{
		{
			name:   "nil club/admin UUID rejected immediately",
			action: adminActionVoid,
			setup:  func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {},
			verify: func(t *testing.T, result *AdminMarketActionResult, err error) {
				if !errors.Is(err, ErrAdminRequired) {
					t.Errorf("expected ErrAdminRequired for nil UUIDs, got %v", err)
				}
			},
		},
		{
			name:   "non-admin user is rejected",
			action: adminActionVoid,
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, uUID, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return memberMembership(memberUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
			},
			verify: func(t *testing.T, result *AdminMarketActionResult, err error) {
				if !errors.Is(err, ErrAdminRequired) {
					t.Errorf("expected ErrAdminRequired, got %v", err)
				}
			},
		},
		{
			name:   "admin void action voids market and refunds bets",
			action: adminActionVoid,
			setup: func(repo *FakeBettingRepository, userRepo *FakeUserRepository, guildRepo *FakeGuildRepository) {
				userRepo.GetClubMembershipFunc = func(_ context.Context, _ bun.IDB, _, _ uuid.UUID) (*userdb.ClubMembership, error) {
					return adminMembership(adminUUID, clubUUID), nil
				}
				userRepo.GetDiscordGuildIDByClubUUIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID) (sharedtypes.GuildID, error) {
					return "guild-1", nil
				}
				guildRepo.ResolveEntitlementsFunc = func(_ context.Context, _ bun.IDB, _ sharedtypes.GuildID) (guildtypes.ResolvedClubEntitlements, error) {
					return enabledEntitlements(), nil
				}
				repo.GetMarketByIDFunc = func(_ context.Context, _ bun.IDB, _ uuid.UUID, _ int64) (*bettingdb.Market, error) {
					return &bettingdb.Market{ID: marketID, ClubUUID: clubUUID, MarketType: winnerMarketType, Status: openMarketStatus, RoundID: roundUUID}, nil
				}
				repo.ListBetsForMarketFunc = func(_ context.Context, _ bun.IDB, _ int64) ([]bettingdb.Bet, error) {
					return []bettingdb.Bet{
						{ID: 1, MarketID: marketID, ClubUUID: clubUUID, Stake: 100, Status: acceptedBetStatus},
					}, nil
				}
			},
			verify: func(t *testing.T, result *AdminMarketActionResult, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result == nil {
					t.Fatal("expected non-nil result")
				}
				if result.Action != adminActionVoid {
					t.Errorf("Action: want %s, got %s", adminActionVoid, result.Action)
				}
				if result.Status != voidedMarketStatus {
					t.Errorf("Status: want %s, got %s", voidedMarketStatus, result.Status)
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
			tt.setup(repo, userRepo, guildRepo)
			svc := newTestService(repo, userRepo, guildRepo, NewFakeLeaderboardRepository(), nil)

			// For the nil UUID test, use zero values
			actorClubUUID := clubUUID
			actorAdminUUID := adminUUID
			if tt.name == "nil club/admin UUID rejected immediately" {
				actorClubUUID = uuid.Nil
				actorAdminUUID = uuid.Nil
			}

			result, err := svc.AdminMarketAction(context.Background(), AdminMarketActionRequest{
				ClubUUID:  actorClubUUID,
				AdminUUID: actorAdminUUID,
				MarketID:  marketID,
				Action:    tt.action,
				Reason:    "test reason",
			})
			tt.verify(t, result, err)
		})
	}
}
