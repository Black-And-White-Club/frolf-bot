package scorehandlers

import (
	"context"
	"fmt"
	"log/slog" // Import slog
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/score"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace" // Import otel trace
)

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	scoreService   scoreservice.Service
	logger         *slog.Logger // Change to slog
	tracer         trace.Tracer // Change to otel trace
	metrics        scoremetrics.ScoreMetrics
	helpers        utils.Helpers
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewScoreHandlers creates a new ScoreHandlers.
func NewScoreHandlers(
	scoreService scoreservice.Service,
	logger *slog.Logger, // Change to slog
	tracer trace.Tracer, // Change to otel trace
	helpers utils.Helpers,
	metrics scoremetrics.ScoreMetrics,
) Handlers {
	return &ScoreHandlers{
		scoreService: scoreService,
		logger:       logger,
		tracer:       tracer,
		helpers:      helpers,
		metrics:      metrics,
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
	logger *slog.Logger, // Change to slog
	metrics scoremetrics.ScoreMetrics,
	tracer trace.Tracer, // Change to otel trace
	helpers utils.Helpers,
) message.HandlerFunc {
	return func(msg *message.Message) ([]*message.Message, error) {
		// Start a span for tracing
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
