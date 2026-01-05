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
		&roundevents.DiscordReminderPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			discordReminderPayload := payload.(*roundevents.DiscordReminderPayloadV1)

			// Additional instrumentation to ensure guild_id propagation for reminder flow
			h.logger.DebugContext(ctx, "Reminder handler payload debug",
				attr.RoundID("round_id", discordReminderPayload.RoundID),
				attr.String("guild_id", string(discordReminderPayload.GuildID)),
				attr.String("reminder_type", discordReminderPayload.ReminderType),
			)

			// Add debugging info to track duplicate processing
			messageID := msg.UUID
			deliveredCount := msg.Metadata.Get("Delivered")
			retryCount := msg.Metadata.Get("retry_count")

			h.logger.InfoContext(ctx, "Received RoundReminder event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", discordReminderPayload.RoundID),
				attr.String("reminder_type", discordReminderPayload.ReminderType),
				attr.String("message_id", messageID),
				attr.String("delivered_count", deliveredCount),
				attr.String("retry_count", retryCount),
				attr.String("event_message_id", discordReminderPayload.EventMessageID),
			)

			// Check if this is a duplicate message by logging all metadata
			h.logger.InfoContext(ctx, "Message metadata debug",
				attr.String("message_id", messageID),
				attr.Any("all_metadata", msg.Metadata),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ProcessRoundReminder(ctx, *discordReminderPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to process round reminder",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process round reminder: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round reminder processing failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundErrorV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				discordPayload := result.Success.(*roundevents.DiscordReminderPayloadV1)

				h.logger.InfoContext(ctx, "Round reminder processed successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", discordPayload.RoundID),
					attr.String("original_message_id", msg.UUID),
					attr.Int("participants_to_notify", len(discordPayload.UserIDs)),
				)

				// Only publish Discord reminder if there are participants to notify
				if len(discordPayload.UserIDs) > 0 {
					successMsg, err := h.helpers.CreateNewMessage(
						discordPayload,
						roundevents.RoundReminderSentV1,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create Discord reminder message: %w", err)
					}

					// Log the outgoing message details
					h.logger.InfoContext(ctx, "Publishing Discord reminder message",
						attr.String("original_message_id", msg.UUID),
						attr.String("new_message_id", successMsg.UUID),
						attr.String("topic", roundevents.RoundReminderSentV1),
						attr.RoundID("round_id", discordPayload.RoundID),
						attr.Int("participants", len(discordPayload.UserIDs)),
					)

					return []*message.Message{successMsg}, nil
				} else {
					// No participants to notify, but processing was successful
					h.logger.InfoContext(ctx, "Round reminder processed but no participants to notify",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", discordPayload.RoundID),
						attr.String("original_message_id", msg.UUID),
					)
					return []*message.Message{}, nil
				}
			}

			// This should never happen now that service always returns Success or Failure
			h.logger.ErrorContext(ctx, "Unexpected result from ProcessRoundReminder service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("service returned neither success nor failure")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
