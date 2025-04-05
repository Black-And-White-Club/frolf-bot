package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleAllScoresSubmitted(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleAllScoresSubmitted",
		&roundevents.AllScoresSubmittedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			allScoresSubmittedPayload := payload.(*roundevents.AllScoresSubmittedPayload)

			h.logger.Info("Received AllScoresSubmitted event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", allScoresSubmittedPayload.RoundID.String()),
			)

			// Call the service function to handle the event
			result, err := h.roundService.FinalizeRound(ctx, *allScoresSubmittedPayload)
			if err != nil {
				h.logger.Error("Failed to handle AllScoresSubmitted event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle AllScoresSubmitted event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Round finalization failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundFinalizationError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.Info("Round finalization successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundFinalized,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from FinalizeRound service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundFinalized(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundFinalized",
		&roundevents.RoundFinalizedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundFinalizedPayload := payload.(*roundevents.RoundFinalizedPayload)

			h.logger.Info("Received RoundFinalized event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundFinalizedPayload.RoundID.String()),
			)

			// Call the service function to handle the event
			result, err := h.roundService.NotifyScoreModule(ctx, *roundFinalizedPayload)
			if err != nil {
				h.logger.Error("Failed to handle RoundFinalized event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle RoundFinalized event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Notify Score Module failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundFinalizationError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.Info("Notify Score Module successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.ProcessRoundScoresRequest,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from NotifyScoreModule service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
