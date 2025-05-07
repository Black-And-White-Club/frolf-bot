package userhandlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	userService    userservice.Service
	logger         *slog.Logger
	tracer         trace.Tracer
	metrics        usermetrics.UserMetrics
	helpers        utils.Helpers
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewUser Handlers creates a new UserHandlers.
func NewUserHandlers(
	userService userservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics usermetrics.UserMetrics,
) Handlers {
	if helpers == nil {
		logger.Error("Helpers is nil")
		return nil
	}
	return &UserHandlers{
		userService: userService,
		logger:      logger,
		tracer:      tracer,
		helpers:     helpers,
		metrics:     metrics,
		// Assign the standalone handlerWrapper function
		handlerWrapper: func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc {
			return handlerWrapper(handlerName, unmarshalTo, handlerFunc, logger, metrics, tracer, helpers)
		},
	}
}

// handlerWrapper is a standalone function that handles common tracing, logging, and metrics for handlers.
func handlerWrapper(
	handlerName string,
	unmarshalTo interface{},
	handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error),
	logger *slog.Logger,
	metrics usermetrics.UserMetrics,
	tracer trace.Tracer,
	helpers utils.Helpers,
) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		ctx, span := tracer.Start(msg.Context(), handlerName, trace.WithAttributes(
			attribute.String("message.id", msg.UUID),
			attribute.String("message.correlation_id", middleware.MessageCorrelationID(msg)),
		))
		defer span.End()

		// Record metrics for handler attempt
		metrics.RecordHandlerAttempt(ctx, handlerName)

		startTime := time.Now()
		defer func() {
			duration := time.Duration(time.Since(startTime).Seconds())
			metrics.RecordHandlerDuration(ctx, handlerName, duration)
		}()

		logger.InfoContext(ctx, handlerName+" triggered",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
		)

		// Create a new instance of the payload type
		payloadInstance := unmarshalTo

		// Unmarshal payload if a target is provided
		if payloadInstance != nil {
			if err := helpers.UnmarshalPayload(msg, payloadInstance); err != nil {
				logger.ErrorContext(ctx, "Failed to unmarshal payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err))
				metrics.RecordHandlerFailure(ctx, handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		// Call the actual handler logic
		result, err := handlerFunc(ctx, msg, payloadInstance)
		if err != nil {
			logger.ErrorContext(ctx, "Error in "+handlerName,
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err),
			)
			metrics.RecordHandlerFailure(ctx, handlerName)
			return nil, err
		}

		logger.InfoContext(ctx, handlerName+" completed successfully", attr.CorrelationIDFromMsg(msg))
		metrics.RecordHandlerSuccess(ctx, handlerName)
		return result, nil
	}
}
