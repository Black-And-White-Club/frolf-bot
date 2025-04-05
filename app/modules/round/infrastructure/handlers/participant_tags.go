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

			h.logger.Info("Received ScheduledRoundTagUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.Any("changed_tags", scheduledRoundTagUpdatePayload.ChangedTags),
			)

			// Call the service function to handle the event
			result, err := h.roundService.UpdateScheduledRoundsWithNewTags(ctx, *scheduledRoundTagUpdatePayload)
			if err != nil {
				h.logger.Error("Failed to handle ScheduledRoundTagUpdate event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ScheduledRoundTagUpdate event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Scheduled round tag update failed",
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
				h.logger.Info("Scheduled round tag update successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				discordUpdatePayload := result.Success.(*roundevents.DiscordRoundUpdatePayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					discordUpdatePayload,
					roundevents.TagsUpdatedForScheduledRounds,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from UpdateScheduledRoundsWithNewTags service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
