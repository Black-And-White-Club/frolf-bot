package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundDeleteRequest",
		&roundevents.RoundDeleteRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundDeleteRequestPayload := payload.(*roundevents.RoundDeleteRequestPayload)

			h.logger.InfoContext(ctx, "Received RoundDeleteRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundDeleteRequestPayload.RoundID.String()),
				attr.String("requesting_user", string(roundDeleteRequestPayload.RequestingUserUserID)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ValidateRoundDeleteRequest(ctx, *roundDeleteRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundDeleteRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundDeleteRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round delete request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundDeleteError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Round delete request validated", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundDeleteAuthorized,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ValidateRoundDeleteRequest service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundDeleteAuthorized(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundDeleteAuthorized",
		&roundevents.RoundDeleteAuthorizedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundDeleteAuthorizedPayload := payload.(*roundevents.RoundDeleteAuthorizedPayload)

			h.logger.InfoContext(ctx, "Received RoundDeleteAuthorized event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundDeleteAuthorizedPayload.RoundID.String()),
			)

			// Call the service function to handle the event
			result, err := h.roundService.DeleteRound(ctx, *roundDeleteAuthorizedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundDeleteAuthorized event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundDeleteAuthorized event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round delete authorized failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundDeleteError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Round delete authorized successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundDeleted,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from DeleteRound service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
