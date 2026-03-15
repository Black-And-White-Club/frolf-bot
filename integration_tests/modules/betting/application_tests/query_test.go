package bettingintegrationtests

import (
	"testing"

	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
)

func TestGetOverview(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, deps BettingTestDeps) BettingWorld
		validate func(t *testing.T, deps BettingTestDeps, world BettingWorld)
	}{
		{
			name: "fresh_user_no_bets_returns_zero_wallet",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				return SeedBettingWorld(t, deps.BunDB)
			},
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld) {
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview returned unexpected error: %v", err)
				}
				if overview == nil {
					t.Fatal("expected non-nil overview")
				}
				if overview.Wallet.Available != 0 {
					t.Errorf("expected Available=0 for fresh user, got %d", overview.Wallet.Available)
				}
				if overview.Wallet.Reserved != 0 {
					t.Errorf("expected Reserved=0 for fresh user, got %d", overview.Wallet.Reserved)
				}
				if overview.AccessState == "" {
					t.Error("expected non-empty AccessState")
				}
				if overview.SeasonID != world.SeasonID {
					t.Errorf("expected SeasonID=%s, got %s", world.SeasonID, overview.SeasonID)
				}
			},
		},
		{
			name: "user_with_wallet_adjustment_reflects_in_overview",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				return SeedBettingWorld(t, deps.BunDB)
			},
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld) {
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    100,
					Reason:    "integration test seeding",
				})
				if err != nil {
					t.Fatalf("AdjustWallet (setup): %v", err)
				}

				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview: %v", err)
				}
				if overview.Wallet.AdjustmentBalance != 100 {
					t.Errorf("expected AdjustmentBalance=100, got %d", overview.Wallet.AdjustmentBalance)
				}
				if overview.Wallet.Available != 100 {
					t.Errorf("expected Available=100, got %d", overview.Wallet.Available)
				}
				if len(overview.Journal) == 0 {
					t.Error("expected at least one journal entry")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world := tt.setup(t, deps)
			tt.validate(t, deps, world)
		})
	}
}

func TestGetAdminMarkets(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, deps BettingTestDeps) BettingWorld
		validate func(t *testing.T, deps BettingTestDeps, world BettingWorld)
	}{
		{
			name: "no_markets_returns_empty_board",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				return SeedBettingWorld(t, deps.BunDB)
			},
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld) {
				board, err := deps.Service.GetAdminMarkets(deps.Ctx, world.ClubUUID, world.AdminUUID)
				if err != nil {
					t.Fatalf("GetAdminMarkets: %v", err)
				}
				if board == nil {
					t.Fatal("expected non-nil board")
				}
				if len(board.Markets) != 0 {
					t.Errorf("expected 0 markets, got %d", len(board.Markets))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world := tt.setup(t, deps)
			tt.validate(t, deps, world)
		})
	}
}
