package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtime "github.com/Black-And-White-Club/frolf-bot/app/modules/round/time_utils"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCreateRoundRequest handles the CreateRoundRequest event.
func (h *RoundHandlers) HandleCreateRoundRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleCreateRoundRequest",
		&roundevents.CreateRoundRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createRoundRequestedPayload := payload.(*roundevents.CreateRoundRequestedPayload)

			h.logger.InfoContext(ctx, "Received CreateRoundRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("title", string(createRoundRequestedPayload.Title)),
				attr.String("description", string(createRoundRequestedPayload.Description)),
				attr.String("location", string(createRoundRequestedPayload.Location)),
				attr.String("start_time", string(createRoundRequestedPayload.StartTime)),
				attr.String("user_id", string(createRoundRequestedPayload.UserID)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ValidateAndProcessRound(ctx, *createRoundRequestedPayload, roundtime.NewTimeParser())
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle CreateRoundRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle CreateRoundRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Create round request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundValidationFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Create round request successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundEntityCreated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ValidateAndProcessRound service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleRoundEntityCreated handles the RoundEntityCreated event.
func (h *RoundHandlers) HandleRoundEntityCreated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundEntityCreated",
		&roundevents.RoundEntityCreatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundEntityCreatedPayload := payload.(*roundevents.RoundEntityCreatedPayload)

			h.logger.InfoContext(ctx, "Received RoundEntityCreated event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundEntityCreatedPayload.Round.ID.String()),
				attr.String("title", string(roundEntityCreatedPayload.Round.Title)),
				attr.String("description", string(*roundEntityCreatedPayload.Round.Description)),
				attr.String("location", string(*roundEntityCreatedPayload.Round.Location)),
				attr.Time("start_time", roundEntityCreatedPayload.Round.StartTime.AsTime()),
				attr.String("user_id", string(roundEntityCreatedPayload.Round.CreatedBy)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.StoreRound(ctx, *roundEntityCreatedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle RoundEntityCreated event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundEntityCreated event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Store round failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundCreationFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Store round successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundCreated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from StoreRound service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
