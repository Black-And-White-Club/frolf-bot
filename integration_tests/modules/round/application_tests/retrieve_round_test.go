package roundintegrationtests

import (
	"context"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestGetRound(t *testing.T) {
	tests := []struct {
		name            string
		setupTestEnv    func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID)
		expectedFailure bool
		validateResult  func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], roundID sharedtypes.RoundID)
	}{
		{
			name: "Successfully retrieve an existing round",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID) {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Round to retrieve",
				})
				round.GuildID = "test-guild"
				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}
				return "test-guild", roundID
			},
			expectedFailure: false,
			validateResult: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], roundID sharedtypes.RoundID) {
				if res.Success == nil {
					t.Fatalf("Expected success payload, got failure: %+v", res.Failure)
				}
				round := *res.Success
				if round.ID != roundID {
					t.Errorf("Expected round ID %s, got %s", roundID, round.ID)
				}
			},
		},
		{
			name: "Round not found",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID) {
				return "test-guild", sharedtypes.RoundID(uuid.New())
			},
			expectedFailure: true,
			validateResult: func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], roundID sharedtypes.RoundID) {
				if res.Failure == nil {
					t.Fatalf("Expected failure payload, but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			guildID, roundID := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.GetRound(deps.Ctx, guildID, roundID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, roundID)
			}
		})
	}
}

func TestGetRoundsForGuild(t *testing.T) {
	tests := []struct {
		name           string
		setupTestEnv   func(ctx context.Context, deps RoundTestDeps) sharedtypes.GuildID
		validateResult func(t *testing.T, rounds []*roundtypes.Round, guildID sharedtypes.GuildID)
	}{
		{
			name: "Successfully retrieve rounds for a guild",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) sharedtypes.GuildID {
				generator := testutils.NewTestDataGenerator()
				guildID := sharedtypes.GuildID("guild-1")
				r1 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{})
				r1.GuildID = guildID
				r2 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{})
				r2.GuildID = guildID

				_ = deps.DB.CreateRound(ctx, deps.BunDB, guildID, &r1)
				_ = deps.DB.CreateRound(ctx, deps.BunDB, guildID, &r2)

				return guildID
			},
			validateResult: func(t *testing.T, rounds []*roundtypes.Round, guildID sharedtypes.GuildID) {
				if len(rounds) != 2 {
					t.Errorf("Expected 2 rounds, got %d", len(rounds))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			guildID := tt.setupTestEnv(deps.Ctx, deps)

			rounds, err := deps.Service.GetRoundsForGuild(deps.Ctx, guildID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, rounds, guildID)
			}
		})
	}
}
