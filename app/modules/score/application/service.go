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
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ScoreService implements the Service interface.
type ScoreService struct {
	repo     scoredb.Repository
	EventBus eventbus.EventBus
	logger   *slog.Logger
	metrics  scoremetrics.ScoreMetrics
	tracer   trace.Tracer
	// Backwards compatible serviceWrapper for tests and callers that override behavior.
	serviceWrapper func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error)
}

// NewScoreService creates a new ScoreService.
func NewScoreService(
	repo scoredb.Repository,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer trace.Tracer,
) Service {
	svc := &ScoreService{
		repo:     repo,
		EventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
	}

	// Default legacy-compatible serviceWrapper that delegates to withTelemetry.
	svc.serviceWrapper = func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error)) (ScoreOperationResult, error) {
		// Adapter: convert legacy ScoreOperationResult <-> results.OperationResult
		adaptedOp := func(ctx context.Context) (results.OperationResult, error) {
			res, err := serviceFunc(ctx)
			if err != nil {
				return results.OperationResult{}, err
			}
			return results.OperationResult{Success: res.Success, Failure: res.Failure}, nil
		}

		opRes, err := svc.withTelemetry(ctx, operationName, roundID, adaptedOp)
		if err != nil {
			return ScoreOperationResult{Error: err}, err
		}

		return ScoreOperationResult{Success: opRes.Success, Failure: opRes.Failure}, nil
	}

	return svc
}

// operationFunc is the signature for service operation functions.
type operationFunc func(ctx context.Context) (results.OperationResult, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
// This standardizes observability across all service methods.
func (s *ScoreService) withTelemetry(
	ctx context.Context,
	operationName string,
	roundID sharedtypes.RoundID,
	op operationFunc,
) (result results.OperationResult, err error) {
	if op == nil {
		return results.OperationResult{}, errors.New("operation function is nil")
	}

	// Start a new tracing span while preserving existing context
	ctx, span := s.tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	// Record the operation attempt
	s.metrics.RecordOperationAttempt(ctx, operationName, roundID)

	startTime := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration(ctx, operationName, time.Since(startTime))
	}()

	// Log the operation start
	s.logger.InfoContext(ctx, operationName+" triggered",
		attr.String("operation", operationName),
		attr.RoundID("round_id", roundID),
		attr.ExtractCorrelationID(ctx),
	)

	// Recover from panic and log the error
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, errorMsg,
				attr.RoundID("round_id", roundID),
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r),
			)
			s.metrics.RecordOperationFailure(ctx, operationName, roundID)

			// Create error with panic information
			err = fmt.Errorf("panic in %s: %v", operationName, r)

			// Record error in span
			span.RecordError(err)
			result = results.OperationResult{}
		}
	}()

	// Call the service function
	result, err = op(ctx)

	// If the service function returned an error AND did NOT populate a Failure payload,
	// then it's a true system error that should be propagated.
	if err != nil && result.Failure == nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Error in "+operationName,
			attr.RoundID("round_id", roundID),
			attr.ExtractCorrelationID(ctx),
			attr.Error(wrappedErr),
		)
		s.metrics.RecordOperationFailure(ctx, operationName, roundID)
		span.RecordError(wrappedErr)
		return results.OperationResult{}, wrappedErr // Return empty result and wrapped error
	}

	// If there was an error but a Failure payload was populated, or no error occurred,
	// log success and return the result.
	if err == nil || result.Failure != nil { // This condition explicitly handles both success and handled business failures
		s.logger.InfoContext(ctx, operationName+" completed successfully",
			attr.String("operation", operationName),
			attr.RoundID("round_id", roundID),
			attr.ExtractCorrelationID(ctx),
		)
		s.metrics.RecordOperationSuccess(ctx, operationName, roundID)
		return result, nil // Return the actual result and nil error
	}

	// Fallback for unexpected scenarios (should ideally not be reached)
	return results.OperationResult{}, fmt.Errorf("unexpected state after %s operation", operationName)
}

// ScoreOperationResult represents a generic result from a score operation
type ScoreOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}

// GetScoresForRound returns the stored round scores (with tag numbers) from persistence.
func (s *ScoreService) GetScoresForRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	return s.repo.GetScoresForRound(ctx, guildID, roundID)
}

// serviceWrapper is a legacy package-level adapter kept for compatibility with existing callers/tests.
// It delegates to a temporary ScoreService.withTelemetry instance.
func serviceWrapper(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (ScoreOperationResult, error), logger *slog.Logger, metrics scoremetrics.ScoreMetrics, tracer trace.Tracer) (ScoreOperationResult, error) {
	// Create a lightweight service instance to reuse withTelemetry logic
	svc := &ScoreService{logger: logger, metrics: metrics, tracer: tracer}

	adaptedOp := func(ctx context.Context) (results.OperationResult, error) {
		res, err := serviceFunc(ctx)
		if err != nil {
			return results.OperationResult{}, err
		}
		return results.OperationResult{Success: res.Success, Failure: res.Failure}, nil
	}

	opRes, err := svc.withTelemetry(ctx, operationName, roundID, adaptedOp)
	if err != nil {
		return ScoreOperationResult{Error: err}, err
	}
	return ScoreOperationResult{Success: opRes.Success, Failure: opRes.Failure}, nil
}
