package saga

import (
	"context"
	"io"
	"log/slog"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

func TestSwapSagaCoordinator_ProcessIntent(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	guildID := sharedtypes.GuildID("guild-1")

	t.Run("no cycle", func(t *testing.T) {
		kv := NewFakeKeyValue()
		svc := &FakeLeaderboardService{}
		coordinator := NewSwapSagaCoordinator(kv, svc, logger)

		intent := SwapIntent{
			UserID:     sharedtypes.DiscordID("user-1"),
			CurrentTag: sharedtypes.TagNumber(1),
			TargetTag:  sharedtypes.TagNumber(2),
			GuildID:    guildID,
		}

		err := coordinator.ProcessIntent(ctx, intent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(svc.trace) > 0 {
			t.Errorf("expected no service calls without cycle, got %v", svc.trace)
		}

		// Verify intent stored
		entry, _ := kv.Get(ctx, "intents.guild-1.user-1")
		if entry == nil {
			t.Error("intent not stored in KV")
		}
	})

	t.Run("2-way cycle", func(t *testing.T) {
		kv := NewFakeKeyValue()
		var batchRequests []sharedtypes.TagAssignmentRequest
		svc := &FakeLeaderboardService{
			ExecuteBatchTagAssignmentFunc: func(ctx context.Context, gID sharedtypes.GuildID, reqs []sharedtypes.TagAssignmentRequest, uID sharedtypes.RoundID, src sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
				batchRequests = reqs
				return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{}), nil
			},
		}
		coordinator := NewSwapSagaCoordinator(kv, svc, logger)

		// User 2 wants Tag 1 (currently held by User 1)
		intent2 := SwapIntent{UserID: sharedtypes.DiscordID("user-2"), CurrentTag: sharedtypes.TagNumber(2), TargetTag: sharedtypes.TagNumber(1), GuildID: guildID}
		coordinator.ProcessIntent(ctx, intent2)

		// User 1 wants Tag 2 (currently held by User 2) -> COMPLETES CYCLE
		intent1 := SwapIntent{UserID: sharedtypes.DiscordID("user-1"), CurrentTag: sharedtypes.TagNumber(1), TargetTag: sharedtypes.TagNumber(2), GuildID: guildID}
		err := coordinator.ProcessIntent(ctx, intent1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify batch executed
		if len(svc.trace) != 1 || svc.trace[0] != "ExecuteBatchTagAssignment" {
			t.Errorf("expected ExecuteBatchTagAssignment call, got %v", svc.trace)
		}

		if len(batchRequests) != 2 {
			t.Errorf("expected 2 requests in batch, got %d", len(batchRequests))
		}

		// Verify cleanup
		for _, userID := range []string{"user-1", "user-2"} {
			key := "intents.guild-1." + userID
			_, err := kv.Get(ctx, key)
			if err == nil {
				t.Errorf("expected key %s to be deleted after cycle", key)
			}
		}
	})

	t.Run("3-way cycle", func(t *testing.T) {
		kv := NewFakeKeyValue()
		var batchRequests []sharedtypes.TagAssignmentRequest
		svc := &FakeLeaderboardService{
			ExecuteBatchTagAssignmentFunc: func(ctx context.Context, gID sharedtypes.GuildID, reqs []sharedtypes.TagAssignmentRequest, uID sharedtypes.RoundID, src sharedtypes.ServiceUpdateSource) (results.OperationResult[leaderboardtypes.LeaderboardData, error], error) {
				batchRequests = reqs
				return results.SuccessResult[leaderboardtypes.LeaderboardData, error](leaderboardtypes.LeaderboardData{}), nil
			},
		}
		coordinator := NewSwapSagaCoordinator(kv, svc, logger)

		// U1 -> Tag 2 (U2 has it)
		// U2 -> Tag 3 (U3 has it)
		// U3 -> Tag 1 (U1 has it)

		intent1 := SwapIntent{UserID: sharedtypes.DiscordID("u1"), CurrentTag: sharedtypes.TagNumber(1), TargetTag: sharedtypes.TagNumber(2), GuildID: guildID}
		intent2 := SwapIntent{UserID: sharedtypes.DiscordID("u2"), CurrentTag: sharedtypes.TagNumber(2), TargetTag: sharedtypes.TagNumber(3), GuildID: guildID}
		intent3 := SwapIntent{UserID: sharedtypes.DiscordID("u3"), CurrentTag: sharedtypes.TagNumber(3), TargetTag: sharedtypes.TagNumber(1), GuildID: guildID}

		coordinator.ProcessIntent(ctx, intent1)
		coordinator.ProcessIntent(ctx, intent2)
		err := coordinator.ProcessIntent(ctx, intent3) // Completes cycle

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(batchRequests) != 3 {
			t.Errorf("expected 3 requests in batch, got %d", len(batchRequests))
		}
	})
}
