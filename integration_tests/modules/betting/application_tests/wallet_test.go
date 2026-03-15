package bettingintegrationtests

import (
	"testing"

	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
)

func TestAdjustWallet(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.AdjustWalletRequest)
		expectErr bool
		validate  func(t *testing.T, deps BettingTestDeps, world BettingWorld)
	}{
		{
			name: "positive_adjustment_creates_journal_entry",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.AdjustWalletRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				return world, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    200,
					Reason:    "prize payout",
				}
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld) {
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview after adjustment: %v", err)
				}
				if overview.Wallet.AdjustmentBalance != 200 {
					t.Errorf("expected AdjustmentBalance=200, got %d", overview.Wallet.AdjustmentBalance)
				}
				if overview.Wallet.Available != 200 {
					t.Errorf("expected Available=200, got %d", overview.Wallet.Available)
				}
			},
		},
		{
			name: "negative_adjustment_decreases_balance",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.AdjustWalletRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				// First give them 100 so negative doesn't go below zero in a surprising way.
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    100,
					Reason:    "seed",
				})
				if err != nil {
					t.Fatalf("seed adjustment: %v", err)
				}
				return world, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    -30,
					Reason:    "correction",
				}
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld) {
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview after negative adjustment: %v", err)
				}
				if overview.Wallet.AdjustmentBalance != 70 {
					t.Errorf("expected AdjustmentBalance=70, got %d", overview.Wallet.AdjustmentBalance)
				}
			},
		},
		{
			name: "zero_amount_rejected",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.AdjustWalletRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				return world, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    0,
					Reason:    "zero",
				}
			},
			expectErr: true,
			validate:  nil,
		},
		{
			name: "empty_reason_rejected",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.AdjustWalletRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				return world, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    50,
					Reason:    "",
				}
			},
			expectErr: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world, req := tt.setup(t, deps)

			result, err := deps.Service.AdjustWallet(deps.Ctx, req)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got result: %+v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.validate != nil {
				tt.validate(t, deps, world)
			}
		})
	}
}
