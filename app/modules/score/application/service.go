package scoreservice

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	scoredb "github.com/Black-And-White-Club/frolf-bot/app/modules/score/infrastructure/repositories"
	"github.com/uptrace/bun"
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
	db       *bun.DB
}

// NewScoreService creates a new ScoreService.
func NewScoreService(
	repo scoredb.Repository,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics scoremetrics.ScoreMetrics,
	tracer trace.Tracer,
	db *bun.DB,
) *ScoreService {
	return &ScoreService{
		repo:     repo,
		EventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		db:       db,
	}
}

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func withTelemetry[S any, F any](
	s *ScoreService,
	ctx context.Context,
	operationName string,
	roundID sharedtypes.RoundID,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {

	ctx, span := s.tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	s.metrics.RecordOperationAttempt(ctx, operationName, roundID)

	startTime := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration(ctx, operationName, time.Since(startTime))
	}()

	s.logger.InfoContext(ctx, operationName+" triggered",
		attr.String("operation", operationName),
		attr.RoundID("round_id", roundID),
		attr.ExtractCorrelationID(ctx),
	)

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.RoundID("round_id", roundID),
				attr.ExtractCorrelationID(ctx),
				attr.Error(err),
			)
			s.metrics.RecordOperationFailure(ctx, operationName, roundID)
			span.RecordError(err)
			result = results.OperationResult[S, F]{}
		}
	}()

	result, err = op(ctx)

	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.RoundID("round_id", roundID),
			attr.Error(wrappedErr),
		)
		s.metrics.RecordOperationFailure(ctx, operationName, roundID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.RoundID("round_id", roundID),
			attr.Any("failure_payload", *result.Failure),
		)
	}

	if result.IsSuccess() {
		s.logger.InfoContext(ctx, operationName+" completed successfully",
			attr.String("operation", operationName),
			attr.RoundID("round_id", roundID),
			attr.ExtractCorrelationID(ctx),
		)
		s.metrics.RecordOperationSuccess(ctx, operationName, roundID)
	}

	return result, nil
}

// runInTx ensures the operation runs within a transaction.
func runInTx[S any, F any](
	s *ScoreService,
	ctx context.Context,
	fn func(ctx context.Context, db bun.IDB) (results.OperationResult[S, F], error),
) (results.OperationResult[S, F], error) {
	if s.db == nil {
		return fn(ctx, nil)
	}

	var result results.OperationResult[S, F]
	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var txErr error
		result, txErr = fn(ctx, tx)
		return txErr
	})

	return result, err
}

// --- Service Methods ---

// GetScoresForRound retrieves all scores for a round.
func (s *ScoreService) GetScoresForRound(ctx context.Context, guildID sharedtypes.GuildID, roundID sharedtypes.RoundID) ([]sharedtypes.ScoreInfo, error) {
	// Simple read methods often don't need the full telemetry/tx wrapper unless complex,
	// but you can wrap it if you want metrics for every call.
	return s.repo.GetScoresForRound(ctx, nil, guildID, roundID)
}
