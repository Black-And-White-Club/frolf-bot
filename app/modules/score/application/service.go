package scoreservice

import (
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
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreService handles score processing logic.
type ScoreService struct {
	ScoreDB        scoredb.ScoreDB
	EventBus       eventbus.EventBus
	logger         lokifrolfbot.Logger
	metrics        scoremetrics.ScoreMetrics
	tracer         tempofrolfbot.Tracer
	serviceWrapper func(msg *message.Message, operationName string, roundID sharedtypes.RoundID, serviceFunc func() (ScoreOperationResult, error)) (ScoreOperationResult, error)
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
		// Assign the serviceWrapper method
		serviceWrapper: func(msg *message.Message, operationName string, roundID sharedtypes.RoundID, serviceFunc func() (ScoreOperationResult, error)) (ScoreOperationResult, error) {
			return serviceWrapper(msg, operationName, roundID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(msg *message.Message, operationName string, roundID sharedtypes.RoundID, serviceFunc func() (ScoreOperationResult, error), logger lokifrolfbot.Logger, metrics scoremetrics.ScoreMetrics, tracer tempofrolfbot.Tracer) (result ScoreOperationResult, err error) {
	ctx, span := tracer.StartSpan(msg.Context(), operationName, msg)
	defer span.End()

	msg = msg.Copy()
	msg.SetContext(ctx)

	metrics.RecordOperationAttempt(operationName, roundID)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordOperationDuration(operationName, duration)
	}()

	logger.Info(operationName+" triggered",
		attr.CorrelationIDFromMsg(msg),
		attr.String("message_id", msg.UUID),
		attr.String("operation", operationName),
		attr.Int64("round_id", int64(roundID)),
	)

	// Modify `err` directly inside the defer so it propagates correctly
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.Error(errorMsg,
				attr.CorrelationIDFromMsg(msg),
				attr.Int64("round_id", int64(roundID)),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(operationName, roundID)
			span.RecordError(errors.New(errorMsg))

			// Since result and err are named return values, modifying them here affects the function return
			result = ScoreOperationResult{}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	// Now, if `serviceFunc()` panics, `defer` will catch it and modify `err`
	result, err = serviceFunc()
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)

		logger.Error("Error in "+operationName,
			attr.CorrelationIDFromMsg(msg),
			attr.Int64("round_id", int64(roundID)),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(operationName, roundID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.Info(operationName+" completed successfully",
		attr.CorrelationIDFromMsg(msg),
		attr.Int64("round_id", int64(roundID)),
		attr.String("operation", operationName),
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
