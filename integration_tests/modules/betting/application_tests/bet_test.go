package bettingintegrationtests

import (
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

func TestPlaceBet(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.PlaceBetRequest)
		expectErr bool
		validate  func(t *testing.T, deps BettingTestDeps, world BettingWorld, ticket *bettingservice.BetTicket)
	}{
		{
			name: "success_bet_placed_and_reserve_increases",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.PlaceBetRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    500,
					Reason:    "test seed",
				})
				if err != nil {
					t.Fatalf("AdjustWallet seed: %v", err)
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
				return world, bettingservice.PlaceBetRequest{
					ClubUUID:     world.ClubUUID,
					UserUUID:     world.MemberUUID,
					RoundID:      roundID,
					SelectionKey: string(world.MemberDiscordID),
					Stake:        50,
				}
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, ticket *bettingservice.BetTicket) {
				if ticket == nil {
					t.Fatal("expected non-nil ticket")
				}
				if ticket.Stake != 50 {
					t.Errorf("expected Stake=50, got %d", ticket.Stake)
				}
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview: %v", err)
				}
				if overview.Wallet.Reserved != 50 {
					t.Errorf("expected Reserved=50 after bet, got %d", overview.Wallet.Reserved)
				}
				if overview.Wallet.Available != 450 {
					t.Errorf("expected Available=450 after bet, got %d", overview.Wallet.Available)
				}
			},
		},
		{
			name: "market_not_found_returns_error",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.PlaceBetRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    500,
					Reason:    "test seed",
				})
				if err != nil {
					t.Fatalf("AdjustWallet seed: %v", err)
				}
				roundRepo := rounddb.NewRepository(deps.BunDB)
				roundID := SeedRound(t, deps.BunDB, roundRepo, world.GuildID)
				// No EnsureMarketsForGuild call — market does not exist.
				return world, bettingservice.PlaceBetRequest{
					ClubUUID:     world.ClubUUID,
					UserUUID:     world.MemberUUID,
					RoundID:      roundID,
					SelectionKey: string(world.MemberDiscordID),
					Stake:        50,
				}
			},
			expectErr: true,
			validate:  nil,
		},
		{
			name: "insufficient_balance_returns_error",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, bettingservice.PlaceBetRequest) {
				world := SeedBettingWorld(t, deps.BunDB)
				roundRepo := rounddb.NewRepository(deps.BunDB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				roundID := SeedRound(t, deps.BunDB, roundRepo, world.GuildID, participants...)
				if _, err := deps.Service.EnsureMarketsForGuild(deps.Ctx, world.GuildID); err != nil {
					t.Fatalf("EnsureMarketsForGuild: %v", err)
				}
				return world, bettingservice.PlaceBetRequest{
					ClubUUID:     world.ClubUUID,
					UserUUID:     world.MemberUUID,
					RoundID:      roundID,
					SelectionKey: string(world.MemberDiscordID),
					Stake:        100,
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

			ticket, err := deps.Service.PlaceBet(deps.Ctx, req)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but PlaceBet succeeded: %+v", ticket)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, deps, world, ticket)
			}
		})
	}
}
