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
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories/mocks"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/mock/gomock"
)

// setupBenchmarkService creates a service with mocked dependencies for benchmarking
func setupBenchmarkService(t *testing.B, mockDB *scoredb.MockScoreDB) *ScoreService {
	logger := loggerfrolfbot.NoOpLogger
	metrics := &scoremetrics.NoOpMetrics{}
	tracerProvider := noop.NewTracerProvider()
	tracer := tracerProvider.Tracer("test")

	return &ScoreService{
		ScoreDB: mockDB,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
			result, err := serviceFunc(ctx)
			if err != nil {
				return ScoreOperationResult{Error: err}, err
			}
			return result, nil
		},
	}
}

// generateScores creates test score data for benchmarking
func generateScores(count int, tagPercentage int) []sharedtypes.ScoreInfo {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	scores := make([]sharedtypes.ScoreInfo, count)

	for i := 0; i < count; i++ {
		// Create a basic score with random values
		score := sharedtypes.ScoreInfo{
			UserID: sharedtypes.DiscordID(fmt.Sprintf("%019d", 1000000000000000000+i)),
			Score:  sharedtypes.Score(rng.Intn(20) - 5), // Scores from -5 to +14
		}

		// Add tag based on percentage
		if rng.Intn(100) < tagPercentage {
			tagNum := sharedtypes.TagNumber(rng.Intn(50) + 1) // Tags 1-50
			score.TagNumber = &tagNum
		}

		scores[i] = score
	}

	return scores
}

// BenchmarkProcessRoundScores benchmarks the ProcessRoundScores method
func BenchmarkProcessRoundScores(b *testing.B) {
	ctx := context.Background()
	roundID := sharedtypes.RoundID(uuid.New())

	// Test with different sizes of score sets
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		// Mixed tagged/untagged scenario
		b.Run(fmt.Sprintf("Size-%d-Mixed50Pct", size), func(b *testing.B) {
			scores := generateScores(size, 50) // 50% tagged

			ctrl := gomock.NewController(b)
			defer ctrl.Finish()

			mockDB := scoredb.NewMockScoreDB(ctrl)

			// Configure mock DB to accept any scores
			mockDB.EXPECT().
				LogScores(gomock.Any(), roundID, gomock.Any(), "auto").
				Return(nil).
				AnyTimes()

			s := setupBenchmarkService(b, mockDB)

			b.ResetTimer() // Reset timer before the actual benchmark
			for i := 0; i < b.N; i++ {
				_, err := s.ProcessRoundScores(ctx, roundID, scores)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})

		// All untagged scenario
		b.Run(fmt.Sprintf("Size-%d-Untagged", size), func(b *testing.B) {
			scores := generateScores(size, 0) // 0% tagged

			ctrl := gomock.NewController(b)
			defer ctrl.Finish()

			mockDB := scoredb.NewMockScoreDB(ctrl)

			// Configure mock DB to accept any scores
			mockDB.EXPECT().
				LogScores(gomock.Any(), roundID, gomock.Any(), "auto").
				Return(nil).
				AnyTimes()

			s := setupBenchmarkService(b, mockDB)

			b.ResetTimer() // Reset timer before the actual benchmark
			for i := 0; i < b.N; i++ {
				_, err := s.ProcessRoundScores(ctx, roundID, scores)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})

		// All tagged scenario
		b.Run(fmt.Sprintf("Size-%d-Tagged", size), func(b *testing.B) {
			scores := generateScores(size, 100) // 100% tagged

			ctrl := gomock.NewController(b)
			defer ctrl.Finish()

			mockDB := scoredb.NewMockScoreDB(ctrl)

			// Configure mock DB to accept any scores
			mockDB.EXPECT().
				LogScores(gomock.Any(), roundID, gomock.Any(), "auto").
				Return(nil).
				AnyTimes()

			s := setupBenchmarkService(b, mockDB)

			b.ResetTimer() // Reset timer before the actual benchmark
			for i := 0; i < b.N; i++ {
				_, err := s.ProcessRoundScores(ctx, roundID, scores)
				if err != nil {
					b.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
