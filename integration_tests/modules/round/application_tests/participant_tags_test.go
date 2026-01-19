package roundintegrationtests

import (
	"context"
	"strings"
	"testing"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestUpdateScheduledRoundsWithNewTags(t *testing.T) {
	tests := []struct {
		name                     string
		setupTestEnv             func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber)
		expectedError            bool
		expectedErrorMessagePart string
		validateResult           func(t *testing.T, ctx context.Context, deps RoundTestDeps, returnedResult results.OperationResult)
	}{
		{
			name: "Successful update of scheduled rounds with new tags",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				generator := testutils.NewTestDataGenerator()
				guildID := sharedtypes.GuildID("test-guild")

				// Setup Rounds
				r1 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{State: roundtypes.RoundStateUpcoming})
				r1.GuildID = guildID
				oldTag1, oldTag2 := sharedtypes.TagNumber(100), sharedtypes.TagNumber(200)
				r1.Participants = []roundtypes.Participant{
					{UserID: "user1", TagNumber: &oldTag1},
					{UserID: "user2", TagNumber: &oldTag2},
				}

				r2 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{State: roundtypes.RoundStateUpcoming})
				r2.GuildID = guildID
				oldTag3 := sharedtypes.TagNumber(300)
				r2.Participants = []roundtypes.Participant{
					{UserID: "user1", TagNumber: &oldTag3},
					{UserID: "user4", TagNumber: nil},
				}

				deps.DB.CreateRound(ctx, guildID, &r1)
				deps.DB.CreateRound(ctx, guildID, &r2)

				return guildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber{
					"user1": 111,
					"user2": 222,
					"user5": 555, // Not in rounds
				}
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res results.OperationResult) {
				success := res.Success.(*roundevents.ScheduledRoundsSyncedPayloadV1)

				if len(success.UpdatedRounds) != 2 {
					t.Errorf("Expected 2 updated rounds, got %d", len(success.UpdatedRounds))
				}
				if success.Summary.ParticipantsUpdated != 3 { // user1 (2 rounds) + user2 (1 round)
					t.Errorf("Expected 3 participants updated, got %d", success.Summary.ParticipantsUpdated)
				}

				// Verify DB State
				rounds, _ := deps.DB.GetUpcomingRounds(ctx, "test-guild")
				for _, r := range rounds {
					for _, p := range r.Participants {
						if p.UserID == "user1" && (p.TagNumber == nil || *p.TagNumber != 111) {
							t.Errorf("User1 tag not updated in DB for round %s", r.ID)
						}
					}
				}
			},
		},
		{
			name: "No rounds affected by tag changes",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				guildID := sharedtypes.GuildID("test-guild")
				return guildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber{"nonexistent": 999}
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res results.OperationResult) {
				success := res.Success.(*roundevents.ScheduledRoundsSyncedPayloadV1)
				if success.Summary.RoundsUpdated != 0 {
					t.Errorf("Expected 0 rounds updated, got %d", success.Summary.RoundsUpdated)
				}
			},
		},
		{
			name: "Empty changed tags map returns early success",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				return "test-guild", make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res results.OperationResult) {
				success := res.Success.(*roundevents.ScheduledRoundsSyncedPayloadV1)
				if success.Summary.TotalRoundsProcessed != 0 {
					t.Error("Expected 0 rounds processed for empty map")
				}
			},
		},
		{
			name: "Missing guild ID returns failure payload",
			setupTestEnv: func(ctx context.Context, deps RoundTestDeps) (sharedtypes.GuildID, map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
				return "", map[sharedtypes.DiscordID]sharedtypes.TagNumber{"u1": 1}
			},
			validateResult: func(t *testing.T, ctx context.Context, deps RoundTestDeps, res results.OperationResult) {
				// Assert that Failure is not nil
				if res.Failure == nil {
					t.Fatal("Expected res.Failure to be populated, but got nil")
				}

				// Type assert the interface{} to the expected error payload type
				failurePayload, ok := res.Failure.(*roundevents.RoundUpdateErrorPayloadV1)
				if !ok {
					t.Fatalf("Expected res.Failure to be *RoundUpdateErrorPayloadV1, got %T", res.Failure)
				}

				// Now you can access .Error
				if !strings.Contains(failurePayload.Error, "missing guild_id") {
					t.Errorf("Expected failure message to contain 'missing guild_id', got: %s", failurePayload.Error)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestRoundService(t)
			guildID, changedTags := tt.setupTestEnv(deps.Ctx, deps)

			result, err := deps.Service.UpdateScheduledRoundsWithNewTags(deps.Ctx, guildID, changedTags)

			if (err != nil) != tt.expectedError {
				t.Fatalf("Unexpected error state: %v", err)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, deps.Ctx, deps, result)
			}
		})
	}
}
