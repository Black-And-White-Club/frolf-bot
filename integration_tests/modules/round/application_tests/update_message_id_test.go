package roundintegrationtests

import (
	"context"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
)

func TestUpdateRoundMessageID(t *testing.T) {
	tests := []struct {
		name           string
		setupTestEnv   func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID)
		newMessageID   string
		validateResult func(t *testing.T, round *roundtypes.Round, newMessageID string)
	}{
		{
			name: "Successfully update round message ID",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, sharedtypes.RoundID) {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID: roundID,
				})
				round.EventMessageID = "old_msg"
				round.GuildID = "test-guild"

				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}
				return "test-guild", roundID
			},
			newMessageID: "new_msg_456",
			validateResult: func(t *testing.T, round *roundtypes.Round, newMessageID string) {
				if round.EventMessageID != newMessageID {
					t.Errorf("Expected EventMessageID %s, got %s", newMessageID, round.EventMessageID)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			guildID, roundID := tt.setupTestEnv(deps.Ctx, deps)

			round, err := deps.Service.UpdateRoundMessageID(deps.Ctx, guildID, roundID, tt.newMessageID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, round, tt.newMessageID)
			}
		})
	}
}
