package roundservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB        rounddb.RoundDB
	logger         lokifrolfbot.Logger
	metrics        roundmetrics.RoundMetrics
	tracer         tempofrolfbot.Tracer
	roundValidator roundutil.RoundValidator
	EventBus       eventbus.EventBus
	serviceWrapper func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (RoundOperationResult, error)
}

// NewRoundService creates a new RoundService.
func NewRoundService(
	db rounddb.RoundDB,
	logger lokifrolfbot.Logger,
	metrics roundmetrics.RoundMetrics,
	tracer tempofrolfbot.Tracer,
) Service {
	return &RoundService{
		RoundDB:        db,
		logger:         logger,
		metrics:        metrics,
		tracer:         tracer,
		roundValidator: roundutil.NewRoundValidator(),
		// Assign the serviceWrapper method
		serviceWrapper: func(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error)) (result RoundOperationResult, err error) {
			return serviceWrapper(ctx, operationName, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(ctx context.Context, operationName string, serviceFunc func() (RoundOperationResult, error), logger lokifrolfbot.Logger, metrics roundmetrics.RoundMetrics, tracer tempofrolfbot.Tracer) (result RoundOperationResult, err error) {
	if serviceFunc == nil {
		return RoundOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.StartSpan(ctx, operationName, nil)
	defer span.End()

	metrics.RecordOperationAttempt(operationName, "RoundService")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordOperationDuration(operationName, "RoundService", duration)
	}()

	correlationID := attr.ExtractCorrelationID(ctx)
	logger.InfoContext(ctx, "Operation triggered",
		attr.LogAttr(correlationID),
		attr.String("operation", operationName),
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.LogAttr(correlationID),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(operationName, "RoundService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = RoundOperationResult{}
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
		metrics.RecordOperationFailure(operationName, "RoundService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.LogAttr(correlationID),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(operationName, "RoundService")

	return result, nil
}

// RoundOperationResult represents a generic result from a round operation
type RoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
