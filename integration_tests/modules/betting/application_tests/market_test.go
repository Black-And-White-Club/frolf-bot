package bettingintegrationtests

import (
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	bettingservice "github.com/Black-And-White-Club/frolf-bot/app/modules/betting/application"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
)

func TestEnsureMarketsForGuild(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, deps BettingTestDeps) BettingWorld
		expectErr bool
		validate  func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketGeneratedResult)
	}{
		{
			name: "no_upcoming_rounds_returns_empty_result",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				return SeedBettingWorld(t, deps.BunDB)
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketGeneratedResult) {
				if len(results) != 0 {
					t.Errorf("expected 0 market results for no rounds, got %d", len(results))
				}
			},
		},
		{
			name: "creates_winner_market_for_upcoming_round",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				world := SeedBettingWorld(t, deps.BunDB)
				roundRepo := rounddb.NewRepository(deps.BunDB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				SeedRound(t, deps.BunDB, roundRepo, world.GuildID, participants...)
				return world
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketGeneratedResult) {
				if len(results) == 0 {
					t.Error("expected at least one market result")
				}
				markets, err := deps.DB.ListMarketsForClub(deps.Ctx, deps.BunDB, world.ClubUUID, 10)
				if err != nil {
					t.Fatalf("ListMarketsForClub: %v", err)
				}
				if len(markets) == 0 {
					t.Error("expected markets to be persisted in DB")
				}
				for _, m := range markets {
					if m.MarketType == "" {
						t.Error("market should have a market_type")
					}
					if m.Status == "" {
						t.Error("market should have a status")
					}
				}
			},
		},
		{
			name: "idempotent_on_second_call_no_duplicate_markets",
			setup: func(t *testing.T, deps BettingTestDeps) BettingWorld {
				world := SeedBettingWorld(t, deps.BunDB)
				roundRepo := rounddb.NewRepository(deps.BunDB)
				participants := []roundtypes.Participant{
					{UserID: world.MemberDiscordID, Response: roundtypes.ResponseAccept},
					{UserID: world.AdminDiscordID, Response: roundtypes.ResponseAccept},
				}
				SeedRound(t, deps.BunDB, roundRepo, world.GuildID, participants...)
				if _, err := deps.Service.EnsureMarketsForGuild(deps.Ctx, world.GuildID); err != nil {
					t.Fatalf("EnsureMarketsForGuild (first): %v", err)
				}
				return world
			},
			expectErr: false,
			validate: func(t *testing.T, deps BettingTestDeps, world BettingWorld, results []bettingservice.MarketGeneratedResult) {
				markets, err := deps.DB.ListMarketsForClub(deps.Ctx, deps.BunDB, world.ClubUUID, 10)
				if err != nil {
					t.Fatalf("ListMarketsForClub: %v", err)
				}
				// A 2-player field produces winner + over/under = 2 distinct markets.
				// The second call must be idempotent — no new markets created.
				if len(results) != 0 {
					t.Errorf("second call should return 0 new results (idempotent), got %d", len(results))
				}
				if len(markets) != 2 {
					t.Errorf("expected exactly 2 markets for a 2-player field (winner + over/under), got %d", len(markets))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestBettingService(t)
			world := tt.setup(t, deps)

			results, err := deps.Service.EnsureMarketsForGuild(deps.Ctx, world.GuildID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got results: %+v", results)
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
