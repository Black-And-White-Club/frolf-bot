package leaderboardintegrationtests

import (
	"context"
	"strings"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestTagSwapRequested(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()
	ctx := context.Background()

	t.Run("successful swap between two users", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "user_swap_A", TagNumber: 10},
			{UserID: "user_swap_B", TagNumber: 20},
			{UserID: "other_user", TagNumber: 30},
		}, true, sharedtypes.RoundID{})

		result, err := deps.Service.TagSwapRequested(ctx, "test_guild", "user_swap_A", 20)
		if err != nil {
			t.Fatalf("unexpected system error: %v", err)
		}
		if result.Success == nil {
			t.Fatalf("expected success payload, got %+v", result)
		}

		finalData, qErr := testutils.QueryLeaderboardData(t, ctx, deps.BunDB, "test_guild")
		if qErr != nil {
			t.Fatalf("failed to query final state: %v", qErr)
		}
		finalMap := testutils.ExtractLeaderboardDataMap(finalData)
		if finalMap["user_swap_A"] != 20 || finalMap["user_swap_B"] != 10 {
			t.Fatalf("expected swapped tags, got %+v", finalMap)
		}
	})

	t.Run("fails when requestor is not on leaderboard", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "user_swap_B", TagNumber: 20},
		}, true, sharedtypes.RoundID{})

		result, err := deps.Service.TagSwapRequested(ctx, "test_guild", "stranger_danger", 20)
		if err != nil {
			t.Fatalf("unexpected system error: %v", err)
		}
		if result.Failure == nil {
			t.Fatalf("expected failure payload, got %+v", result)
		}
		if !strings.Contains((*result.Failure).Error(), "requesting user not on leaderboard") {
			t.Fatalf("unexpected failure: %v", *result.Failure)
		}
	})
}
