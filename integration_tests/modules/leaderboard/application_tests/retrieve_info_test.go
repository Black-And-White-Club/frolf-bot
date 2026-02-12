package leaderboardintegrationtests

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestLeaderboardReadOperations(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	ctx := context.Background()

	t.Run("GetLeaderboard returns data", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "user_1", TagNumber: 1},
		}, true, sharedtypes.RoundID{})

		res, err := deps.Service.GetLeaderboard(ctx, "test_guild", "")
		if err != nil {
			t.Fatalf("GetLeaderboard failed: %v", err)
		}
		if res.Success == nil || len(*res.Success) != 1 {
			t.Fatalf("expected one leaderboard entry, got %+v", res)
		}
	})

	t.Run("GetLeaderboard empty guild returns empty data", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		res, err := deps.Service.GetLeaderboard(ctx, "empty_guild", "")
		if err != nil {
			t.Fatalf("GetLeaderboard failed: %v", err)
		}
		if res.Success == nil {
			t.Fatalf("expected success result, got %+v", res)
		}
		if len(*res.Success) != 0 {
			t.Fatalf("expected empty leaderboard, got %d entries", len(*res.Success))
		}
	})

	t.Run("RoundGetTagByUserID returns existing tag", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "round_user", TagNumber: 42},
		}, true, sharedtypes.RoundID{})

		res, err := deps.Service.RoundGetTagByUserID(ctx, "test_guild", "round_user")
		if err != nil {
			t.Fatalf("RoundGetTagByUserID failed: %v", err)
		}
		if res.Success == nil || *res.Success != 42 {
			t.Fatalf("expected tag 42, got %+v", res)
		}
	})

	t.Run("GetTagByUserID returns existing tag", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{
			{UserID: "raw_user", TagNumber: 10},
		}, true, sharedtypes.RoundID{})

		res, err := deps.Service.GetTagByUserID(ctx, "test_guild", "raw_user")
		if err != nil {
			t.Fatalf("GetTagByUserID failed: %v", err)
		}
		if res.Success == nil || *res.Success != 10 {
			t.Fatalf("expected tag 10, got %+v", res)
		}
	})

	t.Run("GetTagByUserID missing user returns sql.ErrNoRows failure", func(t *testing.T) {
		_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)
		testutils.SetupLeaderboardWithEntries(t, deps.BunDB, leaderboardtypes.LeaderboardData{}, true, sharedtypes.RoundID{})

		opResult, err := deps.Service.GetTagByUserID(ctx, "test_guild", "ghost_user")
		if err != nil {
			t.Fatalf("expected no system error, got %v", err)
		}
		if opResult.Failure == nil {
			t.Fatalf("expected domain failure, got %+v", opResult)
		}
		if !errors.Is(*opResult.Failure, sql.ErrNoRows) {
			t.Fatalf("expected sql.ErrNoRows, got %v", *opResult.Failure)
		}
	})
}

var _ leaderboardservice.Service
var _ results.OperationResult[sharedtypes.TagNumber, error]
