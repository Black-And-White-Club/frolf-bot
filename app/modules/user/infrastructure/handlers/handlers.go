package userhandlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	usermetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/user"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserHandlers handles user-related events.
type UserHandlers struct {
	userService    userservice.Service
	logger         lokifrolfbot.Logger
	tracer         tempofrolfbot.Tracer
	metrics        usermetrics.UserMetrics
	helpers        utils.Helpers
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewUser Handlers creates a new UserHandlers.
func NewUserHandlers(
	userService userservice.Service,
	logger lokifrolfbot.Logger,
	tracer tempofrolfbot.Tracer,
	helpers utils.Helpers,
	metrics usermetrics.UserMetrics,
) Handlers {
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
	logger lokifrolfbot.Logger,
	metrics usermetrics.UserMetrics,
	tracer tempofrolfbot.Tracer,
	helpers utils.Helpers,
) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		// Start a span for tracing
		ctx, span := tracer.StartSpan(msg.Context(), handlerName, msg)
		defer span.End()

		// Record metrics for handler attempt
		metrics.RecordHandlerAttempt(handlerName)

		startTime := time.Now()
		defer func() {
			duration := time.Since(startTime).Seconds()
			metrics.RecordHandlerDuration(handlerName, duration)
		}()

		logger.Info(handlerName+" triggered",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
		)

		// Create a new instance of the payload type
		payloadInstance := unmarshalTo

		// Unmarshal payload if a target is provided
		if payloadInstance != nil {
			if err := helpers.UnmarshalPayload(msg, payloadInstance); err != nil {
				logger.Error("Failed to unmarshal payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err))
				metrics.RecordHandlerFailure(handlerName)
				return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
			}
		}

		// Call the actual handler logic
		result, err := handlerFunc(ctx, msg, payloadInstance)
		if err != nil {
			logger.Error("Error in "+handlerName,
				attr.CorrelationIDFromMsg(msg),
				attr.Error(err),
			)
			metrics.RecordHandlerFailure(handlerName)
			return nil, err
		}

		logger.Info(handlerName+" completed successfully", attr.CorrelationIDFromMsg(msg))
		metrics.RecordHandlerSuccess(handlerName)
		return result, nil
	}
}
