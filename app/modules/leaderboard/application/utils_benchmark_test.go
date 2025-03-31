package leaderboardservice

import (
	"context"
	"fmt"
	"testing"

	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

func BenchmarkGenerateUpdatedLeaderboardSmall(b *testing.B) {
	// Create a test leaderboard with 10 elements
	currentLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 10),
	}
	for i := range currentLeaderboard.LeaderboardData {
		currentLeaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("user%d", i)),
			TagNumber: sharedtypes.TagNumber(i),
		}
	}

	// Create a test sorted participant tags slice with 10 elements
	sortedParticipantTags := make([]string, 10)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("user%d:%d", i, i)
	}

	service := &LeaderboardService{
		logger:  &lokifrolfbot.NoOpLogger{},
		metrics: &leaderboardmetrics.NoOpMetrics{},
		tracer:  tempofrolfbot.NewNoOpTracer(),
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc()
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboard, sortedParticipantTags)
	}
}

func BenchmarkGenerateUpdatedLeaderboardMedium(b *testing.B) {
	// Create a test leaderboard with 100 elements
	currentLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 100),
	}
	for i := range currentLeaderboard.LeaderboardData {
		currentLeaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("user%d", i)),
			TagNumber: sharedtypes.TagNumber(i),
		}
	}

	// Create a test sorted participant tags slice with 100 elements
	sortedParticipantTags := make([]string, 100)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("user%d:%d", i, i)
	}
	service := &LeaderboardService{
		logger:  &lokifrolfbot.NoOpLogger{},
		metrics: &leaderboardmetrics.NoOpMetrics{},
		tracer:  tempofrolfbot.NewNoOpTracer(),
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc()
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboard, sortedParticipantTags)
	}
}

func BenchmarkGenerateUpdatedLeaderboardLarge(b *testing.B) {
	// Create a test leaderboard with 1000 elements
	currentLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 1000),
	}
	for i := range currentLeaderboard.LeaderboardData {
		currentLeaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("user%d", i)),
			TagNumber: sharedtypes.TagNumber(i),
		}
	}

	// Create a test sorted participant tags slice with 1000 elements
	sortedParticipantTags := make([]string, 1000)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("user%d:%d", i, i)
	}

	service := &LeaderboardService{
		logger:  &lokifrolfbot.NoOpLogger{},
		metrics: &leaderboardmetrics.NoOpMetrics{},
		tracer:  tempofrolfbot.NewNoOpTracer(),
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc()
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboard, sortedParticipantTags)
	}
}
