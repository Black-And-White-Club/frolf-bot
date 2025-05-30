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
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
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
// It also includes panic recovery and consistent error handling.
// This function was optimized to reduce memory allocations, particularly related to context and error creation.
func serviceWrapper(ctx context.Context, operationName string, serviceFunc func(ctx context.Context) (LeaderboardOperationResult, error), logger *slog.Logger, metrics leaderboardmetrics.LeaderboardMetrics, tracer trace.Tracer) (result LeaderboardOperationResult, err error) {
	if serviceFunc == nil {
		// Use a predefined error for a static message to avoid repeated allocation.
		// However, in this specific case, errors.New is acceptable as it's an infrequent error path.
		return LeaderboardOperationResult{}, errors.New("service function is nil")
	}

	// Start a new tracing span while preserving existing context.
	// trace.WithAttributes allocates, but is standard for adding span attributes.
	// Consider sampling in your tracing configuration to reduce this overhead in production.
	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
	))
	defer span.End()

	// Record operation attempt metric. Metric recording might involve internal allocations
	// within the metrics library, especially with labels. Review metrics library implementation
	// and label cardinality if this remains a hotspot.
	metrics.RecordOperationAttempt(ctx, operationName, "LeaderboardService")

	startTime := time.Now()
	defer func() {
		// Calculate duration and record metric.
		// time.Since and duration conversion are generally low allocation.
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, "LeaderboardService", duration)
	}()

	// Log operation start. Logging can allocate for message formatting and field handling.
	// Ensure your logger is efficient and consider log levels in production.
	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx), // Extracting correlation ID might involve context lookup but ideally minimal allocation.
		attr.String("operation", operationName),
	)

	// Defer for panic recovery. This uses named return values to set result and err.
	defer func() {
		if r := recover(); r != nil {
			// Creating error message and new error for panic.
			// fmt.Sprintf and errors.New allocate, but are necessary for proper panic reporting.
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r), // attr.Any might allocate depending on the type of 'r'.
			)
			metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
			// Recording error on span allocates for the error object and message.
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases.
			result = LeaderboardOperationResult{}
			err = fmt.Errorf("%s", errorMsg) // fmt.Errorf allocates
		}
	}()

	// Execute the core service function.
	result, err = serviceFunc(ctx)
	if err != nil {
		// Wrap and log the error. fmt.Errorf with %w allocates for the new error object and string formatting.
		// This allocation is generally necessary for proper error chaining and reporting.
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.ExtractCorrelationID(ctx),
			attr.Error(wrappedErr), // attr.Error might allocate depending on the error type.
		)
		metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
		// Recording error on span allocates.
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Log successful completion. Logging allocates.
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

func ptrTag(t sharedtypes.TagNumber) *sharedtypes.TagNumber {
	return &t
}
