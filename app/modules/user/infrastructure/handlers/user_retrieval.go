package userhandlers

import (
	"context"
	"errors"
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

			userID := getUserPayload.UserID

			h.logger.InfoContext(ctx, "Received GetUserRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			result, err := h.userService.GetUser(ctx, userID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to call GetUser service",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process GetUserRequest service call: %w", err)
			}

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.GetUserFailedPayload)
				if !ok {
					return nil, errors.New("unexpected type for failure payload from GetUser")
				}

				h.logger.InfoContext(ctx, "User retrieval failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

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

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.GetUserResponsePayload)
				if !ok {
					return nil, errors.New("unexpected type for success payload from GetUser")
				}

				h.logger.InfoContext(ctx, "User retrieval succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
				)

				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.GetUserResponse,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.WarnContext(ctx, "GetUser returned no success or failure payload when error was nil",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			return nil, errors.New("get user service returned unexpected result")
		},
	)

	return wrappedHandler(msg)
}

// HandleGetUserRoleRequest handles the GetUserRoleRequest event.
func (h *UserHandlers) HandleGetUserRoleRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetUserRoleRequest",
		&userevents.GetUserRoleRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			startTime := time.Now() // Keep for handler duration metric
			requestPayload := payload.(*userevents.GetUserRoleRequestPayload)
			userID := requestPayload.UserID

			h.logger.InfoContext(ctx, "Received GetUserRoleRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			result, err := h.userService.GetUserRole(ctx, userID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to call GetUserRole service",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)
				// The service wrapper handles span.RecordError for the service call
				return nil, fmt.Errorf("failed to process GetUserRoleRequest service call: %w", err)
			}

			var resultMsg *message.Message
			var createErr error

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.GetUserRoleFailedPayload)
				if !ok {
					return nil, errors.New("unexpected type for failure payload from GetUserRole")
				}

				h.logger.InfoContext(ctx, "User role retrieval failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				resultMsg, createErr = h.helpers.CreateResultMessage(msg, failedPayload, userevents.GetUserRoleFailed)

				// The service method handles UserRoleRetrievalFailure metric
			} else if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayload)
				if !ok {
					return nil, errors.New("unexpected type for success payload from GetUserRole")
				}

				h.logger.InfoContext(ctx, "User role retrieval succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
				)

				resultMsg, createErr = h.helpers.CreateResultMessage(msg, successPayload, userevents.GetUserRoleResponse)

				// The service method handles UserRetrievalSuccess metric
			} else {
				// Should not happen if service returns either Success or Failure when err is nil
				h.logger.WarnContext(ctx, "GetUserRole returned no success or failure payload when error was nil",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
				)
				return nil, errors.New("get user role service returned unexpected result")
			}

			if createErr != nil {
				h.logger.ErrorContext(ctx, "Failed to create result message",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(createErr),
				)
				// The handler wrapper can handle the handler-level metric for this failure
				return nil, fmt.Errorf("failed to create result message for GetUserRole: %w", createErr)
			}

			// Track handler duration
			h.metrics.RecordUserRetrievalDuration(ctx, userID, time.Since(startTime)) // Use time.Since directly

			return []*message.Message{resultMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
