package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundUpdateRequest",
		&roundevents.RoundUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundUpdateRequestPayload := payload.(*roundevents.RoundUpdateRequestPayload)

			h.logger.InfoContext(ctx, "Received RoundUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundUpdateRequestPayload.RoundID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ValidateRoundUpdateRequest(ctx, *roundUpdateRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundUpdateRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundUpdateRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Round update request validation failed",
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
				h.logger.InfoContext(ctx, "Round update request validated", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				validatedPayload := result.Success.(*roundevents.RoundUpdateValidatedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					validatedPayload,
					roundevents.RoundUpdateValidated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ValidateRoundUpdateRequest service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
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

			// Call the service function to handle the event
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
				h.logger.InfoContext(ctx, "Round entity updated successfully", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				updatedPayload := result.Success.(*roundevents.RoundEntityUpdatedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					updatedPayload,
					roundevents.RoundUpdated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateRoundEntity service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundScheduleUpdate(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundScheduleUpdate",
		&roundevents.RoundScheduleUpdatePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundScheduleUpdatePayload := payload.(*roundevents.RoundScheduleUpdatePayload)

			h.logger.InfoContext(ctx, "Received RoundScheduleUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundScheduleUpdatePayload.RoundID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.UpdateScheduledRoundEvents(ctx, *roundScheduleUpdatePayload)
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
				h.logger.InfoContext(ctx, "Scheduled round update successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				storedPayload := result.Success.(*roundevents.RoundStoredPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					storedPayload,
					roundevents.RoundScheduleUpdate,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateScheduledRoundEvents service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
