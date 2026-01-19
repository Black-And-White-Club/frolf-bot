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
	db      *bun.DB // Keep for runInTx helper (justified deviation)
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

// operationFunc is the signature for service operation functions.
type operationFunc func(ctx context.Context) (results.OperationResult, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
// This standardizes observability across all service methods.
func (s *LeaderboardService) withTelemetry(
	ctx context.Context,
	operationName string,
	guildID sharedtypes.GuildID,
	op operationFunc,
) (result results.OperationResult, err error) {
	// Start span (nil-safe tracer)
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("guild_id", string(guildID)),
		))
	} else {
		// Use a no-op span from context when tracer is not provided (tests may pass nil)
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
			s.logger.ErrorContext(ctx, "Critical panic recovered", attr.ExtractCorrelationID(ctx), attr.String("guild_id", string(guildID)), attr.Error(err))
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
			}
			span.RecordError(err)
			result = results.OperationResult{}
		}
	}()

	// Execute operation
	result, err = op(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("guild_id", string(guildID)),
			attr.Error(wrappedErr),
			attr.Any("result_has_failure", result.Failure != nil),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, "LeaderboardService")
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Check for business logic failures even when err is nil
	if result.Failure != nil {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("guild_id", string(guildID)),
			attr.Any("failure_payload", result.Failure),
			attr.Any("failure_type", fmt.Sprintf("%T", result.Failure)),
		)
		// Note: Not recording as operation failure in metrics since err is nil
		// and the operation technically succeeded (business validation failed)
	}

	// Log successful operations at debug level with result type
	if result.Success != nil {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("guild_id", string(guildID)),
			attr.Any("success_type", fmt.Sprintf("%T", result.Success)),
		)
	}

	s.logger.InfoContext(ctx, operationName+" completed successfully", attr.ExtractCorrelationID(ctx), attr.String("operation", operationName))
	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, "LeaderboardService")
	}

	return result, nil
}

// runInTx is a helper to ensure service operations are atomic.
// Justified deviation: Leaderboard requires multi-operation transactions.
func (s *LeaderboardService) runInTx(ctx context.Context, fn func(ctx context.Context, db bun.IDB) (results.OperationResult, error)) (results.OperationResult, error) {
	var result results.OperationResult
	if s.db == nil {
		var err error
		result, err = fn(ctx, nil)
		return result, err
	}

	err := s.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		var err error
		result, err = fn(ctx, tx)
		return err
	})
	return result, err
}

// EnsureGuildLeaderboard creates an empty active leaderboard for the guild if none exists.
func (s *LeaderboardService) EnsureGuildLeaderboard(ctx context.Context, guildID sharedtypes.GuildID) error {
	_, err := s.repo.GetActiveLeaderboard(ctx, guildID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, leaderboarddb.ErrNoActiveLeaderboard) {
		return err
	}

	s.logger.InfoContext(ctx, "Ensuring active leaderboard for guild", attr.String("guild_id", string(guildID)))

	empty := &leaderboarddb.Leaderboard{
		LeaderboardData: leaderboardtypes.LeaderboardData{},
		IsActive:        true,
		UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
		GuildID:         guildID,
	}

	if _, err := s.repo.CreateLeaderboard(ctx, s.db, guildID, empty); err != nil {
		return fmt.Errorf("failed to create empty leaderboard for guild %s: %w", guildID, err)
	}
	return nil
}
