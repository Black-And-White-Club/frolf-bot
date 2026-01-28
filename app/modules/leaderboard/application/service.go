package leaderboardservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardService implements the Service interface.
type LeaderboardService struct {
	repo    leaderboarddb.Repository
	logger  *slog.Logger
	metrics leaderboardmetrics.LeaderboardMetrics
	tracer  trace.Tracer
	db      *bun.DB
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(
	db *bun.DB,
	repo leaderboarddb.Repository,
	logger *slog.Logger,
	metrics leaderboardmetrics.LeaderboardMetrics,
	tracer trace.Tracer,
) *LeaderboardService {

	if logger == nil {
		logger = slog.Default()
	}
	return &LeaderboardService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		db:      db,
	}
}

// EnsureGuildLeaderboard creates an empty active leaderboard for the guild if none exists.
// This is an infrastructure setup method.
func (s *LeaderboardService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {

	// Named transaction function
	ensureTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[bool, error], error) {
		return s.ensureGuildLeaderboardLogic(ctx, db, guildID)
	}

	return withTelemetry(s, ctx, "EnsureGuildLeaderboard", guildID, func(ctx context.Context) (results.OperationResult[bool, error], error) {
		return runInTx(s, ctx, ensureTx)
	})
}

// ensureGuildLeaderboardLogic contains the core logic.
func (s *LeaderboardService) ensureGuildLeaderboardLogic(ctx context.Context, db bun.IDB, guildID sharedtypes.GuildID) (results.OperationResult[bool, error], error) {
	_, err := s.repo.GetActiveLeaderboard(ctx, db, guildID)
	if err == nil {
		// Already exists
		return results.SuccessResult[bool, error](false), nil
	}
	if !errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
		return results.OperationResult[bool, error]{}, err
	}

	s.logger.InfoContext(ctx, "Ensuring active leaderboard for guild", attr.String("guild_id", string(guildID)))

	empty := &leaderboardtypes.Leaderboard{
		LeaderboardData: leaderboardtypes.LeaderboardData{},
		IsActive:        true,
		UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
		GuildID:         guildID,
	}

	if err := s.repo.SaveLeaderboard(ctx, db, empty); err != nil {
		return results.OperationResult[bool, error]{}, fmt.Errorf("failed to create empty leaderboard for guild %s: %w", guildID, err)
	}
	return results.SuccessResult[bool, error](true), nil
}

// -----------------------------------------------------------------------------
// Generic Helpers (Defined as functions because methods cannot have type params)
// -----------------------------------------------------------------------------

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
// It is generic [S, F] to handle any return type safely.
func withTelemetry[S any, F any](
	s *LeaderboardService,
	ctx context.Context,
	operationName string,
	guildID sharedtypes.GuildID,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {

	// Start span
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("guild_id", string(guildID)),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	// Record attempt
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, "LeaderboardService")
	}

	// Track duration
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, "LeaderboardService", time.Since(startTime))
		}
	}()

	// Log operation start
	s.logger.InfoContext(ctx, "Operation triggered", attr.ExtractCorrelationID(ctx), attr.String("operation", operationName))

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
			}
			span.RecordError(err)
			// Return zero value for result
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
			attr.String("guild_id", string(guildID)),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Handle Domain Failure (Validation, etc)
	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("guild_id", string(guildID)),
			// We can dereference failure safely because IsFailure checked it
			attr.Any("failure_payload", *result.Failure),
		)
		// Domain failures are NOT system failures, so we don't increment Failure metric
		// or span.RecordError. They are successful "decisions" to reject.
	}

	// Handle Success
	if result.IsSuccess() {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("guild_id", string(guildID)),
		)
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, "LeaderboardService")
	}

	return result, nil
}

// runInTx ensures the operation runs within a transaction.
func runInTx[S any, F any](
	s *LeaderboardService,
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
