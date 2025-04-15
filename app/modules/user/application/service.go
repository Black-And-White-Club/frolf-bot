package userservice

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
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
	serviceWrapper func(msg *message.Message, operationName string, userID sharedtypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error)
}

// NewUser Service creates a new UserService.
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
		// Assign the serviceWrapper method
		serviceWrapper: func(msg *message.Message, operationName string, userID sharedtypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
			return serviceWrapper(msg, operationName, userID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(msg *message.Message, operationName string, userID sharedtypes.DiscordID, serviceFunc func() (UserOperationResult, error), logger *slog.Logger, metrics usermetrics.UserMetrics, tracer trace.Tracer) (result UserOperationResult, err error) {
	ctx, span := tracer.Start(msg.Context(), operationName, trace.WithAttributes(
		attribute.String("message.id", msg.UUID),
		attribute.String("message.correlation_id", middleware.MessageCorrelationID(msg)),
	))
	// Ensure the span ends when the function returns
	defer span.End()

	msg = msg.Copy()
	msg.SetContext(ctx)

	metrics.RecordOperationAttempt(ctx, operationName, userID)

	startTime := time.Now()
	defer func() {
		duration := time.Duration(time.Since(startTime).Seconds())
		metrics.RecordOperationDuration(ctx, operationName, duration, userID)
	}()

	logger.InfoContext(ctx, operationName+" triggered",
		attr.CorrelationIDFromMsg(msg),
		attr.String("message_id", msg.UUID),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)

	// Modify `err` directly inside the defer so it propagates correctly
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.ErrorContext(ctx, errorMsg,
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(ctx, operationName, userID)
			span.RecordError(errors.New(errorMsg))

			// Since result and err are named return values, modifying them here affects the function return
			result = UserOperationResult{}
			err = fmt.Errorf("%s", errorMsg)
		}
	}()

	// Now, if `serviceFunc()` panics, `defer` will catch it and modify `err`
	result, err = serviceFunc()
	if err != nil {
		wrappedErr := fmt.Errorf("%s operation failed: %w", operationName, err)

		logger.ErrorContext(ctx, "Error in "+operationName,
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(ctx, operationName, userID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.InfoContext(ctx, operationName+" completed successfully",
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)),
		attr.String("operation", operationName),
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
