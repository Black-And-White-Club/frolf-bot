package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleGetRoundRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetRoundRequest",
		&roundevents.GetRoundRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			getRoundRequestPayload := payload.(*roundevents.GetRoundRequestPayload)

			h.logger.InfoContext(ctx, "Received GetRoundRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", getRoundRequestPayload.RoundID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.GetRound(ctx, getRoundRequestPayload.RoundID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle GetRoundRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle GetRoundRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Get round request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Get round request successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				round := result.Success.(*roundtypes.Round)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					round,
					roundevents.RoundRetrieved,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from GetRound service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
