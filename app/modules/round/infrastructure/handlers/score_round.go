package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleScoreUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScoreUpdateRequest",
		&roundevents.ScoreUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateRequestPayload := payload.(*roundevents.ScoreUpdateRequestPayload)

			h.logger.InfoContext(ctx, "Received ScoreUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", scoreUpdateRequestPayload.RoundID),
				attr.String("participant_id", string(scoreUpdateRequestPayload.Participant)),
				attr.Int("score", int(*scoreUpdateRequestPayload.Score)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ValidateScoreUpdateRequest(ctx, *scoreUpdateRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScoreUpdateRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ScoreUpdateRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Score update request validation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundScoreUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Score update request validated", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				validatedPayload := result.Success.(*roundevents.ScoreUpdateValidatedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					validatedPayload,
					roundevents.RoundScoreUpdateValidated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from ValidateScoreUpdateRequest service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleScoreUpdateValidated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleScoreUpdateValidated",
		&roundevents.ScoreUpdateValidatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			scoreUpdateValidatedPayload := payload.(*roundevents.ScoreUpdateValidatedPayload)

			h.logger.InfoContext(ctx, "Received ScoreUpdateValidated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.RoundID),
				attr.String("participant_id", string(scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.Participant)),
				attr.Int("score", int(*scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.Score)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.UpdateParticipantScore(ctx, *scoreUpdateValidatedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ScoreUpdateValidated event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ScoreUpdateValidated event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Participant score update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundScoreUpdateError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Participant score updated successfully", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				updatedPayload := result.Success.(*roundevents.ParticipantScoreUpdatedPayload)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					updatedPayload,
					roundevents.RoundParticipantScoreUpdated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateParticipantScore service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleParticipantScoreUpdated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantScoreUpdated",
		&roundevents.ParticipantScoreUpdatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantScoreUpdatedPayload := payload.(*roundevents.ParticipantScoreUpdatedPayload)

			h.logger.InfoContext(ctx, "Received ParticipantScoreUpdated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", participantScoreUpdatedPayload.RoundID),
				attr.String("participant_id", string(participantScoreUpdatedPayload.Participant)),
				attr.Int("score", int(participantScoreUpdatedPayload.Score)),
				attr.RoundID("event_message_id", *participantScoreUpdatedPayload.EventMessageID),
			)

			// Call the service function to handle the event
			result, err := h.roundService.CheckAllScoresSubmitted(ctx, *participantScoreUpdatedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ParticipantScoreUpdated event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantScoreUpdated event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "All scores submitted check failed",
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
				h.logger.InfoContext(ctx, "All scores submitted check successful", attr.CorrelationIDFromMsg(msg))

				// Check if all scores have been submitted
				if allScoresSubmittedPayload, ok := result.Success.(*roundevents.AllScoresSubmittedPayload); ok {
					allScoresSubmittedMsg, err := h.helpers.CreateResultMessage(
						msg,
						allScoresSubmittedPayload,
						roundevents.RoundAllScoresSubmitted,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create all scores submitted message: %w", err)
					}

					return []*message.Message{allScoresSubmittedMsg}, nil
				} else if notAllScoresSubmittedPayload, ok := result.Success.(*roundevents.NotAllScoresSubmittedPayload); ok {
					notAllScoresSubmittedMsg, err := h.helpers.CreateResultMessage(
						msg,
						notAllScoresSubmittedPayload,
						roundevents.RoundNotAllScoresSubmitted,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create not all scores submitted message: %w", err)
					}

					return []*message.Message{notAllScoresSubmittedMsg}, nil
				}

				// If neither AllScoresSubmitted nor NotAllScoresSubmitted is set, return an error
				h.logger.ErrorContext(ctx, "Unexpected result from CheckAllScoresSubmitted service",
					attr.CorrelationIDFromMsg(msg),
				)
				return nil, fmt.Errorf("unexpected result from service")
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from CheckAllScoresSubmitted service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
