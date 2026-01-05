package guildservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// GuildServiceImpl handles guild-related logic.
type GuildService struct {
	GuildDB        guilddb.GuildDB
	eventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        guildmetrics.GuildMetrics
	tracer         trace.Tracer
	serviceWrapper func(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error)) (GuildOperationResult, error)
}

// NewGuildService creates a new GuildService.
func NewGuildService(
	db guilddb.GuildDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics guildmetrics.GuildMetrics,
	tracer trace.Tracer,
) Service {
	return &GuildService{
		GuildDB:  db,
		eventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		serviceWrapper: func(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error)) (result GuildOperationResult, err error) {
			return serviceWrapper(ctx, operationName, guildID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper is a helper function that wraps service operations with common logic.
func serviceWrapper(ctx context.Context, operationName string, guildID sharedtypes.GuildID, serviceFunc func(ctx context.Context) (GuildOperationResult, error), logger *slog.Logger, metrics guildmetrics.GuildMetrics, tracer trace.Tracer) (result GuildOperationResult, err error) {
	if ctx == nil {
		err := errors.New("context cannot be nil")
		return GuildOperationResult{
			Success: nil,
			Failure: nil,
			Error:   err,
		}, err
	}

	if serviceFunc == nil {
		err := errors.New("service function is nil")
		return GuildOperationResult{
			Success: nil,
			Failure: nil,
			Error:   err,
		}, err
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("guild_id", string(guildID)),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, guildID, "GuildService")

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, guildID, "GuildService", duration)
	}()

	logger.InfoContext(ctx, "Operation triggered",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("guild_id", string(guildID)),
	)

	// Handle panic recovery
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.ExtractCorrelationID(ctx),
				attr.String("guild_id", string(guildID)),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, guildID, "GuildService")
			span.RecordError(errors.New(errorMsg))

			// Set the return values explicitly for panic cases
			result = GuildOperationResult{
				Success: nil,
				Failure: nil,
				Error:   fmt.Errorf("%s", errorMsg),
			}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	result, err = serviceFunc(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.ExtractCorrelationID(ctx),
			attr.String("guild_id", string(guildID)),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, guildID, "GuildService")
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.ExtractCorrelationID(ctx),
		attr.String("operation", operationName),
		attr.String("guild_id", string(guildID)),
	)
	metrics.RecordOperationSuccess(ctx, operationName, guildID, "GuildService")

	return result, nil
}

// GuildOperationResult represents a generic result from a guild operation
type GuildOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
