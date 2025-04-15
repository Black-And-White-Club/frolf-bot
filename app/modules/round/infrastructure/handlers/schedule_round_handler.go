package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundStored(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundStored",
		&roundevents.RoundStoredPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundStoredPayload := payload.(*roundevents.RoundStoredPayload)

			h.logger.InfoContext(ctx, "Received RoundStored event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundStoredPayload.Round.ID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ScheduleRoundEvents(ctx, *roundStoredPayload, *roundStoredPayload.Round.StartTime)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundStored event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundStored event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round scheduling failed",
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
				h.logger.InfoContext(ctx, "Round scheduling successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				scheduledPayload := result.Success.(*roundevents.RoundScheduledPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					scheduledPayload,
					roundevents.RoundScheduled,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ScheduleRoundEvents service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
