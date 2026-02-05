package userservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// -----------------------------------------------------------------------------
// Service Struct & Constructor
// -----------------------------------------------------------------------------

// UserService handles user-related logic.
type UserService struct {
	repo    userdb.Repository
	logger  *slog.Logger
	metrics usermetrics.UserMetrics
	tracer  trace.Tracer
	db      *bun.DB
}

// NewUserService creates a new UserService.
func NewUserService(
	repo userdb.Repository,
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
	db *bun.DB,
) *UserService {
	return &UserService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		db:      db,
	}
}

// -----------------------------------------------------------------------------
// Generic Helpers (functions because methods cannot have type params)
// -----------------------------------------------------------------------------

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func withTelemetry[S any, F any](
	s *UserService,
	ctx context.Context,
	operationName string,
	userID sharedtypes.DiscordID,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {
	if ctx == nil {
		return results.FailureResult[S, F](any(errors.New("context cannot be nil")).(F)), errors.New("context cannot be nil")
	}

	// Start span
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("user_id", string(userID)),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	// Record attempt
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, userID)
	}

	// Track duration
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, time.Since(startTime), userID)
		}
	}()

	// Log operation start
	s.logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("user_id", string(userID)),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, userID)
			}
			span.RecordError(err)
			result = results.OperationResult[S, F]{}
		}
	}()

	// Execute operation
	result, err = op(ctx)

	// Handle Infrastructure Error
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, userID)
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Handle Domain Failure
	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
			attr.Any("failure_payload", *result.Failure),
		)
	}

	// Handle Success
	if result.IsSuccess() {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("user_id", string(userID)),
		)
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, userID)
	}

	return result, nil
}

// runInTx ensures the operation runs within a database transaction.
func runInTx[S any, F any](
	s *UserService,
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
