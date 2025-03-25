package userhandlers

import (
	"context"
	"fmt"
	"time"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetUser Request handles the GetUser Request event.
func (h *UserHandlers) HandleGetUserRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetUserRequest",
		&userevents.GetUserRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			getUserPayload := payload.(*userevents.GetUserRequestPayload)

			// Create convenient variables for frequently used fields
			userID := getUserPayload.UserID

			h.logger.Info("Received GetUserRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			successPayload, failedPayload, err := h.userService.GetUser(ctx, msg, userID)
			if err != nil {
				h.logger.Error("Failed to get user",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process GetUserRequest: %w", err)
			}

			if failedPayload != nil {
				// Log user retrieval failure
				h.logger.Info("User retrieval failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				// Create failure message to publish
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.GetUserFailed,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}

				return []*message.Message{failureMsg}, nil
			}

			// Log user retrieval success
			h.logger.Info("User retrieval succeeded",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			// Create success message to publish
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				successPayload,
				userevents.GetUserResponse,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			return []*message.Message{successMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleGetUserRoleRequest handles the GetUserRoleRequest event.
func (h *UserHandlers) HandleGetUserRoleRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetUserRoleRequest",
		&userevents.GetUserRoleRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			startTime := time.Now()
			requestPayload := payload.(*userevents.GetUserRoleRequestPayload)
			userID := requestPayload.UserID

			h.logger.Info("Received GetUserRoleRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			// Track operation attempt
			h.metrics.RecordOperationAttempt("GetUserRole", userID)

			// Trace user role retrieval
			ctx, span := h.tracer.StartSpan(ctx, "GetUserRole", msg)
			defer span.End()

			// Retrieve user role from service
			successPayload, failedPayload, err := h.userService.GetUserRole(ctx, msg, userID)
			if err != nil {
				span.RecordError(err)
				h.logger.Error("Failed to get user role",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)

				// Track failure
				h.metrics.RecordOperationFailure("GetUserRole", userID)
				h.metrics.RecordUserRoleRetrieval(false, userID)

				// If failedPayload is not nil, use it
				var failurePayload *userevents.GetUserRoleFailedPayload
				if failedPayload != nil {
					failurePayload = failedPayload
				} else {
					// Otherwise, create a new failure payload
					failurePayload = &userevents.GetUserRoleFailedPayload{
						UserID: userID,
						Reason: err.Error(),
					}
				}

				// Return failure message
				failureMsg, err := h.helpers.CreateResultMessage(msg, failurePayload, userevents.GetUserRoleFailed)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create GetUserRoleFailed message: %w", err)
				}

				return []*message.Message{failureMsg}, nil
			}

			// Return success message
			successMsg, err := h.helpers.CreateResultMessage(msg, successPayload, userevents.GetUserRoleResponse)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to create GetUserRoleResponse message: %w", err)
			}

			// Track success
			h.metrics.RecordOperationSuccess("GetUserRole", userID)
			h.metrics.RecordUserRoleRetrieval(true, userID)

			// Track duration
			h.metrics.RecordUserRetrievalDuration(time.Since(startTime).Seconds())

			return []*message.Message{successMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
