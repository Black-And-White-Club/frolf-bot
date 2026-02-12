package leaderboardservice

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	leaderboarddomain "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/domain"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CommandPipeline defines the normalized command flow contract used by the service boundary.
type CommandPipeline interface {
	ProcessRound(ctx context.Context, cmd ProcessRoundCommand) (*ProcessRoundOutput, error)
	ApplyTagAssignments(
		ctx context.Context,
		guildID string,
		requests []sharedtypes.TagAssignmentRequest,
		source sharedtypes.ServiceUpdateSource,
		updateID sharedtypes.RoundID,
	) (leaderboardtypes.LeaderboardData, error)
	StartSeason(ctx context.Context, guildID, seasonID, seasonName string) error
	EndSeason(ctx context.Context, guildID string) error
	ResetTags(ctx context.Context, guildID string, finishOrder []string) ([]leaderboarddomain.TagChange, error)
	GetTaggedMembers(ctx context.Context, guildID string) ([]TaggedMemberView, error)
	GetMemberTag(ctx context.Context, guildID, memberID string) (int, bool, error)
	CheckTagAvailability(ctx context.Context, guildID, memberID string, tagNumber int) (bool, string, error)

	// Tag History
	GetTagHistory(ctx context.Context, guildID, memberID string, limit int) ([]TagHistoryView, error)

	// Chart Generation
	GenerateTagGraphPNG(ctx context.Context, guildID, memberID string) ([]byte, error)
}

// TagHistoryView is a read model for tag history entries returned by the service.
type TagHistoryView struct {
	ID          int64
	TagNumber   int
	OldMemberID string
	NewMemberID string
	Reason      string
	RoundID     *string
	CreatedAt   time.Time
}

// toTagHistoryView converts a repository model to a service view model.
func toTagHistoryView(entry leaderboarddb.TagHistoryEntry) TagHistoryView {
	view := TagHistoryView{
		ID:          entry.ID,
		TagNumber:   entry.TagNumber,
		NewMemberID: entry.NewMemberID,
		Reason:      entry.Reason,
		CreatedAt:   entry.CreatedAt,
	}
	if entry.OldMemberID != nil {
		view.OldMemberID = *entry.OldMemberID
	}
	if entry.RoundID != nil {
		s := entry.RoundID.String()
		view.RoundID = &s
	}
	return view
}

// LeaderboardService implements the Service interface.
type LeaderboardService struct {
	repo            leaderboarddb.Repository
	memberRepo      leaderboarddb.LeagueMemberRepository
	tagHistRepo     leaderboarddb.TagHistoryRepository
	outcomeRepo     leaderboarddb.RoundOutcomeRepository
	logger          *slog.Logger
	metrics         leaderboardmetrics.LeaderboardMetrics
	tracer          trace.Tracer
	db              *bun.DB
	commandPipeline CommandPipeline
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
	service := &LeaderboardService{
		repo:        repo,
		memberRepo:  leaderboarddb.NewLeagueMemberRepo(),
		tagHistRepo: leaderboarddb.NewTagHistoryRepo(),
		outcomeRepo: leaderboarddb.NewRoundOutcomeRepo(),
		logger:      logger,
		metrics:     metrics,
		tracer:      tracer,
		db:          db,
	}
	service.commandPipeline = &serviceCommandPipeline{service: service}
	return service
}

// SetCommandPipeline overrides the default command flow.
// Intended for tests and specialized wiring only.
func (s *LeaderboardService) SetCommandPipeline(handler CommandPipeline) {
	s.commandPipeline = handler
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
	s.logger.InfoContext(ctx, "Guild leaderboard initialization is a no-op in normalized mode", attr.String("guild_id", string(guildID)))
	return results.SuccessResult[bool, error](false), nil
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
