package roundintegrationtests

import (
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestStartRound(t *testing.T) {
	tests := []struct {
		name            string
		initialSetup    func(t *testing.T, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID)
		expectedFailure bool
		validateResult  func(t *testing.T, res results.OperationResult[*roundtypes.Round, error], roundID sharedtypes.RoundID)
	}{
		{
			name: "Successful round start",
			initialSetup: func(t *testing.T, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID) {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Test Round for Start",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"

				err := deps.DB.CreateRound(deps.Ctx, deps.BunDB, "test-guild", &round)
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
				if round.State != roundtypes.RoundStateInProgress {
					t.Errorf("Expected state InProgress, got %s", round.State)
				}
			},
		},
		{
			name: "Round not found",
			initialSetup: func(t *testing.T, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID) {
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
			guildID, roundID := tt.initialSetup(t, deps)

			result, err := deps.Service.StartRound(deps.Ctx, &roundtypes.StartRoundRequest{
				GuildID: guildID,
				RoundID: roundID,
			})
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, roundID)
			}
		})
	}
}
