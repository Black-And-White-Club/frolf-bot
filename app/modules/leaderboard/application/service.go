package leaderboardservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardService handles leaderboard-related logic.
type LeaderboardService struct {
	LeaderboardDB  leaderboarddb.LeaderboardDB
	eventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        leaderboardmetrics.LeaderboardMetrics
	tracer         trace.Tracer
	serviceWrapper func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error)
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(
	db leaderboarddb.LeaderboardDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics leaderboardmetrics.LeaderboardMetrics,
	tracer trace.Tracer,
) Service {
	return &LeaderboardService{
		LeaderboardDB: db,
		eventBus:      eventBus,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		// Assign the serviceWrapper method
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error)) (result LeaderboardOperationResult, err error) {
			return serviceWrapper(ctx, operationName, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error), logger *slog.Logger, metrics leaderboardmetrics.LeaderboardMetrics, tracer trace.Tracer) (result LeaderboardOperationResult, err error) {
	if serviceFunc == nil {
		return LeaderboardOperationResult{}, errors.New("service function is nil")
	}

	// Start a new tracing span while preserving existing context
	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, "LeaderboardService")

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, "LeaderboardService", duration)
	}()

	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)

	// Important: The defer for panic recovery needs to modify the named return values
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = LeaderboardOperationResult{}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	result, err = serviceFunc(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.ExtractCorrelationID(ctx),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(ctx, operationName, "LeaderboardService")

	return result, nil
}

// LeaderboardOperationResult represents a generic result from a leaderboard operation
type LeaderboardOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
