package bettingintegrationtests

import (
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"

	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

func TestVoidRoundMarkets(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, deps BettingTestDeps) (BettingWorld, sharedtypes.RoundID)
		expectErr bool
		validate  func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketVoidResult)
	}{
		{
			name: "no_markets_for_round_noop",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, sharedtypes.RoundID) {
				world := SeedBettingWorld(t, deps.BunDB)
				roundRepo := rounddb.NewRepository(deps.BunDB)
				roundID := SeedRound(t, deps.BunDB, roundRepo, world.GuildID)
				return world, roundID
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketVoidResult) {
				if len(results) != 0 {
					t.Errorf("expected 0 voided markets, got %d", len(results))
				}
			},
		},
		{
			name: "voids_open_market_and_refunds_bet",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, sharedtypes.RoundID) {
				world := SeedBettingWorld(t, deps.BunDB)
				// Give member balance so they can bet.
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    500,
					Reason:    "seed for void test",
				})
				if err != nil {
					t.Fatalf("AdjustWallet: %v", err)
				}
				roundRepo := rounddb.NewRepository(deps.BunDB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				roundID := SeedRound(t, deps.BunDB, roundRepo, world.GuildID, participants...)
				if _, err := deps.Service.EnsureMarketsForGuild(deps.Ctx, world.GuildID); err != nil {
					t.Fatalf("EnsureMarketsForGuild: %v", err)
				}
				_, err = deps.Service.PlaceBet(deps.Ctx, bettingservice.PlaceBetRequest{
					ClubUUID:     world.ClubUUID,
					UserUUID:     world.MemberUUID,
					RoundID:      roundID,
					SelectionKey: string(world.MemberDiscordID),
					Stake:        50,
				})
				if err != nil {
					t.Fatalf("PlaceBet: %v", err)
				}
				return world, roundID
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketVoidResult) {
				if len(results) == 0 {
					t.Fatal("expected at least one voided market")
				}
				// After void, stake should be refunded — reserved back to 0, available back to 500.
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview after void: %v", err)
				}
				if overview.Wallet.Reserved != 0 {
					t.Errorf("expected Reserved=0 after void, got %d", overview.Wallet.Reserved)
				}
				if overview.Wallet.Available != 500 {
					t.Errorf("expected Available=500 after void refund, got %d", overview.Wallet.Available)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world, roundID := tt.setup(t, deps)

			results, err := deps.Service.VoidRoundMarkets(deps.Ctx, world.GuildID, roundID, "test", nil, "integration test void")
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got results: %+v", results)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, deps, world, results)
			}
		})
	}
}
