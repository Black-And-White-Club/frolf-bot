package scoreservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ScoreService handles score processing logic.
type ScoreService struct {
	ScoreDB        scoredb.ScoreDB
	EventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        scoremetrics.ScoreMetrics
	tracer         trace.Tracer
	serviceWrapper func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error)
}

// NewScoreService creates a new ScoreService.
func NewScoreService(
	db scoredb.ScoreDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer trace.Tracer,
) Service {
	return &ScoreService{
		ScoreDB:  db,
		EventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
			return serviceWrapper(ctx, operationName, roundID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(
	ctx context.Context,
	operationName string,
	roundID sharedtypes.RoundID,
	serviceFunc func(ctx context.Context) (ScoreOperationResult, error),
	logger *slog.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer trace.Tracer,
) (result ScoreOperationResult, err error) {
	if serviceFunc == nil {
		return ScoreOperationResult{}, errors.New("service function is nil")
	}

	// Start a new tracing span while preserving existing context
	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	// Record the operation attempt
	metrics.RecordOperationAttempt(ctx, operationName, roundID)

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, duration)
	}()

	// Log the operation start
	logger.InfoContext(ctx, operationName+" triggered",
		attr.String("operation", operationName),
		attr.RoundID("round_id", roundID),
		attr.ExtractCorrelationID(ctx),
	)

	// Recover from panic and log the error
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.RoundID("round_id", roundID),
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, roundID)

			// Create error with panic information
			err = fmt.Errorf("panic in %s: %v", operationName, r)

			// Record error in span
			span.RecordError(err)
		}
	}()

	// Call the service function
	result, err = serviceFunc(ctx)

	// If the service function returned an error AND did NOT populate a Failure payload,
	// then it's a true system error that should be propagated.
	if err != nil && result.Failure == nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.RoundID("round_id", roundID),
			attr.ExtractCorrelationID(ctx),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, roundID)
		span.RecordError(wrappedErr)
		return ScoreOperationResult{}, wrappedErr // Return empty result and wrapped error
	}

	// If there was an error but a Failure payload was populated, or no error occurred,
	// log success and return the result.
	if err == nil || result.Failure != nil { // This condition explicitly handles both success and handled business failures
		logger.InfoContext(ctx, operationName+" completed successfully",
			attr.String("operation", operationName),
			attr.RoundID("round_id", roundID),
			attr.ExtractCorrelationID(ctx),
		)
		metrics.RecordOperationSuccess(ctx, operationName, roundID)
		return result, nil // Return the actual result and nil error
	}

	// Fallback for unexpected scenarios (should ideally not be reached)
	return ScoreOperationResult{}, fmt.Errorf("unexpected state after %s operation", operationName)
}

// ScoreOperationResult represents a generic result from a score operation
type ScoreOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
