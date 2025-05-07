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
	// Use the handlerWrapper for tracing, logging, and error handling
	wrappedHandler := h.handlerWrapper( // Assuming h.handlerWrapper is defined
		"HandleScoreUpdateValidated",
		&roundevents.ScoreUpdateValidatedPayload{}, // Expecting this payload type
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			// Assert the payload to the expected type
			scoreUpdateValidatedPayload, ok := payload.(*roundevents.ScoreUpdateValidatedPayload)
			if !ok {
				// Log an error if the payload type is unexpected
				h.logger.ErrorContext(ctx, "Invalid payload type for HandleScoreUpdateValidated",
					attr.Any("payload", payload), // Log the received payload
				)
				// Return an error, this will likely be handled by the message bus error handler
				return nil, fmt.Errorf("invalid payload type for HandleScoreUpdateValidated")
			}

			// Log that the event was received
			h.logger.InfoContext(ctx, "Received ScoreUpdateValidated event",
				attr.CorrelationIDFromMsg(msg), // Assuming CorrelationIDFromMsg helper exists
				attr.RoundID("round_id", scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.RoundID), // Assuming attr.RoundID helper exists
				attr.String("participant_id", string(scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.Participant)),
				// Safely dereference the Score pointer for logging, handle nil case
				attr.Int("score", func() int {
					if scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.Score != nil {
						return int(*scoreUpdateValidatedPayload.ScoreUpdateRequestPayload.Score)
					}
					return 0 // Or some indicator for nil score
				}()),
			)

			// Call the service function to update the participant score in the database
			result, err := h.roundService.UpdateParticipantScore(ctx, *scoreUpdateValidatedPayload)
			if err != nil {
				// Log and return operational errors from the service call
				h.logger.ErrorContext(ctx, "Failed to handle ScoreUpdateValidated event during score update",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				// Return the operational error
				return nil, fmt.Errorf("failed to handle ScoreUpdateValidated event during score update: %w", err)
			}

			// Check the result from the service operation
			if result.Failure != nil {
				// Log that the score update failed based on service logic
				h.logger.InfoContext(ctx, "Participant score update failed (service logic)",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure), // Log the failure payload
				)

				// Create and return a failure message to be published
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,                               // Original message for context/metadata
					result.Failure,                    // The failure payload from the service
					roundevents.RoundScoreUpdateError, // The topic for score update failures
				)
				if errMsg != nil {
					// Log and return error if message creation fails
					h.logger.ErrorContext(ctx, "Failed to create failure message for score update error",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				// Return the created failure message
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				// Log that the participant score was updated successfully
				h.logger.InfoContext(ctx, "Participant score updated successfully in DB", attr.CorrelationIDFromMsg(msg))

				// Create a success message to publish RoundParticipantScoreUpdated
				// The Success field is expected to contain the payload for the next event
				// Assuming Success is roundevents.ParticipantScoreUpdatedPayload (struct or pointer)
				var updatedPayload roundevents.ParticipantScoreUpdatedPayload

				if updatedPayloadPtr, isPtr := result.Success.(*roundevents.ParticipantScoreUpdatedPayload); isPtr && updatedPayloadPtr != nil {
					updatedPayload = *updatedPayloadPtr // Dereference if it's a non-nil pointer
				} else if updatedPayloadVal, isVal := result.Success.(roundevents.ParticipantScoreUpdatedPayload); isVal {
					updatedPayload = updatedPayloadVal // Use the value directly
				} else {
					// Handle unexpected success payload type from UpdateParticipantScore
					h.logger.ErrorContext(ctx, "Unexpected success payload type from UpdateParticipantScore service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, fmt.Errorf("unexpected success payload type from UpdateParticipantScore service: %T", result.Success)
				}

				successMsg, err := h.helpers.CreateResultMessage(
					msg,                                      // Original message
					&updatedPayload,                          // The payload for the next event (publish pointer)
					roundevents.RoundParticipantScoreUpdated, // The topic for the next step (the check)
				)
				if err != nil {
					// Log and return error if message creation fails
					h.logger.ErrorContext(ctx, "Failed to create success message for participant score updated",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(err),
					)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// Return the created success message to be published by the router
				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, something unexpected happened
			h.logger.ErrorContext(ctx, "Unexpected result from UpdateParticipantScore service (neither success nor failure)",
				attr.CorrelationIDFromMsg(msg),
			)
			// Return a general error
			return nil, fmt.Errorf("unexpected result from service (neither success nor failure)")
		},
	)

	// Execute the wrapped handler with the incoming message and return its result
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
				attr.String("event_message_id", participantScoreUpdatedPayload.EventMessageID),
			)

			// Call the service function to handle the event (CheckAllScoresSubmitted)
			// CheckAllScoresSubmitted expects ParticipantScoreUpdatedPayload as input
			result, err := h.roundService.CheckAllScoresSubmitted(ctx, *participantScoreUpdatedPayload) // Pass the payload value
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle ParticipantScoreUpdated event during score check",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantScoreUpdated event during score check: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "All scores submitted check failed (service returned failure)",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundError, // Or a more specific failure topic for check
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				return []*message.Message{failureMsg}, nil
			}

			// Based on the check result (Success payload), decide which event to publish
			if result.Success != nil {
				// CORRECTED: Check for struct value instead of pointer
				if allScoresData, ok := result.Success.(roundevents.AllScoresSubmittedPayload); ok {
					// --- All scores submitted ---
					h.logger.InfoContext(ctx, "All scores submitted, publishing RoundAllScoresSubmitted", attr.CorrelationIDFromMsg(msg))

					// Create message for RoundAllScoresSubmitted
					// Use the data extracted as a value
					allScoresSubmittedMsg, err := h.helpers.CreateResultMessage(
						msg,
						&allScoresData,                      // Pass the address of the struct value as payload needs a pointer
						roundevents.RoundAllScoresSubmitted, // Publish this topic
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create all scores submitted message: %w", err)
					}
					return []*message.Message{allScoresSubmittedMsg}, nil

					// CORRECTED: Check for struct value instead of pointer
				} else if notAllScoresData, ok := result.Success.(roundevents.NotAllScoresSubmittedPayload); ok {
					// --- Not all scores submitted ---
					h.logger.InfoContext(ctx, "Not all scores submitted, publishing RoundNotAllScoresSubmitted", attr.CorrelationIDFromMsg(msg))

					// Create message for RoundNotAllScoresSubmitted
					// Use the data extracted as a value
					notAllScoresSubmittedMsg, err := h.helpers.CreateResultMessage(
						msg,
						&notAllScoresData,                      // Pass the address of the struct value as payload needs a pointer
						roundevents.RoundNotAllScoresSubmitted, // Publish this topic
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create not all scores submitted message: %w", err)
					}
					return []*message.Message{notAllScoresSubmittedMsg}, nil

				} else {
					// Handle unexpected success payload type from CheckAllScoresSubmitted service
					h.logger.ErrorContext(ctx, "Unexpected success payload type from CheckAllScoresSubmitted service",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("payload_type", fmt.Sprintf("%T", result.Success)),
					)
					return nil, fmt.Errorf("unexpected success payload type from service: %T", result.Success)
				}
			}

			// If neither Failure nor Success is set for checkResult (unexpected)
			h.logger.ErrorContext(ctx, "Unexpected result from CheckAllScoresSubmitted service (neither success nor failure)",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service (neither success nor failure)")
		},
	)

	return wrappedHandler(msg)
}
