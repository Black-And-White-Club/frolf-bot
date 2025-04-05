package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundReminder(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundReminder",
		&roundevents.DiscordReminderPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			discordReminderPayload := payload.(*roundevents.DiscordReminderPayload)

			h.logger.Info("Received RoundReminder event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", discordReminderPayload.RoundID),
				attr.String("reminder_type", discordReminderPayload.ReminderType),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ProcessRoundReminder(ctx, *discordReminderPayload)
			if err != nil {
				h.logger.Error("Failed to handle RoundReminder event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundReminder event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Round reminder processing failed",
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
				h.logger.Info("Round reminder processed successfully", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				reminderProcessedPayload := result.Success.(*roundevents.RoundReminderProcessedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					reminderProcessedPayload,
					roundevents.DiscordRoundReminder,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}
			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from ProcessRoundReminder service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
