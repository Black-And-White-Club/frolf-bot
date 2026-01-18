package guildservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GuildService implements the Service interface.
type GuildService struct {
	repo    guilddb.Repository
	logger  *slog.Logger
	metrics guildmetrics.GuildMetrics
	tracer  trace.Tracer
}

// NewGuildService creates a new GuildService.
func NewGuildService(
	repo guilddb.Repository,
	logger *slog.Logger,
	metrics guildmetrics.GuildMetrics,
	tracer trace.Tracer,
) *GuildService {
	return &GuildService{
		repo:    repo,
		logger:  logger,
		metrics: metrics,
		tracer:  tracer,
	}
}

// operationFunc is the signature for service operation functions.
type operationFunc func(ctx context.Context) (results.OperationResult, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
// This standardizes observability across all service methods.
func (s *GuildService) withTelemetry(
	ctx context.Context,
	operationName string,
	guildID sharedtypes.GuildID,
	op operationFunc,
) (result results.OperationResult, err error) {
	// Start span
	ctx, span := s.tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("guild_id", string(guildID)),
	))
	defer span.End()

	// Record attempt
	s.metrics.RecordOperationAttempt(ctx, operationName, guildID, "GuildService")

	// Track duration
	startTime := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration(ctx, operationName, guildID, "GuildService", time.Since(startTime))
	}()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(guildID)),
				attr.Error(err),
			)
			s.metrics.RecordOperationFailure(ctx, operationName, guildID, "GuildService")
			span.RecordError(err)
			result = results.OperationResult{}
		}
	}()

	// Execute operation
	result, err = op(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed",
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(guildID)),
			attr.Error(wrappedErr),
		)
		s.metrics.RecordOperationFailure(ctx, operationName, guildID, "GuildService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	s.metrics.RecordOperationSuccess(ctx, operationName, guildID, "GuildService")
	return result, nil
}
