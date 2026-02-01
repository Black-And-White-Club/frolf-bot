package roundintegrationtests

import (
	"context"
	"testing"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestUpdateScheduledRoundsWithNewTags(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) *roundtypes.UpdateScheduledRoundsWithNewTagsRequest
		expectedFailure          bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult[*roundtypes.ScheduledRoundsSyncResult, error])
	}{
		{
			name: "Successful update of scheduled rounds with new tags",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) *roundtypes.UpdateScheduledRoundsWithNewTagsRequest {
				generator := testutils.NewTestDataGenerator()
				guildID := sharedtypes.GuildID("test-guild")

				r1 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{State: roundtypes.RoundStateUpcoming})
				r1.GuildID = guildID
				tag100 := sharedtypes.TagNumber(100)
				r1.Participants = []roundtypes.Participant{{UserID: "user1", TagNumber: &tag100}}

				err := deps.DB.CreateRound(ctx, deps.BunDB, guildID, &r1)
				if err != nil {
					t.Fatalf("Failed to create round: %v", err)
				}

				return &roundtypes.UpdateScheduledRoundsWithNewTagsRequest{
					GuildID: guildID,
					ChangedTags: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
						"user1": 111,
					},
				}
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res results.OperationResult[*roundtypes.ScheduledRoundsSyncResult, error]) {
				if res.Success == nil {
					t.Fatalf("Expected success, got failure: %+v", res.Failure)
				}
				success := *res.Success
				if len(success.Updates) != 1 {
					t.Errorf("Expected 1 update, got %d", len(success.Updates))
				}

				// Verify DB state
				rounds, err := deps.DB.GetUpcomingRounds(ctx, deps.BunDB, "test-guild")
				if err != nil {
					t.Fatalf("Failed to get upcoming rounds: %v", err)
				}
				found := false
				for _, r := range rounds {
					for _, p := range r.Participants {
						if p.UserID == "user1" {
							found = true
							if p.TagNumber == nil || *p.TagNumber != 111 {
								t.Errorf("User1 tag not updated in DB, got %v", p.TagNumber)
							}
						}
					}
				}
				if !found {
					t.Errorf("User1 not found in upcoming rounds")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			payload := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateScheduledRoundsWithNewTags(deps.Ctx, payload)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
