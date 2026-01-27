package scoreservice

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/trace/noop"
)

// setupBenchmarkService creates a service with the FakeScoreRepository
func setupBenchmarkService(b *testing.B) (*ScoreService, *FakeScoreRepository) {
	fakeRepo := NewFakeScoreRepository()

	// Pre-configure the fake to always return success for the benchmark.
	fakeRepo.GetScoresForRoundFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
		return nil, nil
	}
	fakeRepo.LogScoresFunc = func(ctx context.Context, db bun.IDB, gID sharedtypes.GuildID, rID sharedtypes.RoundID, s []sharedtypes.ScoreInfo, src string) error {
		return nil
	}

	s := &ScoreService{
		repo:    fakeRepo,
		logger:  loggerfrolfbot.NoOpLogger,
		metrics: &scoremetrics.NoOpMetrics{},
		tracer:  noop.NewTracerProvider().Tracer("test"),
	}
	return s, fakeRepo
}

// generateScores creates dummy data for the benchmark
func generateScores(count int, tagPercentage int) []sharedtypes.ScoreInfo {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	scores := make([]sharedtypes.ScoreInfo, count)

	for i := 0; i < count; i++ {
		score := sharedtypes.ScoreInfo{
			UserID: sharedtypes.DiscordID(fmt.Sprintf("%019d", 1000000000000000000+i)),
			Score:  sharedtypes.Score(rng.Intn(20) - 5),
		}
		if rng.Intn(100) < tagPercentage {
			tagNum := sharedtypes.TagNumber(rng.Intn(50) + 1)
			score.TagNumber = &tagNum
		}
		scores[i] = score
	}
	return scores
}

func BenchmarkProcessRoundScores(b *testing.B) {
	ctx := context.Background()
	guildID := sharedtypes.GuildID("guild-1234")
	roundID := sharedtypes.RoundID(uuid.New())

	sizes := []int{10, 100, 1000}
	scenarios := []struct {
		name string
		pct  int
	}{
		{"Mixed50Pct", 50},
		{"Untagged", 0},
		{"Tagged", 100},
	}

	for _, size := range sizes {
		for _, scenario := range scenarios {
			b.Run(fmt.Sprintf("Size-%d-%s", size, scenario.name), func(b *testing.B) {
				// 1. Setup Data
				scores := generateScores(size, scenario.pct)
				s, _ := setupBenchmarkService(b)

				// 2. Run Benchmark
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// result type is now results.OperationResult[ProcessRoundScoresResult, error]
					res, err := s.ProcessRoundScores(ctx, guildID, roundID, scores, true)
					if err != nil {
						b.Fatalf("unexpected error: %v", err)
					}

					// Use the result to prevent the compiler from optimizing the call away
					if res.Success != nil && len(res.Success.TagMappings) > size {
						b.Fatal("impossible result")
					}
				}
			})
		}
	}
}
