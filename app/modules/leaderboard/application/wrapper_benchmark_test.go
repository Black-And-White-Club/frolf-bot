package leaderboardservice

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/leaderboard"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	"github.com/ThreeDotsLabs/watermill/message"
)

// BenchmarkServiceWrapper benchmarks the performance of the serviceWrapper function
func BenchmarkServiceWrapper(b *testing.B) {
	// Create a simple message

	// Use standard no-op implementations
	logger := &lokifrolfbot.NoOpLogger{}
	metrics := &leaderboardmetrics.NoOpMetrics{}
	tracer := tempofrolfbot.NewNoOpTracer()

	// Define a simple service function that succeeds
	successFunc := func() (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	// Define a simple service function that fails
	failFunc := func() (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func() (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh message copy for each iteration to avoid potential side effects
				ctx := context.Background()

				// Call the service wrapper
				serviceWrapper(ctx, "BenchmarkOperation", bm.serviceFunc, logger, metrics, tracer)
			}
		})
	}
}

// Also benchmark the method-based approach
func BenchmarkServiceWrapperMethod(b *testing.B) {
	// Create a simple message
	msg := message.NewMessage("test-id", []byte("test-payload"))

	// Create a service with no-op implementations
	service := &LeaderboardService{
		logger:  &lokifrolfbot.NoOpLogger{},
		metrics: &leaderboardmetrics.NoOpMetrics{},
		tracer:  tempofrolfbot.NewNoOpTracer(),
	}

	// Define success and failure functions
	successFunc := func() (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{Success: "success"}, nil
	}

	failFunc := func() (LeaderboardOperationResult, error) {
		return LeaderboardOperationResult{}, errors.New("test error")
	}

	// Implement the method-based wrapper function for benchmark
	methodBasedWrapper := func(msg *message.Message, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error) {
		if serviceFunc == nil {
			return LeaderboardOperationResult{}, errors.New("service function is nil")
		}

		ctx, span := service.tracer.StartSpan(msg.Context(), operationName, msg)
		defer span.End()

		msg.SetContext(ctx)

		service.metrics.RecordOperationAttempt(operationName, "LeaderboardService")

		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime).Seconds()
			service.metrics.RecordOperationDuration(operationName, "LeaderboardService", duration)
		}()

		service.logger.Info("Operation triggered",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
			attr.String("operation", operationName),
		)

		defer func() {
			if r := recover(); r != nil {
				errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
				service.logger.Error(errorMsg,
					attr.CorrelationIDFromMsg(msg),
					attr.Any("panic", r),
				)
				service.metrics.RecordOperationFailure(operationName, "LeaderboardService")
				span.RecordError(errors.New(errorMsg))
			}
		}()

		result, err := serviceFunc()
		if err != nil {
			wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
			service.logger.Error("Error in "+operationName,
				attr.CorrelationIDFromMsg(msg),
				attr.Error(wrappedErr),
			)
			service.metrics.RecordOperationFailure(operationName, "LeaderboardService")
			span.RecordError(wrappedErr)
			return result, wrappedErr
		}

		service.logger.Info(operationName+" completed successfully",
			attr.CorrelationIDFromMsg(msg),
			attr.String("operation", operationName),
		)
		service.metrics.RecordOperationSuccess(operationName, "LeaderboardService")

		return result, nil
	}

	// Benchmark scenarios
	benchmarks := []struct {
		name        string
		serviceFunc func() (LeaderboardOperationResult, error)
	}{
		{"SuccessPath", successFunc},
		{"ErrorPath", failFunc},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a fresh message copy for each iteration to avoid potential side effects
				msgCopy := message.NewMessage(msg.UUID, msg.Payload)

				// Call the method-based service wrapper
				methodBasedWrapper(msgCopy, "BenchmarkOperation", bm.serviceFunc)
			}
		})
	}
}
