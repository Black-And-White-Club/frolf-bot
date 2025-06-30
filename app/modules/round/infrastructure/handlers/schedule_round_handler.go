package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleDiscordMessageIDUpdate handles the event published after a round has been
// successfully updated with the Discord message ID and is ready for scheduling.
// It calls the service method to schedule the reminder and start events.
func (h *RoundHandlers) HandleDiscordMessageIDUpdated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleDiscordMessageIDUpdate",
		&roundevents.RoundScheduledPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scheduledPayload := payload.(*roundevents.RoundScheduledPayload)

			h.logger.InfoContext(ctx, "Received RoundScheduled event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", scheduledPayload.RoundID),
				attr.String("discord_message_id", scheduledPayload.EventMessageID),
			)

			result, err := h.roundService.ScheduleRoundEvents(ctx, scheduledPayload.GuildID, *scheduledPayload, scheduledPayload.EventMessageID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during ScheduleRoundEvents service call",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", scheduledPayload.RoundID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to schedule round events: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round events scheduling failed in service",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", scheduledPayload.RoundID),
					attr.Any("failure_payload", result.Failure),
				)

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
				h.logger.InfoContext(ctx, "Round events scheduling successful",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", scheduledPayload.RoundID),
				)

				// Since this handler only schedules events and doesn't publish anything,
				// we return an empty slice to indicate successful processing
				return []*message.Message{}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from ScheduleRoundEvents service",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", scheduledPayload.RoundID),
			)
			return nil, fmt.Errorf("service returned neither success nor failure")
		},
	)

	return wrappedHandler(msg)
}
