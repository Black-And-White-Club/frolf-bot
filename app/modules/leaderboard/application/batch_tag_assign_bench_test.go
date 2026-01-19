package leaderboardservice

import (
	"fmt"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

// buildRequests creates N tag assignment requests by rotating existing users' tags.
// This reflects real-world batch updates where all users already exist.
func buildRequests(
	data leaderboardtypes.LeaderboardData,
	n int,
) []sharedtypes.TagAssignmentRequest {
	reqs := make([]sharedtypes.TagAssignmentRequest, 0, n)
	if len(data) == 0 {
		return reqs
	}

	for i := 0; i < n; i++ {
		idx := i % len(data)
		newTag := sharedtypes.TagNumber((i % len(data)) + 1)

		reqs = append(reqs, sharedtypes.TagAssignmentRequest{
			UserID:    data[idx].UserID,
			TagNumber: newTag,
		})
	}

	return reqs
}

// newBenchmarkService constructs a minimal LeaderboardService with no-op dependencies.
func newBenchmarkService() *LeaderboardService {
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("bench")

	return &LeaderboardService{
		logger:  loggerfrolfbot.NoOpLogger,
		tracer:  tracer,
		metrics: &leaderboardmetrics.NoOpMetrics{},
	}
}

// createBenchmarkLeaderboardData builds a leaderboard with sequential tag numbers.
func createBenchmarkLeaderboardData(n int) leaderboardtypes.LeaderboardData {
	data := make(leaderboardtypes.LeaderboardData, n)
	for i := 0; i < n; i++ {
		data[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("benchuser%d", i)),
			TagNumber: sharedtypes.TagNumber(i + 1),
		}
	}
	return data
}

// ----------------------
// GenerateUpdatedSnapshot Benchmarks
// ----------------------

func BenchmarkGenerateUpdatedSnapshot_Small(b *testing.B) {
	svc := newBenchmarkService()
	current := createBenchmarkLeaderboardData(100)
	reqs := buildRequests(current, 100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = svc.GenerateUpdatedSnapshot(current, reqs)
	}
}

func BenchmarkGenerateUpdatedSnapshot_Medium(b *testing.B) {
	svc := newBenchmarkService()
	current := createBenchmarkLeaderboardData(1_000)
	reqs := buildRequests(current, 1_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = svc.GenerateUpdatedSnapshot(current, reqs)
	}
}

func BenchmarkGenerateUpdatedSnapshot_Large(b *testing.B) {
	svc := newBenchmarkService()
	current := createBenchmarkLeaderboardData(5_000)
	reqs := buildRequests(current, 5_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = svc.GenerateUpdatedSnapshot(current, reqs)
	}
}

// XLarge helps reveal when sorting dominates CPU time.
func BenchmarkGenerateUpdatedSnapshot_XLarge(b *testing.B) {
	svc := newBenchmarkService()
	current := createBenchmarkLeaderboardData(20_000)
	reqs := buildRequests(current, 20_000)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = svc.GenerateUpdatedSnapshot(current, reqs)
	}
}
