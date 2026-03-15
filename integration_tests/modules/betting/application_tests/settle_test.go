package bettingintegrationtests

import (
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

func TestSettleRound(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, deps BettingTestDeps) (BettingWorld, *bettingservice.BettingSettlementRound)
		expectErr bool
		validate  func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketSettlementResult)
	}{
		{
			name: "no_markets_for_round_noop",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, *bettingservice.BettingSettlementRound) {
				world := SeedBettingWorld(t, deps.BunDB)
				roundRepo := rounddb.NewRepository(deps.BunDB)
				roundID := SeedRound(t, deps.BunDB, roundRepo, world.GuildID)
				score := 36
				return world, &bettingservice.BettingSettlementRound{
					ID:        roundID,
					Title:     "Test Round",
					GuildID:   world.GuildID,
					Finalized: true,
					Participants: []bettingservice.BettingSettlementParticipant{
						{MemberID: string(world.MemberDiscordID), Response: string(roundtypes.ResponseAccept), Score: &score},
					},
				}
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketSettlementResult) {
				if len(results) != 0 {
					t.Errorf("expected 0 settled markets for round with no markets, got %d", len(results))
				}
			},
		},
		{
			name: "settles_winner_market_winning_bet_gets_payout",
			setup: func(t *testing.T, deps BettingTestDeps) (BettingWorld, *bettingservice.BettingSettlementRound) {
				world := SeedBettingWorld(t, deps.BunDB)
				_, err := deps.Service.AdjustWallet(deps.Ctx, bettingservice.AdjustWalletRequest{
					ClubUUID:  world.ClubUUID,
					AdminUUID: world.AdminUUID,
					MemberID:  world.MemberDiscordID,
					Amount:    500,
					Reason:    "seed for settle test",
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
				memberScore := 36
				adminScore := 40
				return world, &bettingservice.BettingSettlementRound{
					ID:        roundID,
					Title:     "Test Round",
					GuildID:   world.GuildID,
					Finalized: true,
					Participants: []bettingservice.BettingSettlementParticipant{
						{MemberID: string(world.MemberDiscordID), Response: string(roundtypes.ResponseAccept), Score: &memberScore},
						{MemberID: string(world.AdminDiscordID), Response: string(roundtypes.ResponseAccept), Score: &adminScore},
					},
				}
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketSettlementResult) {
				if len(results) == 0 {
					t.Fatal("expected at least one settled market")
				}
				// After settlement, the reserved balance should be zero (bet resolved).
				overview, err := deps.Service.GetOverview(deps.Ctx, world.ClubUUID, world.MemberUUID)
				if err != nil {
					t.Fatalf("GetOverview after settle: %v", err)
				}
				if overview.Wallet.Reserved != 0 {
					t.Errorf("expected Reserved=0 after settlement, got %d", overview.Wallet.Reserved)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world, finalizedRound := tt.setup(t, deps)

			results, err := deps.Service.SettleRound(deps.Ctx, world.GuildID, finalizedRound, "test", nil, "integration test settle")
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
