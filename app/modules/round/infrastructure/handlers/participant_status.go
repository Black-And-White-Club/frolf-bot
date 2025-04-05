package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (h *RoundHandlers) HandleParticipantJoinRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantJoinRequest",
		&roundevents.ParticipantJoinRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantJoinRequestPayload := payload.(*roundevents.ParticipantJoinRequestPayload)

			h.logger.Info("Received ParticipantJoinRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantJoinRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantJoinRequestPayload.UserID)),
				attr.String("response", string(participantJoinRequestPayload.Response)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.CheckParticipantStatus(ctx, *participantJoinRequestPayload)
			if err != nil {
				h.logger.Error("Failed to handle ParticipantJoinRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantJoinRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Participant join request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantStatusCheckError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.Info("Participant join request validated", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundParticipantJoinValidationRequest,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from CheckParticipantStatus service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleParticipantJoinValidationRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantJoinValidationRequest",
		&roundevents.ParticipantJoinValidationRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantJoinValidationRequestPayload := payload.(*roundevents.ParticipantJoinValidationRequestPayload)

			h.logger.Info("Received ParticipantJoinValidationRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantJoinValidationRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantJoinValidationRequestPayload.UserID)),
				attr.String("response", string(participantJoinValidationRequestPayload.Response)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ValidateParticipantJoinRequest(ctx, roundevents.ParticipantJoinRequestPayload{
				RoundID:  participantJoinValidationRequestPayload.RoundID,
				UserID:   participantJoinValidationRequestPayload.UserID,
				Response: participantJoinValidationRequestPayload.Response,
			})
			if err != nil {
				h.logger.Error("Failed to handle ParticipantJoinValidationRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantJoinValidationRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Participant join validation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantJoinError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.Info("Participant join validation successful", attr.CorrelationIDFromMsg(msg))

				// Check if the response is Declined
				if participantJoinValidationRequestPayload.Response == roundtypes.ResponseDecline {
					// Create success message to publish
					updateRequest := result.Success.(*roundevents.ParticipantJoinRequestPayload)
					successMsg, err := h.helpers.CreateResultMessage(
						msg,
						updateRequest,
						roundevents.RoundParticipantStatusUpdateRequest,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create success message: %w", err)
					}

					return []*message.Message{successMsg}, nil
				} else {
					// Create success message to publish
					updateRequest := result.Success.(*roundevents.ParticipantJoinRequestPayload)
					tagLookupRequest := roundevents.TagLookupRequestPayload{
						UserID:     updateRequest.UserID,
						RoundID:    updateRequest.RoundID,
						Response:   updateRequest.Response,
						JoinedLate: updateRequest.JoinedLate,
					}
					successMsg, err := h.helpers.CreateResultMessage(
						msg,
						tagLookupRequest,
						roundevents.LeaderboardGetTagNumberRequest,
					)
					if err != nil {
						return nil, fmt.Errorf("failed to create success message: %w", err)
					}

					return []*message.Message{successMsg}, nil
				}
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from ValidateParticipantJoinRequest service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleParticipantRemovalRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantRemovalRequest",
		&roundevents.ParticipantRemovalRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantRemovalRequestPayload := payload.(*roundevents.ParticipantRemovalRequestPayload)

			h.logger.Info("Received ParticipantRemovalRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantRemovalRequestPayload.RoundID.String()),
				attr.String("user_id", string(participantRemovalRequestPayload.UserID)),
			)

			// Call the service function to handle the event
			result, err := h.roundService.ParticipantRemoval(ctx, *participantRemovalRequestPayload)
			if err != nil {
				h.logger.Error("Failed to handle ParticipantRemovalRequest event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle ParticipantRemovalRequest event: %w", err)
			}

			if result.Failure != nil {
				h.logger.Info("Participant removal request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)

				// Create failure message
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundParticipantRemovalError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.Info("Participant removal request successful", attr.CorrelationIDFromMsg(msg))

				// Create success message to publish
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundParticipantRemoved,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.Error("Unexpected result from ParticipantRemoval service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleTagNumberFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberFound",
		&roundevents.RoundTagNumberFoundPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagNumberFoundPayload := payload.(*roundevents.RoundTagNumberFoundPayload)

			// Log the received event
			h.logger.Info("Received TagNumberFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagNumberFoundPayload.RoundID.String()),
				attr.String("user_id", string(tagNumberFoundPayload.UserID)),
				attr.Int("tag_number", int(*tagNumberFoundPayload.TagNumber)),
			)

			// Call the helper method to update participant status
			return h.handleParticipantUpdate(ctx, msg, tagNumberFoundPayload, roundtypes.ResponseAccept)
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleTagNumberNotFound(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagNumberNotFound",
		&roundevents.RoundTagNumberNotFoundPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagNumberNotFoundPayload := payload.(*roundevents.RoundTagNumberNotFoundPayload)

			// Log the received event
			h.logger.Info("Received TagNumberNotFound event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", tagNumberNotFoundPayload.RoundID.String()),
				attr.String("user_id", string(tagNumberNotFoundPayload.UserID)),
			)

			// Call the helper method to update participant status without a tag number
			return h.handleParticipantUpdate(ctx, msg, tagNumberNotFoundPayload, roundtypes.ResponseAccept)
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleParticipantDeclined(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleParticipantDeclined",
		&roundevents.ParticipantDeclinedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			participantDeclinedPayload := payload.(*roundevents.ParticipantDeclinedPayload)

			// Log the received event
			h.logger.Info("Received ParticipantDeclined event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", participantDeclinedPayload.RoundID.String()),
				attr.String("user_id", string(participantDeclinedPayload.UserID)),
			)

			// Call the helper method to update participant status for decline
			return h.handleParticipantUpdate(ctx, msg, participantDeclinedPayload, roundtypes.ResponseDecline)
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// In the shared function:
func (h *RoundHandlers) handleParticipantUpdate(ctx context.Context, msg *message.Message, payload roundevents.ParticipantUpdatePayload, response roundtypes.Response) ([]*message.Message, error) {
	roundID := payload.GetRoundID()
	userID := payload.GetUserID()
	tagNumber := payload.GetTagNumber()

	// Log the received event
	h.logger.Info("Processing participant update",
		attr.CorrelationIDFromMsg(msg),
		attr.String("round_id", roundID.String()),
		attr.String("user_id", string(userID)),
		attr.String("response", string(response)),
	)

	// Check if the tag number is nil
	if tagNumber != nil {
		h.logger.Info("Tag number is not nil",
			attr.CorrelationIDFromMsg(msg),
			attr.Int("tag_number", int(*tagNumber)),
		)
	} else {
		h.logger.Info("Tag number is nil",
			attr.CorrelationIDFromMsg(msg),
		)
	}

	// Perform DB update
	result, err := h.roundService.UpdateParticipantStatus(ctx, roundevents.ParticipantJoinRequestPayload{
		RoundID:   roundID,
		UserID:    userID,
		Response:  response,
		TagNumber: tagNumber,
	})
	if err != nil {
		h.logger.Error("Failed to update participant status",
			attr.CorrelationIDFromMsg(msg),
			attr.Any("error", err),
		)
		return nil, fmt.Errorf("failed to update participant status: %w", err)
	}

	// Check if the result is empty
	if result.Success == nil {
		return nil, fmt.Errorf("unexpected result from service")
	}

	// Create success message
	successMsg, err := h.helpers.CreateResultMessage(msg, result, roundevents.RoundParticipantJoined)
	if err != nil {
		return nil, fmt.Errorf("failed to create success message: %w", err)
	}

	return []*message.Message{successMsg}, nil
}
