package roundhandlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/logging"
	roundmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/round"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/tracing"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	roundservice "github.com/Black-And-White-Club/frolf-bot/app/modules/round/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// RoundHandlers handles round-related events.
type RoundHandlers struct {
	roundService   roundservice.Service
	logger         lokifrolfbot.Logger
	tracer         tempofrolfbot.Tracer
	metrics        roundmetrics.RoundMetrics
	helpers        utils.Helpers
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewRoundHandlers creates a new instance of RoundHandlers.
func NewRoundHandlers(
	roundService roundservice.Service,
	logger lokifrolfbot.Logger,
	tracer tempofrolfbot.Tracer,
	helpers utils.Helpers,
	metrics roundmetrics.RoundMetrics,
) Handlers {
	return &RoundHandlers{
		roundService: roundService,
		logger:       logger,
		tracer:       tracer,
		helpers:      helpers,
		metrics:      metrics,
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
	metrics roundmetrics.RoundMetrics,
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
				metrics.RecordHandlerFailure(handlerName)
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
			metrics.RecordHandlerFailure(handlerName)
			return nil, err
		}

		logger.InfoContext(ctx, handlerName+" completed successfully", attr.CorrelationIDFromMsg(msg))
		metrics.RecordHandlerSuccess(handlerName)
		return result, nil
	}
}
