package userservice

import (
	"errors"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/eventbus"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	usertypes "github.com/Black-And-White-Club/frolf-bot-shared/types/user"
	userdb "github.com/Black-And-White-Club/frolf-bot/app/modules/user/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserServiceImpl handles user-related logic.
type UserServiceImpl struct {
	UserDB         userdb.UserDB
	eventBus       eventbus.EventBus
	logger         lokifrolfbot.Logger
	metrics        usermetrics.UserMetrics
	tracer         tempofrolfbot.Tracer
	serviceWrapper func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error)
}

// NewUser Service creates a new UserService.
func NewUserService(
	db userdb.UserDB,
	eventBus eventbus.EventBus,
	logger lokifrolfbot.Logger,
	metrics usermetrics.UserMetrics,
	tracer tempofrolfbot.Tracer,
) Service {
	return &UserServiceImpl{
		UserDB:   db,
		eventBus: eventBus,
		logger:   logger,
		metrics:  metrics,
		tracer:   tracer,
		// Assign the serviceWrapper method
		serviceWrapper: func(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error)) (UserOperationResult, error) {
			return serviceWrapper(msg, operationName, userID, serviceFunc, logger, metrics, tracer)
		},
	}
}

// serviceWrapper handles common tracing, logging, and metrics for service operations.
func serviceWrapper(msg *message.Message, operationName string, userID usertypes.DiscordID, serviceFunc func() (UserOperationResult, error), logger lokifrolfbot.Logger, metrics usermetrics.UserMetrics, tracer tempofrolfbot.Tracer) (result UserOperationResult, err error) {
	ctx, span := tracer.StartSpan(msg.Context(), operationName, msg)
	defer span.End()

	msg = msg.Copy()
	msg.SetContext(ctx)

	metrics.RecordOperationAttempt(operationName, userID)

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.RecordOperationDuration(operationName, duration)
	}()

	logger.Info(operationName+" triggered",
		attr.CorrelationIDFromMsg(msg),
		attr.String("message_id", msg.UUID),
		attr.String("operation", operationName),
		attr.String("user_id", string(userID)),
	)

	// Modify `err` directly inside the defer so it propagates correctly
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Panic in %s: %v", operationName, r)
			logger.Error(errorMsg,
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Any("panic", r),
			)
			metrics.RecordOperationFailure(operationName, userID)
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

		logger.Error("Error in "+operationName,
			attr.CorrelationIDFromMsg(msg),
			attr.String("user_id", string(userID)),
			attr.Error(wrappedErr),
		)
		metrics.RecordOperationFailure(operationName, userID)
		span.RecordError(wrappedErr)
		return result, wrappedErr
	}

	logger.Info(operationName+" completed successfully",
		attr.CorrelationIDFromMsg(msg),
		attr.String("user_id", string(userID)),
		attr.String("operation", operationName),
	)
	metrics.RecordOperationSuccess(operationName, userID)

	return result, nil
}

// UserOperationResult represents a generic result from a user operation
type UserOperationResult struct {
	Success interface{}
	Failure interface{}
	Error   error
}
