package scoreservice

import (
	"context"
	"fmt"
	"testing"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
)

func BenchmarkProcessScoresForStorage_Scaling(b *testing.B) {
	s := &ScoreService{
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &scoremetrics.NoOpMetrics{},
	}

	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1")
	roundID := sharedtypes.RoundID(uuid.New())

	// Define different scales of data
	scales := []struct {
		name int
	}{
		{10},   // Small group
		{100},  // Large tournament
		{1000}, // Massive league (unlikely, but good for stress testing)
	}

	for _, tc := range scales {
		b.Run(fmt.Sprintf("Players-%d", tc.name), func(b *testing.B) {
			// Generate the baseline data once outside the loop
			originalScores := generateScores(tc.name, 50)

			b.ResetTimer()
			b.ReportAllocs() // This tracks how much memory you are using per operation

			for i := 0; i < b.N; i++ {
				// IMPORTANT: Since ProcessScoresForStorage sorts the slice in-place,
				// we must copy the original data for every iteration.
				// Otherwise, we are benchmarking sorting an already sorted list.
				testData := make([]sharedtypes.ScoreInfo, len(originalScores))
				copy(testData, originalScores)

				_, err := s.ProcessScoresForStorage(ctx, guildID, roundID, testData)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
