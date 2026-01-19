package roundservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RoundService implements the Service interface.
type RoundService struct {
	repo                rounddb.Repository
	queueService        roundqueue.QueueService
	eventBus            eventbus.EventBus
	userLookup          UserLookup
	metrics             roundmetrics.RoundMetrics
	logger              *slog.Logger
	tracer              trace.Tracer
	roundValidator      roundutil.RoundValidator
	guildConfigProvider GuildConfigProvider
}

// GuildConfigProvider supplies guild config for enrichment (DB-backed, no events)
type GuildConfigProvider interface {
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
}

// NewRoundService creates a new RoundService.
func NewRoundService(
	repo rounddb.Repository,
	queueService roundqueue.QueueService,
	eventBus eventbus.EventBus,
	userLookup UserLookup,
	metrics roundmetrics.RoundMetrics,
	logger *slog.Logger,
	tracer trace.Tracer,
	roundValidator roundutil.RoundValidator,
) *RoundService {
	return &RoundService{
		repo:           repo,
		queueService:   queueService,
		eventBus:       eventBus,
		userLookup:     userLookup,
		metrics:        metrics,
		logger:         logger,
		tracer:         tracer,
		roundValidator: roundValidator,
	}
}

// operationFunc is the signature for service operation functions.
type operationFunc func(ctx context.Context) (results.OperationResult, error)

// withTelemetry wraps a service operation with tracing, metrics, and panic recovery.
func (s *RoundService) withTelemetry(
	ctx context.Context,
	operationName string,
	roundID sharedtypes.RoundID,
	op operationFunc,
) (result results.OperationResult, err error) {

	ctx, span := s.tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	s.metrics.RecordOperationAttempt(ctx, operationName, "RoundService")

	startTime := time.Now()
	defer func() {
		s.metrics.RecordOperationDuration(ctx, operationName, "RoundService", time.Since(startTime))
	}()

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in %s: %v", operationName, r)
			s.logger.ErrorContext(ctx, "Critical panic recovered",
				attr.ExtractCorrelationID(ctx),
				attr.String("round_id", roundID.String()),
				attr.Error(err),
			)
			s.metrics.RecordOperationFailure(ctx, operationName, "RoundService")
			span.RecordError(err)
			result = results.OperationResult{}
		}
	}()

	result, err = op(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s: %w", operationName, err)
		s.logger.ErrorContext(ctx, "Operation failed with error",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("round_id", roundID.String()),
			attr.Error(wrappedErr),
			attr.Any("result_has_failure", result.Failure != nil),
		)
		s.metrics.RecordOperationFailure(ctx, operationName, "RoundService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	// Check for business logic failures even when err is nil
	if result.Failure != nil {
		s.logger.WarnContext(ctx, "Operation returned failure result",
			attr.ExtractCorrelationID(ctx),
			attr.String("operation", operationName),
			attr.String("round_id", roundID.String()),
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
			attr.String("round_id", roundID.String()),
			attr.Any("success_type", fmt.Sprintf("%T", result.Success)),
		)
	}

	s.metrics.RecordOperationSuccess(ctx, operationName, "RoundService")
	return result, nil
}

// WithGuildConfigProvider injects a provider (fluent style)
func (s *RoundService) WithGuildConfigProvider(p GuildConfigProvider) *RoundService {
	s.guildConfigProvider = p
	return s
}

// getGuildConfigForEnrichment attempts to retrieve a guild config for adding config fragments
// to outbound round events. This is a placeholder; wire in actual retrieval (e.g., injected
// GuildConfigProvider) later. Returning nil is safe: enrichment is optional.
func (s *RoundService) getGuildConfigForEnrichment(ctx context.Context, guildID sharedtypes.GuildID) *guildtypes.GuildConfig {
	if s.guildConfigProvider == nil || guildID == "" {
		return nil
	}
	cfg, err := s.guildConfigProvider.GetConfig(ctx, guildID)
	if err != nil {
		s.logger.DebugContext(ctx, "Guild config enrichment fetch failed",
			attr.String("guild_id", string(guildID)),
			attr.Error(err),
		)
		return nil
	}
	return cfg
}
