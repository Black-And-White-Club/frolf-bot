package scoreservice

// import (
// 	"context"
// 	"fmt"
// 	"math/rand"
// 	"testing"
// 	"time"

// 	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
// 	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/score"
// 	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
// 	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
// )

// // setupBenchmarkService creates a test score service with no-op dependencies
// func setupBenchmarkService() *ScoreService {
// 	logger := &lokifrolfbot.NoOpLogger{}
// 	metrics := &scoremetrics.NoOpMetrics{}
// 	tracer := tempofrolfbot.NewNoOpTracer()

// 	return &ScoreService{
// 		logger:  logger,
// 		metrics: metrics,
// 		tracer:  tracer,
// 	}
// }

// // generateScores creates test score data with optional configuration
// func generateScores(count int, tagPercentage int) []sharedtypes.ScoreInfo {
// 	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
// 	scores := make([]sharedtypes.ScoreInfo, count)

// 	for i := 0; i < count; i++ {
// 		// Create a basic score with random values
// 		score := sharedtypes.ScoreInfo{
// 			UserID: sharedtypes.DiscordID(fmt.Sprintf("%019d", 1000000000000000000+i)),
// 			Score:  sharedtypes.Score(rng.Intn(20) - 5), // Scores from -5 to +14
// 		}

// 		// Add tag based on percentage
// 		if rng.Intn(100) < tagPercentage {
// 			tagNum := sharedtypes.TagNumber(rng.Intn(50) + 1) // Tags 1-50
// 			score.TagNumber = &tagNum
// 		}

// 		scores[i] = score
// 	}

// 	return scores
// }

// // BenchmarkProcessScoresForStorage benchmarks the ProcessScoresForStorage method
// func BenchmarkProcessScoresForStorage(b *testing.B) {
// 	service := setupBenchmarkService()
// 	ctx := context.Background()
// 	roundID := sharedtypes.RoundID("benchmark-round-id")

// 	// Test with different sizes of score sets
// 	sizes := []int{10, 100, 1000, 10000}

// 	for _, size := range sizes {
// 		// Standard benchmark - Mixed tagged/untagged
// 		b.Run(fmt.Sprintf("Size-%d-Mixed50Pct", size), func(b *testing.B) {
// 			scores := generateScores(size, 50) // 50% tagged

// 			b.ResetTimer()
// 			b.ReportAllocs()

// 			for i := 0; i < b.N; i++ {
// 				_, err := service.ProcessScoresForStorage(ctx, roundID, scores)
// 				if err != nil {
// 					b.Fatal(err)
// 				}
// 			}
// 		})

// 		// All tagged scenario
// 		b.Run(fmt.Sprintf("Size-%d-AllTagged", size), func(b *testing.B) {
// 			scores := generateScores(size, 100) // 100% tagged

// 			b.ResetTimer()
// 			b.ReportAllocs()

// 			for i := 0; i < b.N; i++ {
// 				_, err := service.ProcessScoresForStorage(ctx, roundID, scores)
// 				if err != nil {
// 					b.Fatal(err)
// 				}
// 			}
// 		})
// 		// All untagged scenario
// 		b.Run(fmt.Sprintf("Size-%d-AllUntagged", size), func(b *testing.B) {
// 			scores := generateScores(size, 0) // 0% tagged

// 			b.ResetTimer()
// 			b.ReportAllocs()

// 			for i := 0; i < b.N; i++ {
// 				_, err := service.ProcessScoresForStorage(ctx, roundID, scores)
// 				if err != nil {
// 					b.Fatal(err)
// 				}
// 			}
// 		})
// 	}
// }
