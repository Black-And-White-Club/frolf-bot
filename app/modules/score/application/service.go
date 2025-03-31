package scoreservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
)

// ScoreService handles score processing logic.
type ScoreService struct {
	ScoreDB        scoredb.ScoreDB
	EventBus       eventbus.EventBus
	logger         lokifrolfbot.Logger
	metrics        scoremetrics.ScoreMetrics
	tracer         tempofrolfbot.Tracer
	serviceWrapper func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error)
}

// NewScoreService creates a new ScoreService.
func NewScoreService(
	db scoredb.ScoreDB,
	eventBus eventbus.EventBus,
	logger lokifrolfbot.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer tempofrolfbot.Tracer,
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
	logger lokifrolfbot.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer tempofrolfbot.Tracer,
) (result ScoreOperationResult, err error) {
	if serviceFunc == nil {
		return ScoreOperationResult{}, errors.New("service function is nil")
	}

	// Start a new tracing span while preserving existing context
	newCtx, span := tracer.StartSpan(ctx, operationName, nil)
	defer span.End()

	// Extract correlation ID (if available)
	correlationID := attr.ExtractCorrelationID(ctx)

	// Record the operation attempt
	metrics.RecordOperationAttempt(operationName, roundID)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordOperationDuration(operationName, duration)
	}()

	// Log the operation start
	logger.Info(operationName+" triggered",
		attr.String("operation", operationName),
		attr.RoundID("round_id", roundID),
		attr.LogAttr(correlationID),
	)

	// Recover from panic and log the error
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.Error(errorMsg,
				attr.RoundID("round_id", roundID),
				attr.LogAttr(correlationID),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(operationName, roundID)

			// Create error with panic information
			err = fmt.Errorf("panic in %s: %v", operationName, r)

			// Record error in span
			span.RecordError(err)
		}
	}()

	// Call the service function **with new context**
	result, err = serviceFunc(newCtx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.Error("Error in "+operationName,
			attr.RoundID("round_id", roundID),
			attr.LogAttr(correlationID),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(operationName, roundID)
		span.RecordError(wrappedErr)
		return ScoreOperationResult{}, wrappedErr
	}

	// Log success
	logger.Info(operationName+" completed successfully",
		attr.String("operation", operationName),
		attr.RoundID("round_id", roundID),
		attr.LogAttr(correlationID),
	)
	metrics.RecordOperationSuccess(operationName, roundID)

	return result, nil
}

// ScoreOperationResult represents a generic result from a score operation
type ScoreOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
