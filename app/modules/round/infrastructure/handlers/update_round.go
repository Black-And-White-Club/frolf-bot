package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundUpdateRequest",
		&roundevents.UpdateRoundRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updateRequestPayload := payload.(*roundevents.UpdateRoundRequestedPayload)

			h.logger.InfoContext(ctx, "Received RoundUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", updateRequestPayload.RoundID),
			)

			// ✅ Debug log incoming metadata
			h.logger.InfoContext(ctx, "DEBUG: HandleRoundUpdateRequest received metadata",
				attr.String("channel_id", msg.Metadata.Get("channel_id")),
				attr.String("message_id", msg.Metadata.Get("message_id")))

			result, err := h.roundService.ValidateAndProcessRoundUpdate(ctx, *updateRequestPayload, roundtime.NewTimeParser())
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to validate and process round update",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to validate and process round update: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round update validation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				// ✅ Preserve Discord metadata
				failureMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
				failureMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
				failureMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Round update validation successful",
					attr.CorrelationIDFromMsg(msg))

				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundUpdateValidated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// ✅ Preserve Discord metadata for the next step
				successMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
				successMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
				successMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

				// ✅ Debug log outgoing metadata
				h.logger.InfoContext(ctx, "DEBUG: HandleRoundUpdateRequest sending metadata",
					attr.String("channel_id", msg.Metadata.Get("channel_id")),
					attr.String("message_id", msg.Metadata.Get("message_id")))

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from ValidateAndProcessRoundUpdate service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundUpdateValidated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundUpdateValidated",
		&roundevents.RoundUpdateValidatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundUpdateValidatedPayload := payload.(*roundevents.RoundUpdateValidatedPayload)

			h.logger.InfoContext(ctx, "Received RoundUpdateValidated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundUpdateValidatedPayload.RoundUpdateRequestPayload.RoundID),
			)

			result, err := h.roundService.UpdateRoundEntity(ctx, *roundUpdateValidatedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundUpdateValidated event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundUpdateValidated event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round entity update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				failureMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
				failureMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
				failureMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				updatedPayload := result.Success.(*roundevents.RoundEntityUpdatedPayload)

				var messagesToReturn []*message.Message

				// Always create the main Discord update message
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					updatedPayload,
					roundevents.RoundUpdated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				successMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
				successMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
				successMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

				messagesToReturn = append(messagesToReturn, successMsg)

				// Check if we need to reschedule (only for time-sensitive fields)
				needsReschedule := h.shouldRescheduleEvents(roundUpdateValidatedPayload.RoundUpdateRequestPayload)

				if needsReschedule {
					h.logger.InfoContext(ctx, "Creating schedule update message for rescheduling",
						attr.RoundID("round_id", updatedPayload.Round.ID),
					)

					// Create schedule update message using RoundEntityUpdatedPayload
					scheduleMsg, err := h.helpers.CreateResultMessage(
						msg,
						updatedPayload, // Send the RoundEntityUpdatedPayload
						roundevents.RoundScheduleUpdate,
					)
					if err != nil {
						h.logger.WarnContext(ctx, "Failed to create schedule message, continuing without rescheduling",
							attr.Error(err))
					} else {
						scheduleMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
						scheduleMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
						scheduleMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

						messagesToReturn = append(messagesToReturn, scheduleMsg)
					}
				}

				h.logger.InfoContext(ctx, "DEBUG: HandleRoundUpdateValidated sending metadata",
					attr.String("channel_id", msg.Metadata.Get("channel_id")),
					attr.String("message_id", msg.Metadata.Get("message_id")))

				return messagesToReturn, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from UpdateRoundEntity service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	return wrappedHandler(msg)
}

// Helper method to determine if rescheduling is needed
func (h *RoundHandlers) shouldRescheduleEvents(payload roundevents.RoundUpdateRequestPayload) bool {
	// Only reschedule if the START TIME changed
	return payload.StartTime != nil
}

func (h *RoundHandlers) HandleRoundScheduleUpdate(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundScheduleUpdate",
		&roundevents.RoundEntityUpdatedPayload{}, // Receives RoundEntityUpdatedPayload
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatedPayload := payload.(*roundevents.RoundEntityUpdatedPayload)

			roundID := updatedPayload.Round.ID

			h.logger.InfoContext(ctx, "Received RoundScheduleUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
			)

			// Create RoundScheduleUpdatePayload from the updated round data
			schedulePayload := roundevents.RoundScheduleUpdatePayload{
				RoundID:   updatedPayload.Round.ID,
				Title:     updatedPayload.Round.Title,
				StartTime: updatedPayload.Round.StartTime,
				Location:  updatedPayload.Round.Location,
			}

			// Call the service function with the converted payload
			result, err := h.roundService.UpdateScheduledRoundEvents(ctx, schedulePayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundScheduleUpdate event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundScheduleUpdate event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Scheduled round update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				failureMsg.Metadata.Set("channel_id", msg.Metadata.Get("channel_id"))
				failureMsg.Metadata.Set("message_id", msg.Metadata.Get("message_id"))
				failureMsg.Metadata.Set("user_id", msg.Metadata.Get("user_id"))

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Scheduled round update successful", attr.CorrelationIDFromMsg(msg))

				scheduleUpdatedPayload := result.Success.(*roundevents.RoundScheduleUpdatePayload)

				h.logger.InfoContext(ctx, "Round events successfully rescheduled",
					attr.RoundID("round_id", scheduleUpdatedPayload.RoundID),
					attr.Time("new_start_time", scheduleUpdatedPayload.StartTime.AsTime()),
					attr.String("channel_id", msg.Metadata.Get("channel_id")),
					attr.String("message_id", msg.Metadata.Get("message_id")),
					attr.String("user_id", msg.Metadata.Get("user_id")))

				// Since UpdateScheduledRoundEvents now handles everything internally,
				// we don't need to publish additional events for scheduling
				// The rescheduling is complete at this point
				return []*message.Message{}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from UpdateScheduledRoundEvents service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	return wrappedHandler(msg)
}
