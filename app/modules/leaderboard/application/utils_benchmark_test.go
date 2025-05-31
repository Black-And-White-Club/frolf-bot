package leaderboardservice

import (
	"context"
	"fmt" // Import strconv for parsing tag numbers

	// Import strings for splitting tag strings
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"go.opentelemetry.io/otel/trace/noop"
)

// Helper functions from benchmark tests (included for completeness if not in a shared testutils file)
// If these are already in a shared file, you can remove them from here.
func createBenchmarkLeaderboardData(n int) leaderboardtypes.LeaderboardData {
	data := make(leaderboardtypes.LeaderboardData, n)
	for i := 0; i < n; i++ {
		data[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("existinguser%d", i)),
			TagNumber: sharedtypes.TagNumber(i + 1),
		}
	}
	return data
}

func createBenchmarkSortedParticipantTags(size int) []string {
	tags := make([]string, size)
	for i := range tags {
		// Assuming sequential new tags based on performance order (0 to size-1)
		// and user IDs corresponding to their original index.
		newTag := i + 1 // New tags start from 1
		userID := fmt.Sprintf("existinguser%d", i)
		tags[i] = fmt.Sprintf("%d:%s", newTag, userID)
	}
	return tags
}

func BenchmarkGenerateUpdatedLeaderboardDataSmall(b *testing.B) {
	// Create a test leaderboard data slice with 10 elements
	currentLeaderboardData := createBenchmarkLeaderboardData(10)

	// Create a test sorted participant tags slice with 10 elements
	sortedParticipantTags := createBenchmarkSortedParticipantTags(10)

	// Initialize service with no-op dependencies for benchmarking
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	service := &LeaderboardService{
		logger:  loggerfrolfbot.NoOpLogger,
		tracer:  tracer,
		metrics: &leaderboardmetrics.NoOpMetrics{},
		// The serviceWrapper is not typically used in benchmarks as we're testing the core logic
		// directly, but including it if your service relies on it for context propagation etc.
		// If serviceWrapper adds significant overhead, consider removing it for pure function benchmarks.
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Call the correctly named function and pass the data slice
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboardData, sortedParticipantTags)
	}
}

func BenchmarkGenerateUpdatedLeaderboardDataMedium(b *testing.B) {
	// Create a test leaderboard data slice with 100 elements
	currentLeaderboardData := createBenchmarkLeaderboardData(100)

	// Create a test sorted participant tags slice with 100 elements
	sortedParticipantTags := createBenchmarkSortedParticipantTags(100)

	// Initialize service with no-op dependencies for benchmarking
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	service := &LeaderboardService{
		logger:  loggerfrolfbot.NoOpLogger,
		tracer:  tracer,
		metrics: &leaderboardmetrics.NoOpMetrics{},
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Call the correctly named function and pass the data slice
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboardData, sortedParticipantTags)
	}
}

func BenchmarkGenerateUpdatedLeaderboardDataLarge(b *testing.B) {
	// Create a test leaderboard data slice with 1000 elements
	currentLeaderboardData := createBenchmarkLeaderboardData(1000)

	// Create a test sorted participant tags slice with 1000 elements
	sortedParticipantTags := createBenchmarkSortedParticipantTags(1000)

	// Initialize service with no-op dependencies for benchmarking
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	service := &LeaderboardService{
		logger:  loggerfrolfbot.NoOpLogger,
		tracer:  tracer,
		metrics: &leaderboardmetrics.NoOpMetrics{},
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
			return serviceFunc(ctx)
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Call the correctly named function and pass the data slice
		_, _ = service.GenerateUpdatedLeaderboard(currentLeaderboardData, sortedParticipantTags)
	}
}
