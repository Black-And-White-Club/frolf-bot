package roundhandlers

import (
	"context"
	"fmt"

	roundevents "github.com/Black-And-White-Club/frolf-bot-shared/events/round"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func (h *RoundHandlers) HandleRoundDeleteRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"OnRoundDeleteRequested",
		&roundevents.RoundDeleteRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundDeleteRequestPayload := payload.(*roundevents.RoundDeleteRequestPayload)

			// Check for nil/zero UUID before proceeding
			if roundDeleteRequestPayload.RoundID == sharedtypes.RoundID(uuid.Nil) {
				h.logger.ErrorContext(ctx, "Round delete request has nil UUID",
					attr.CorrelationIDFromMsg(msg),
				)
				return nil, fmt.Errorf("invalid round ID: cannot process delete request with nil UUID")
			}

			h.logger.InfoContext(ctx, "Received RoundDeleteRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("round_id", roundDeleteRequestPayload.RoundID.String()),
				attr.String("requesting_user", string(roundDeleteRequestPayload.RequestingUserUserID)),
			)

			// First validate the request format
			validateResult, err := h.roundService.ValidateRoundDeleteRequest(ctx, *roundDeleteRequestPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to validate RoundDeleteRequest",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to validate RoundDeleteRequest: %w", err)
			}

			if validateResult.Failure != nil {
				h.logger.InfoContext(ctx, "Round delete request validation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", validateResult.Failure),
				)

				// Create failure message for validation failure
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					validateResult.Failure,
					roundevents.RoundDeleteError,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create validation failure message: %w", errMsg)
				}

				return []*message.Message{failureMsg}, nil
			}

			// If validation succeeded, the service should handle everything else
			if validateResult.Success != nil {
				h.logger.InfoContext(ctx, "Round delete request validated successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.String("round_id", roundDeleteRequestPayload.RoundID.String()),
				)

				// Create success message with the validated payload (dereference the pointer)
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					*validateResult.Success.(*roundevents.RoundDeleteValidatedPayload),
					roundevents.RoundDeleteValidated,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			// If neither Failure nor Success is set, return an error
			h.logger.ErrorContext(ctx, "Unexpected result from round delete validation and fetch",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundDeleteValidated(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundDeleteValidated",
		&roundevents.RoundDeleteValidatedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundDeleteValidatedPayload := payload.(*roundevents.RoundDeleteValidatedPayload)

			h.logger.InfoContext(ctx, "Received RoundDeleteValidated event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundDeleteValidatedPayload.RoundDeleteRequestPayload.RoundID),
			)

			// Convert validated payload to authorized payload
			authorizedPayload := &roundevents.RoundDeleteAuthorizedPayload{
				GuildID: roundDeleteValidatedPayload.RoundDeleteRequestPayload.GuildID,
				RoundID: roundDeleteValidatedPayload.RoundDeleteRequestPayload.RoundID,
			}

			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				authorizedPayload,
				roundevents.RoundDeleteAuthorized,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to create RoundDeleteAuthorized message",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundDeleteValidatedPayload.RoundDeleteRequestPayload.RoundID),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create RoundDeleteAuthorized message: %w", err)
			}

			return []*message.Message{successMsg}, nil
		},
	)

	return wrappedHandler(msg)
}

func (h *RoundHandlers) HandleRoundDeleteAuthorized(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRoundDeleteAuthorized",
		&roundevents.RoundDeleteAuthorizedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			roundDeleteAuthorizedPayload := payload.(*roundevents.RoundDeleteAuthorizedPayload)

			h.logger.InfoContext(ctx, "Received RoundDeleteAuthorized event",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
			)

			result, err := h.roundService.DeleteRound(ctx, *roundDeleteAuthorizedPayload)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to execute RoundService.DeleteRound for RoundDeleteAuthorized event",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
					attr.Any("service_call_error", err),
				)
				// Return the error, which will cause the message to be nacked/retried.
				return nil, fmt.Errorf("RoundService.DeleteRound failed: %w", err)
			}

			// Check the result returned by the service for business logic success or failure
			if result.Failure != nil {
				h.logger.InfoContext(ctx, "RoundService.DeleteRound returned failure result",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
					attr.Any("service_failure_payload", result.Failure),
				)

				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					roundevents.RoundDeleteError,
				)
				if errMsg != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after RoundService.DeleteRound failure",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
						attr.Error(errMsg),
					)
					return nil, fmt.Errorf("RoundService.DeleteRound failed and failed to create failure message: %w", errMsg)
				}

				// ✅ FIX: Return nil error for business logic failures
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "RoundService.DeleteRound successful",
					attr.CorrelationIDFromMsg(msg),
					attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
				)

				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					roundevents.RoundDeleted,
				)
				if err != nil {
					h.logger.ErrorContext(ctx, "Failed to create RoundDeleted success message after service success",
						attr.CorrelationIDFromMsg(msg),
						attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
						attr.Error(err),
					)
					return nil, fmt.Errorf("failed to create RoundDeleted success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from RoundService.DeleteRound - neither Success nor Failure is set",
				attr.CorrelationIDFromMsg(msg),
				attr.RoundID("round_id", roundDeleteAuthorizedPayload.RoundID),
			)
			return nil, fmt.Errorf("unexpected result from RoundService.DeleteRound for round %s", roundDeleteAuthorizedPayload.RoundID.String())
		},
	)

	return wrappedHandler(msg)
}
