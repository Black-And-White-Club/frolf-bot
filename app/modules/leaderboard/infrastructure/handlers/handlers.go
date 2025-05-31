package leaderboardhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardmetrics "github.com/Black-And-White-Club/frolf-bot-shared/observability/otel/metrics/leaderboard"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// LeaderboardHandlers handles leaderboard-related events.
type LeaderboardHandlers struct {
	leaderboardService leaderboardservice.Service
	logger             *slog.Logger
	tracer             trace.Tracer
	metrics            leaderboardmetrics.LeaderboardMetrics
	Helpers            utils.Helpers
	handlerWrapper     func(handlerName string, unmarshalTo interface{}, handlerFunc func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error)) message.HandlerFunc
}

// NewLeaderboardHandlers creates a new instance of LeaderboardHandlers.
func NewLeaderboardHandlers(
	leaderboardService leaderboardservice.Service,
	logger *slog.Logger,
	tracer trace.Tracer,
	helpers utils.Helpers,
	metrics leaderboardmetrics.LeaderboardMetrics,
) Handlers {
	return &LeaderboardHandlers{
		leaderboardService: leaderboardService,
		logger:             logger,
		tracer:             tracer,
		Helpers:            helpers,
		metrics:            metrics,
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
	metrics leaderboardmetrics.LeaderboardMetrics,
	tracer trace.Tracer,
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
			duration := time.Since(startTime) // Use time.Duration directly
			metrics.RecordHandlerDuration(ctx, handlerName, duration)
		}()

		logger.InfoContext(ctx, handlerName+" triggered",
			attr.CorrelationIDFromMsg(msg),
			attr.String("message_id", msg.UUID),
		)

		// Create a new instance of the payload type
		// Ensure payloadInstance is a pointer if unmarshalTo is a pointer to a struct
		var payloadInstance interface{}
		if unmarshalTo != nil {
			// Use reflection to create a new instance of the type pointed to by unmarshalTo
			// This assumes unmarshalTo is a pointer to a struct
			payloadInstance = utils.NewInstance(unmarshalTo)
		}

		// Unmarshal payload if a target is provided
		if payloadInstance != nil {
			if err := helpers.UnmarshalPayload(msg, payloadInstance); err != nil {
				if _, ok := err.(*json.UnmarshalTypeError); ok || strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "cannot unmarshal") {
					logger.ErrorContext(ctx, "Permanent unmarshal error, terminating message",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
						attr.String("message_uuid", msg.UUID),
					)
					metrics.RecordHandlerFailure(ctx, handlerName)
					// Return nil, nil to signal Watermill that this message is processed (failed permanently)
					return nil, nil
				} else {
					// For other types of errors from UnmarshalPayload, treat as potentially temporary
					logger.ErrorContext(ctx, "Transient error during unmarshal, retrying message",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
						attr.String("message_uuid", msg.UUID),
					)
					metrics.RecordHandlerFailure(ctx, handlerName)
					// Return the error to Watermill for retries
					return nil, fmt.Errorf("transient unmarshal error: %w", err)
				}
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
