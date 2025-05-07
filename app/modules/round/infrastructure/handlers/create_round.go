package roundhandlers

import (
	"context"
	"errors"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
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

func (h *RoundHandlers) HandleRoundEventMessageIDUpdate(msg *message.Message) ([]*message.Message, error) {
	// Use the handlerWrapper for consistent logging, tracing, and error handling
	return h.handlerWrapper(
		"HandleRoundEventMessageIDUpdate",          // Operation name for logging/tracing
		&roundevents.RoundMessageIDUpdatePayload{}, // Expected payload type for unmarshalling
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Assert the unmarshalled payload to the correct type
			updatePayload, ok := payload.(*roundevents.RoundMessageIDUpdatePayload)
			if !ok {
				h.logger.ErrorContext(ctx, "Received unexpected payload type for RoundEventMessageIDUpdate",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("payload_type", fmt.Sprintf("%T", payload)),
				)
				return nil, errors.New("unexpected payload type")
			}

			roundID := updatePayload.RoundID

			h.logger.InfoContext(ctx, "Received RoundEventMessageIDUpdate event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
			)

			// Extract the Discord message ID from the message metadata
			discordMessageID, ok := msg.Metadata["discord_message_id"]
			if !ok || discordMessageID == "" {
				h.logger.ErrorContext(ctx, "Discord message ID not found or empty in metadata",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
				)
				return nil, errors.New("discord message ID not found in metadata")
			}

			// 1. Update the round in the database with the Discord message ID
			// Call the service method which now returns the updated round object
			updatedRound, err := h.roundService.UpdateRoundMessageID(ctx, roundID, discordMessageID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to update round with Discord message ID via service",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.String("discord_message_id", discordMessageID),
					attr.Error(err),
				)
				// Return the error so Watermill retries the message
				return nil, fmt.Errorf("failed to update round message ID: %w", err)
			}

			// Check if the updatedRound is nil (e.g., if UpdateRoundMessageID didn't find the round)
			if updatedRound == nil {
				h.logger.ErrorContext(ctx, "Updated round object is nil after UpdateRoundMessageID service call",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
				)
				// This might indicate a deeper issue or a race condition, handle appropriately
				return nil, errors.New("updated round object is nil")
			}

			h.logger.InfoContext(ctx, "Successfully updated round with Discord message ID in DB",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
				attr.String("discord_message_id", discordMessageID),
			)

			// 2. Construct the RoundScheduledPayload using the updated round object
			scheduledPayload := roundevents.RoundScheduledPayload{
				BaseRoundPayload: roundtypes.BaseRoundPayload{
					RoundID:     updatedRound.ID,
					Title:       updatedRound.Title,
					Description: updatedRound.Description,
					Location:    updatedRound.Location,
					StartTime:   updatedRound.StartTime,
					UserID:      updatedRound.CreatedBy,
				},
				EventMessageID: discordMessageID, // Use the Discord message ID from metadata
			}

			// 3. Publish the RoundScheduled event
			// This event will be consumed by the (now refactored) ScheduleRoundEvents function
			scheduledMsg, err := h.helpers.CreateResultMessage(
				msg, // Use the original message for metadata propagation (like correlation ID)
				scheduledPayload,
				roundevents.RoundEventMessageIDUpdated,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create RoundScheduled message",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundID),
					attr.Error(err),
				)
				// Return the error so Watermill retries the message
				return nil, fmt.Errorf("failed to create RoundScheduled message: %w", err)
			}

			h.logger.InfoContext(ctx, "Successfully published RoundScheduled event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundID),
			)

			// Return the message to be published
			return []*message.Message{scheduledMsg}, nil
		},
	)(msg) // Execute the wrapped handler with the incoming message
}
