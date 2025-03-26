package scorehandlers

import (
	"context"
	"fmt"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	lokifrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/loki"
	scoremetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/prometheus/score"
	tempofrolfbot "github.com/Black-And-White-Club/frolf-bot-shared/observability/tempo"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	scoreservice "github.com/Black-And-White-Club/frolf-bot/app/modules/score/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// ScoreHandlers handles score-related events.
type ScoreHandlers struct {
	scoreService   scoreservice.Service
	logger         lokifrolfbot.Logger
	tracer         tempofrolfbot.Tracer
	metrics        scoremetrics.ScoreMetrics
	helpers        utils.Helpers
	handlerWrapper func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewScoreHandlers creates a new ScoreHandlers.
func NewScoreHandlers(
	scoreService scoreservice.Service,
	logger lokifrolfbot.Logger,
	tracer tempofrolfbot.Tracer,
	helpers utils.Helpers,
	metrics scoremetrics.ScoreMetrics,
) Handlers {
	return &ScoreHandlers{
		scoreService: scoreService,
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
	metrics scoremetrics.ScoreMetrics,
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
					attr.Error(err),
				)
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
