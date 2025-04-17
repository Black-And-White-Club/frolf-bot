package roundservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RoundService handles round-related logic.
type RoundService struct {
	RoundDB        rounddb.RoundDB
	logger         *slog.Logger
	metrics        roundmetrics.RoundMetrics
	tracer         trace.Tracer
	roundValidator roundutil.RoundValidator
	EventBus       eventbus.EventBus
	serviceWrapper func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error)
}

// NewRoundService creates a new RoundService.
func NewRoundService(
	db rounddb.RoundDB,
	logger *slog.Logger,
	metrics roundmetrics.RoundMetrics,
	tracer trace.Tracer,
) Service {
	return &RoundService{
		RoundDB:        db,
		logger:         logger,
		metrics:        metrics,
		tracer:         tracer,
		roundValidator: roundutil.NewRoundValidator(),
		// Assign the serviceWrapper method
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (result RoundOperationResult, err error) {
			return serviceWrapper(ctx, operationName, roundID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error), logger *slog.Logger, metrics roundmetrics.RoundMetrics, tracer trace.Tracer) (result RoundOperationResult, err error) {
	if serviceFunc == nil {
		return RoundOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, "RoundService")

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, "RoundService", duration)
	}()

	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, "RoundService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = RoundOperationResult{}
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
		metrics.RecordOperationFailure(ctx, operationName, "RoundService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(ctx, operationName, "RoundService")

	return result, nil
}

// RoundOperationResult represents a generic result from a round operation
type RoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
