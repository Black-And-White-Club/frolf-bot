package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleRoundStarted(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundStarted",
		&roundevents.RoundStartedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundStartedPayload := payload.(*roundevents.RoundStartedPayload)

			h.logger.Info("Received RoundStarted event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundStartedPayload.RoundID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ProcessRoundStart(ctx, *roundStartedPayload)
			if err != nil {
				h.logger.Error("Failed to handle RoundStarted event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundStarted event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Round start processing failed",
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
				h.logger.Info("Round start processed successfully", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				discordStartPayload := result.Success.(*roundevents.DiscordRoundStartPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					discordStartPayload,
					roundevents.DiscordRoundStarted,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from ProcessRoundStart service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
