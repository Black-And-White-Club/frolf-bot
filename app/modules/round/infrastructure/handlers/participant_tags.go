package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleScheduledRoundTagUpdate(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScheduledRoundTagUpdate",
		&roundevents.ScheduledRoundTagUpdatePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scheduledRoundTagUpdatePayload := payload.(*roundevents.ScheduledRoundTagUpdatePayload)

			h.logger.InfoContext(ctx, "Received ScheduledRoundTagUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("changed_tags", scheduledRoundTagUpdatePayload.ChangedTags),
			)

			// Call the service function to handle the event
			result, err := h.roundService.UpdateScheduledRoundsWithNewTags(ctx, *scheduledRoundTagUpdatePayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScheduledRoundTagUpdate event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ScheduledRoundTagUpdate event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scheduled round tag update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Scheduled round tag update successful",
					attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				// Remove the type assertion since result.Success is already the correct type
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success, // This is already a pointer to the correct type
					roundevents.TagsUpdatedForScheduledRounds,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// This should never happen now that service always returns Success
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateScheduledRoundsWithNewTags service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("service returned neither success nor failure")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
