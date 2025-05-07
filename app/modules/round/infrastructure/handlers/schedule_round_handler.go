package roundhandlers

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleDiscordMessageIDUpdate handles the event published after a round has been
// successfully updated with the Discord message ID and is ready for scheduling.
// It calls the service method to schedule the reminder and start events.
func (h *RoundHandlers) HandleDiscordMessageIDUpdated(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleDiscordMessageIDUpdate",
		&roundevents.RoundScheduledPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scheduledPayload, ok := payload.(*roundevents.RoundScheduledPayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Received unexpected payload type for RoundScheduled",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_type", fmt.Sprintf("%T", payload)),
				)
				return nil, errors.New("unexpected payload type")
			}

			roundID := scheduledPayload.RoundID

			h.logger.InfoContext(ctx, "Received RoundScheduled event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", scheduledPayload.EventMessageID),
			)

			result, err := h.roundService.ScheduleRoundEvents(ctx, *scheduledPayload, scheduledPayload.EventMessageID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed during ScheduleRoundEvents service call",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to schedule round events: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round events scheduling failed in service",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundError,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after scheduling failure",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundID),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Round events scheduling successful",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
				)

				return nil, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from ScheduleRoundEvents service",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
			)
			return nil, errors.New("unexpected result from service")
		},
	)(msg)
}
