package leaderboardservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	loggerfrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// BenchmarkServiceWrapper benchmarks the performance of the standalone serviceWrapper function
func BenchmarkServiceWrapper(b *testing.B) {
	// Create dependencies
	logger := loggerfrolfbot.NoOpLogger
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	// Define a simple service function that succeeds
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	// Define a simple service function that fails
	failFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh context for each iteration
				ctx := context.Background()

				// Call the actual serviceWrapper function
				serviceWrapper(ctx, "BenchmarkOperation", bm.serviceFunc, logger, metrics, tracer)
			}
		})
	}
}

// BenchmarkServiceMethod benchmarks the service's method-based wrapper approach
func BenchmarkServiceMethod(b *testing.B) {
	// Create a simple message
	msg := message.NewMessage("test-id", []byte("test-payload"))

	// Create a service with no-op implementations
	logger := loggerfrolfbot.NoOpLogger
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	service := &LeaderboardService{
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (result LeaderboardOperationResult, err error) {
			return serviceWrapper(ctx, operationName, serviceFunc, logger, metrics, tracer)
		},
	}

	// Define success and failure functions
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	failFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh context for each iteration
				ctx := msg.Context()

				// Call the service's wrapper method
				service.serviceWrapper(ctx, "BenchmarkOperation", bm.serviceFunc)
			}
		})
	}
}

// BenchmarkDirectNoWrapper benchmarks the direct function calls without any wrapper
// to establish a baseline for performance comparison
func BenchmarkDirectNoWrapper(b *testing.B) {
	// Define a simple service function that succeeds
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	// Define a simple service function that fails
	failFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh context for each iteration
				ctx := context.Background()

				// Call the function directly without any wrapper
				_, _ = bm.serviceFunc(ctx)
			}
		})
	}
}

// Mock implementation of the serviceWrapper for more focused testing
func BenchmarkMockServiceWrapper(b *testing.B) {
	// Define a minimal serviceWrapper function that only does essential operations
	mockServiceWrapper := func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
		if serviceFunc == nil {
			return LeaderboardOperationResult{}, errors.New("service function is nil")
		}

		// Skip tracing, metrics, and logging for this mock
		result, err := serviceFunc(ctx)
		if err != nil {
			return result, fmt.Errorf("%s operation failed: %w", operationName, err)
		}

		return result, nil
	}

	// Define success and failure functions
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	failFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh context for each iteration
				ctx := context.Background()

				// Call the mock service wrapper
				mockServiceWrapper(ctx, "BenchmarkOperation", bm.serviceFunc)
			}
		})
	}
}

// BenchmarkTracingOverhead specifically measures the impact of tracing
func BenchmarkTracingOverhead(b *testing.B) {
	// Create dependencies with a real tracer
	tracer := noop.NewTracerProvider().Tracer("test")

	// Success function
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	// Version with tracing
	tracingWrapper := func(ctx context.Context, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
		ctx, span := tracer.Start(ctx, "BenchmarkOperation", trace.WithAttributes(
			attribute.String("operation", "BenchmarkOperation"),
		))
		defer span.End()

		result, err := serviceFunc(ctx)
		if err != nil {
			span.RecordError(err)
		}
		return result, err
	}

	// Version without tracing
	noTracingWrapper := func(ctx context.Context, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
		return serviceFunc(ctx)
	}

	b.Run("WithTracing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			tracingWrapper(ctx, successFunc)
		}
	})

	b.Run("WithoutTracing", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			noTracingWrapper(ctx, successFunc)
		}
	})
}

// Mock implementation of serviceWrapper for components isolation
type componentBenchmark struct {
	name    string
	wrapper func(ctx context.Context, successFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error)
}

// BenchmarkComponentIsolation measures individual components of the wrapper
func BenchmarkComponentIsolation(b *testing.B) {
	// Create dependencies
	logger := loggerfrolfbot.NoOpLogger
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := noop.NewTracerProvider().Tracer("test")

	// Success function
	successFunc := func(ctx context.Context) (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	// Define benchmarks for different components
	benchmarks := []componentBenchmark{
		{
			name: "BaselineNoWrapper",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
				return fn(ctx)
			},
		},
		{
			name: "TracingOnly",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
				ctx, span := tracer.Start(ctx, "BenchmarkOperation")
				defer span.End()
				return fn(ctx)
			},
		},
		{
			name: "LoggingOnly",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
				logger.InfoContext(ctx, "Operation triggered",
					attr.String("operation", "BenchmarkOperation"))
				result, err := fn(ctx)
				if err != nil {
					logger.ErrorContext(ctx, "Error in operation",
						attr.Error(err))
				} else {
					logger.InfoContext(ctx, "Operation completed successfully")
				}
				return result, err
			},
		},
		{
			name: "MetricsOnly",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
				metrics.RecordOperationAttempt(ctx, "BenchmarkOperation", "LeaderboardService")
				startTime := time.Now()
				result, err := fn(ctx)
				duration := time.Duration(time.Since(startTime).Seconds())
				metrics.RecordOperationDuration(ctx, "BenchmarkOperation", "LeaderboardService", duration)

				if err != nil {
					metrics.RecordOperationFailure(ctx, "BenchmarkOperation", "LeaderboardService")
				} else {
					metrics.RecordOperationSuccess(ctx, "BenchmarkOperation", "LeaderboardService")
				}
				return result, err
			},
		},
		{
			name: "PanicRecoveryOnly",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (result LeaderboardOperationResult, err error) {
				defer func() {
					if r := recover(); r != nil {
						result = LeaderboardOperationResult{}
						err = fmt.Errorf("panic recovered: %v", r)
					}
				}()
				return fn(ctx)
			},
		},
		{
			name: "ErrorWrappingOnly",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
				result, err := fn(ctx)
				if err != nil {
					return result, fmt.Errorf("BenchmarkOperation operation failed: %w", err)
				}
				return result, nil
			},
		},
		{
			name: "FullWrapper",
			wrapper: func(ctx context.Context, fn func(ctx context.Context) (LeaderboardOperationResult, error)) (result LeaderboardOperationResult, err error) {
				return serviceWrapper(ctx, "BenchmarkOperation", fn, logger, metrics, tracer)
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				bm.wrapper(ctx, successFunc)
			}
		})
	}
}
