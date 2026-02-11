package leaderboardintegrationtests

import (
	"context"
	"errors"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestExecuteBatchTagAssignment_Integration(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()
	ctx := context.Background()

	t.Run("successful complex internal swap", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "user_1", TagNumber: 1},
			{UserID: "user_2", TagNumber: 2},
		}, true, sharedtypes.RoundID{})

		result, err := deps.Service.ExecuteBatchTagAssignment(
			ctx,
			"test_guild",
			[]sharedtypes.TagAssignmentRequest{
				{UserID: "user_1", TagNumber: 2},
				{UserID: "user_2", TagNumber: 3},
			},
			sharedtypes.RoundID{},
			sharedtypes.ServiceUpdateSourceManual,
		)
		if err != nil {
			t.Fatalf("expected success, got %v", err)
		}
		if result.Success == nil || len(*result.Success) != 2 {
			t.Fatalf("expected two entries in result, got %+v", result)
		}

		finalData, qErr := testutils.QueryLeaderboardData(t, ctx, deps.BunDB, "test_guild")
		if qErr != nil {
			t.Fatalf("failed to query final state: %v", qErr)
		}
		finalMap := testutils.ExtractLeaderboardDataMap(finalData)
		if finalMap["user_1"] != 2 || finalMap["user_2"] != 3 {
			t.Fatalf("unexpected final assignments: %+v", finalMap)
		}
	})

	t.Run("conflict with external user returns TagSwapNeededError", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "user_external", TagNumber: 10},
		}, true, sharedtypes.RoundID{})

		result, err := deps.Service.ExecuteBatchTagAssignment(
			ctx,
			"test_guild",
			[]sharedtypes.TagAssignmentRequest{{UserID: "user_new", TagNumber: 10}},
			sharedtypes.RoundID{},
			sharedtypes.ServiceUpdateSourceManual,
		)
		if err != nil {
			t.Fatalf("expected no system error, got %v", err)
		}
		if result.Failure == nil {
			t.Fatalf("expected failure payload, got %+v", result)
		}

		var swapErr *leaderboardservice.TagSwapNeededError
		if !errors.As(*result.Failure, &swapErr) {
			t.Fatalf("expected TagSwapNeededError, got %v", *result.Failure)
		}
		if swapErr.TargetUserID != "user_external" {
			t.Fatalf("expected conflict with user_external, got %s", swapErr.TargetUserID)
		}
	})
}
