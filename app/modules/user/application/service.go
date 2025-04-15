package userservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB         userdb.UserDB
	eventBus       eventbus.EventBus
	logger         *slog.Logger
	metrics        usermetrics.UserMetrics
	tracer         trace.Tracer
	serviceWrapper func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error)
}

// NewUserService creates a new UserService.
func NewUserService(
	db userdb.UserDB,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
) Service {
	return &UserServiceImpl{
		UserDB:   db,
		eventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		serviceWrapper: func(ctx context.Context, operationName string, userID sharedtypes.DiscordID, serviceFunc func(ctx context.Context) (UserOperationResult, error)) (UserOperationResult, error) {
			return serviceWrapper(ctx, operationName, userID, serviceFunc, logger, metrics, tracer)
		},
	}
}

func serviceWrapper(
	ctx context.Context,
	operationName string,
	userID sharedtypes.DiscordID,
	serviceFunc func(ctx context.Context) (UserOperationResult, error),
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
) (result UserOperationResult, err error) {
	if serviceFunc == nil {
		return UserOperationResult{}, errors.New("service function is nil")
	}

	ctx, span := tracer.Start(ctx, operationName, trace.WithAttributes(
		attribute.String("operation", operationName),
		attribute.String("user_id", string(userID)),
	))
	defer span.End()

	metrics.RecordOperationAttempt(ctx, operationName, userID)

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, duration, userID)
	}()

	logger.InfoContext(ctx, operationName+" triggered",
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
		attr.ExtractCorrelationID(ctx),
	)

	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			err = fmt.Errorf("%s", errorMsg)
			result = UserOperationResult{}

			logger.ErrorContext(ctx, errorMsg,
				attr.String("user_id", string(userID)),
				attr.Any("panic", r),
				attr.ExtractCorrelationID(ctx),
			)
			metrics.RecordOperationFailure(ctx, operationName, userID)
			span.RecordError(err)
		}
	}()

	result, err = serviceFunc(ctx)
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)
		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
			attr.ExtractCorrelationID(ctx),
		)
		metrics.RecordOperationFailure(ctx, operationName, userID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
		attr.ExtractCorrelationID(ctx),
	)
	metrics.RecordOperationSuccess(ctx, operationName, userID)

	return result, nil
}

// UserOperationResult represents a generic result from a user operation
type UserOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
