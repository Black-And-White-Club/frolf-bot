package roundservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	roundqueue "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/queue"
	rounddb "github.com/Black-And-White-Club/frolf-bot/app/modules/round/infrastructure/repositories"
	roundutil "github.com/Black-And-White-Club/frolf-bot/app/modules/round/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// RoundService uses the concrete queue service directly
type RoundService struct {
	RoundDB             rounddb.RoundDB
	QueueService        roundqueue.QueueService // Use the interface from infrastructure
	EventBus            eventbus.EventBus
	userLookup          UserLookup
	metrics             roundmetrics.RoundMetrics
	logger              *slog.Logger
	tracer              trace.Tracer
	roundValidator      roundutil.RoundValidator
	serviceWrapper      func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (RoundOperationResult, error)
	guildConfigProvider GuildConfigProvider // optional provider for enrichment
}

// GuildConfigProvider supplies guild config for enrichment (DB-backed, no events)
type GuildConfigProvider interface {
	GetConfig(ctx context.Context, guildID sharedtypes.GuildID) (*guildtypes.GuildConfig, error)
}

// Constructor takes the concrete implementation
func NewRoundService(
	roundDB rounddb.RoundDB,
	queueService roundqueue.QueueService, // Interface from infrastructure
	eventBus eventbus.EventBus,
	userLookup UserLookup,
	metrics roundmetrics.RoundMetrics,
	logger *slog.Logger,
	tracer trace.Tracer,
	roundValidator roundutil.RoundValidator,
) *RoundService {
	return &RoundService{
		RoundDB:        roundDB,
		QueueService:   queueService,
		EventBus:       eventBus,
		userLookup:     userLookup,
		metrics:        metrics,
		logger:         logger,
		tracer:         tracer,
		roundValidator: roundValidator,
		serviceWrapper: func(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error)) (result RoundOperationResult, err error) {
			return serviceWrapper(ctx, operationName, roundID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// WithGuildConfigProvider injects a provider (fluent style)
func (s *RoundService) WithGuildConfigProvider(p GuildConfigProvider) *RoundService {
	s.guildConfigProvider = p
	return s
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(ctx context.Context, operationName string, roundID sharedtypes.RoundID, serviceFunc func(ctx context.Context) (RoundOperationResult, error), logger *slog.Logger, metrics roundmetrics.RoundMetrics, tracer trace.Tracer) (result RoundOperationResult, err error) {
	if serviceFunc == nil {
		return RoundOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("round_id", roundID.String()),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, "RoundService")

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, "RoundService", duration)
	}()

	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, "RoundService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = RoundOperationResult{}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	result, err = serviceFunc(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.ExtractCorrelationID(ctx),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, "RoundService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(ctx, operationName, "RoundService")

	return result, nil
}

// RoundOperationResult represents a generic result from a round operation
type RoundOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
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
