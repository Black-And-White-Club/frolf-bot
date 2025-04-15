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
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// LeaderboardService handles leaderboard-related logic.
type LeaderboardService struct {
	LeaderboardDB  leaderboarddb.LeaderboardDB
	eventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        leaderboardmetrics.LeaderboardMetrics
	tracer         tempofrolfbot.Tracer
	serviceWrapper func(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (LeaderboardOperationResult, error)
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(
	db leaderboarddb.LeaderboardDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics leaderboardmetrics.LeaderboardMetrics,
	tracer tempofrolfbot.Tracer,
) Service {
	return &LeaderboardService{
		LeaderboardDB: db,
		eventBus:      eventBus,
		logger:        logger,
		metrics:       metrics,
		tracer:        tracer,
		// Assign the serviceWrapper method
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error)) (result LeaderboardOperationResult, err error) {
			return serviceWrapper(ctx, operationName, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(ctx context.Context, operationName string, serviceFunc func() (LeaderboardOperationResult, error), logger *slog.Logger, metrics leaderboardmetrics.LeaderboardMetrics, tracer tempofrolfbot.Tracer) (result LeaderboardOperationResult, err error) {
	if serviceFunc == nil {
		return LeaderboardOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.StartSpan(ctx, operationName, nil)
	defer span.End()

	metrics.RecordOperationAttempt(operationName, "LeaderboardService")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordOperationDuration(operationName, "LeaderboardService", duration)
	}()

	correlationID := attr.ExtractCorrelationID(ctx)
	logger.InfoContext(ctx, "Operation triggered",
		attr.LogAttr(correlationID),
		attr.String("operation", operationName),
	)

	// Important: The defer for panic recovery needs to modify the named return values
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.LogAttr(correlationID),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(operationName, "LeaderboardService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = LeaderboardOperationResult{}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	result, err = serviceFunc()
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.LogAttr(correlationID),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(operationName, "LeaderboardService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.LogAttr(correlationID),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(operationName, "LeaderboardService")

	return result, nil
}

// LeaderboardOperationResult represents a generic result from a leaderboard operation
type LeaderboardOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
