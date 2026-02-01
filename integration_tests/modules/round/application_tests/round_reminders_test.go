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

func TestProcessRoundReminder(t *testing.T) {
	tests := []struct {
		name           string
		setupTestEnv   func(ctx context.Context, deps RoundTestDeps) *roundtypes.ProcessRoundReminderRequest
		validateResult func(t *testing.T, res results.OperationResult[roundtypes.ProcessRoundReminderResult, error], roundID sharedtypes.RoundID)
	}{
		{
			name: "Successfully process round reminder",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.ProcessRoundReminderRequest {
				roundID := sharedtypes.RoundID(uuid.New())
				generator := testutils.NewTestDataGenerator()
				round := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
					ID:    roundID,
					Title: "Reminder Round",
					State: roundtypes.RoundStateUpcoming,
				})
				round.GuildID = "test-guild"
				round.Participants = []roundtypes.Participant{
					{UserID: "user1", Response: roundtypes.ResponseAccept},
				}

				err := deps.DB.CreateRound(ctx, deps.BunDB, "test-guild", &round)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.ProcessRoundReminderRequest{
					GuildID:      "test-guild",
					RoundID:      roundID,
					ReminderType: "UPCOMING_ROUND",
				}
			},
			validateResult: func(t *testing.T, res results.OperationResult[roundtypes.ProcessRoundReminderResult, error], roundID sharedtypes.RoundID) {
				if res.Success == nil {
					t.Fatalf("Expected success, got failure: %+v", res.Failure)
				}
				success := *res.Success
				if success.RoundID != roundID {
					t.Errorf("Expected round ID %s, got %s", roundID, success.RoundID)
				}
				if len(success.UserIDs) != 1 || success.UserIDs[0] != "user1" {
					t.Errorf("Expected UserIDs ['user1'], got %v", success.UserIDs)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			req := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.ProcessRoundReminder(deps.Ctx, req)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, req.RoundID)
			}
		})
	}
}
