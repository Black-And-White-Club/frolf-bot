package clubservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	clubmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/club"
	clubtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/club"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	clubdb "github.com/Black-And-White-Club/frolf-bot/app/modules/club/infrastructure/repositories"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ClubService implements the Service interface.
type ClubService struct {
	repo    clubdb.Repository
	logger  *slog.Logger
	metrics clubmetrics.ClubMetrics
	tracer  trace.Tracer
	db      *bun.DB
}

// NewClubService creates a new ClubService.
func NewClubService(
	repo clubdb.Repository,
	logger *slog.Logger,
	metrics clubmetrics.ClubMetrics,
	tracer trace.Tracer,
	db *bun.DB,
) *ClubService {
	if logger == nil {
		logger = slog.Default()
	}
	return &ClubService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
		db:      db,
	}
}

// GetClub retrieves club info by UUID.
func (s *ClubService) GetClub(ctx context.Context, clubUUID uuid.UUID) (*clubtypes.ClubInfo, error) {
	getClubTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return s.getClubLogic(ctx, db, clubUUID)
	}

	result, err := withTelemetry(s, ctx, "GetClub", clubUUID.String(), func(ctx context.Context) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return runInTx(s, ctx, getClubTx)
	})
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}
	return *result.Success, nil
}

// getClubLogic contains the core logic.
func (s *ClubService) getClubLogic(ctx context.Context, db bun.IDB, clubUUID uuid.UUID) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
	club, err := s.repo.GetByUUID(ctx, db, clubUUID)
	if err != nil {
		if errors.Is(err, clubdb.ErrNotFound) {
			return results.FailureResult[*clubtypes.ClubInfo, error](err), nil
		}
		return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to get club: %w", err)
	}

	return results.SuccessResult[*clubtypes.ClubInfo, error](&clubtypes.ClubInfo{
		UUID:    club.UUID.String(),
		Name:    club.Name,
		IconURL: club.IconURL,
	}), nil
}

// UpsertClubFromDiscord creates or updates a club from Discord guild info.
func (s *ClubService) UpsertClubFromDiscord(ctx context.Context, guildID, name string, iconURL *string) (*clubtypes.ClubInfo, error) {
	upsertTx := func(ctx context.Context, db bun.IDB) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return s.upsertClubFromDiscordLogic(ctx, db, guildID, name, iconURL)
	}

	result, err := withTelemetry(s, ctx, "UpsertClubFromDiscord", guildID, func(ctx context.Context) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
		return runInTx(s, ctx, upsertTx)
	})
	if err != nil {
		return nil, err
	}
	if result.IsFailure() {
		return nil, *result.Failure
	}
	return *result.Success, nil
}

// upsertClubFromDiscordLogic contains the core logic.
func (s *ClubService) upsertClubFromDiscordLogic(ctx context.Context, db bun.IDB, guildID, name string, iconURL *string) (results.OperationResult[*clubtypes.ClubInfo, error], error) {
	existing, err := s.repo.GetByDiscordGuildID(ctx, db, guildID)
	if err != nil && !errors.Is(err, clubdb.ErrNotFound) {
		return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to check existing club: %w", err)
	}

	var club *clubdb.Club
	if existing != nil {
		existing.Name = name
		existing.IconURL = iconURL
		if err := s.repo.Upsert(ctx, db, existing); err != nil {
			return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to update club: %w", err)
		}
		club = existing
	} else {
		club = &clubdb.Club{
			UUID:           uuid.New(),
			Name:           name,
			IconURL:        iconURL,
			DiscordGuildID: &guildID,
		}
		if err := s.repo.Upsert(ctx, db, club); err != nil {
			// Retry once in case of race condition on discord_guild_id
			existing, retryErr := s.repo.GetByDiscordGuildID(ctx, db, guildID)
			if retryErr == nil && existing != nil {
				existing.Name = name
				existing.IconURL = iconURL
				if updateErr := s.repo.Upsert(ctx, db, existing); updateErr != nil {
					return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to update club on retry: %w", updateErr)
				}
				club = existing
			} else {
				return results.OperationResult[*clubtypes.ClubInfo, error]{}, fmt.Errorf("failed to create club: %w", err)
			}
		}
	}

	return results.SuccessResult[*clubtypes.ClubInfo, error](&clubtypes.ClubInfo{
		UUID:    club.UUID.String(),
		Name:    club.Name,
		IconURL: club.IconURL,
	}), nil
}

// -----------------------------------------------------------------------------
// Generic Helpers (Defined as functions because methods cannot have type params)
// -----------------------------------------------------------------------------

// operationFunc is the generic signature for service operation functions.
type operationFunc[S any, F any] func(ctx context.Context) (results.OperationResult[S, F], error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func withTelemetry[S any, F any](
	s *ClubService,
	ctx context.Context,
	operationName string,
	identifier string,
	op operationFunc[S, F],
) (result results.OperationResult[S, F], err error) {

	// Start span
	var span trace.Span
	if s.tracer != nil {
		ctx, span = s.tracer.Start(ctx, operationName, trace.WithAttributes(
			attribute.String("operation", operationName),
			attribute.String("identifier", identifier),
		))
	} else {
		span = trace.SpanFromContext(ctx)
	}
	defer span.End()

	// Record attempt
	if s.metrics != nil {
		s.metrics.RecordOperationAttempt(ctx, operationName, "ClubService")
	}

	// Track duration
	startTime := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.RecordOperationDuration(ctx, operationName, "ClubService", time.Since(startTime))
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
				attr.String("identifier", identifier),
				attr.Error(err),
			)
			if s.metrics != nil {
				s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
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
			attr.String("identifier", identifier),
			attr.Error(wrappedErr),
		)
		if s.metrics != nil {
			s.metrics.RecordOperationFailure(ctx, operationName, "ClubService")
		}
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Handle Domain Failure
	if result.IsFailure() {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
			attr.Any("failure_payload", *result.Failure),
		)
	}

	// Handle Success
	if result.IsSuccess() {
		s.logger.InfoContext(ctx, "Operation completed successfully",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("identifier", identifier),
		)
	}

	if s.metrics != nil {
		s.metrics.RecordOperationSuccess(ctx, operationName, "ClubService")
	}

	return result, nil
}

// runInTx ensures the operation runs within a transaction.
func runInTx[S any, F any](
	s *ClubService,
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
