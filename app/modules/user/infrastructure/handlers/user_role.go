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

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleUserRoleUpdateRequest",
		&userevents.UserRoleUpdateRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			startTime := time.Now() // Keep for handler duration metric
			requestPayload := payload.(*userevents.UserRoleUpdateRequestPayload)
			userID := requestPayload.UserID
			newRole := requestPayload.Role

			h.logger.InfoContext(ctx, "Received UserRoleUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.String("role", string(newRole)),
				attr.String("requester_id", string(requestPayload.RequesterID)),
			)

			// Handler attempt metric handled by the handlerWrapper

			result, err := h.userService.UpdateUserRoleInDatabase(ctx, userID, newRole)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to call UpdateUserRoleInDatabase service",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)
				// The service wrapper handles span.RecordError for the service call
				return nil, fmt.Errorf("failed to process UserRoleUpdateRequest service call: %w", err)
			}

			var resultPayload interface{}
			var eventType string

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.UserRoleUpdateFailedPayload)
				if !ok {
					return nil, errors.New("unexpected type for failure payload from UpdateUserRoleInDatabase")
				}

				h.logger.InfoContext(ctx, "User role update failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				resultPayload = failedPayload
				eventType = userevents.UserRoleUpdateFailed

				// The service method handles RoleUpdateFailure metric
			} else if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.UserRoleUpdateResultPayload)
				if !ok {
					return nil, errors.New("unexpected type for success payload from UpdateUserRoleInDatabase")
				}

				h.logger.InfoContext(ctx, "User role updated successfully",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.String("role", string(successPayload.Role)),
				)

				resultPayload = successPayload
				eventType = userevents.UserRoleUpdated

				// The service method handles RoleUpdateSuccess metric
			} else {
				// Should not happen if service returns either Success or Failure when err is nil
				h.logger.WarnContext(ctx, "UpdateUserRoleInDatabase returned no success or failure payload when error was nil",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.String("role", string(newRole)),
				)
				return nil, errors.New("update user role service returned unexpected result")
			}

			resultMsg, createErr := h.helpers.CreateResultMessage(msg, resultPayload, eventType)
			if createErr != nil {
				h.logger.ErrorContext(ctx, "Failed to create result message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("event_type", eventType),
					attr.Error(createErr),
				)
				// The handler wrapper can handle the handler-level metric for this failure
				return nil, fmt.Errorf("failed to create result message for UserRoleUpdateRequest: %w", createErr)
			}

			// Handler success metric handled by the handlerWrapper
			// Duration metric
			h.metrics.RecordUserRetrievalDuration(ctx, userID, time.Since(startTime)) // Assuming this metric is appropriate for handler duration

			return []*message.Message{resultMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
