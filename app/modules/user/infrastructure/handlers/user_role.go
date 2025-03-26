package userhandlers

import (
	"context"
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
			startTime := time.Now()
			requestPayload := payload.(*userevents.UserRoleUpdateRequestPayload)
			userID := requestPayload.UserID
			newRole := requestPayload.Role

			h.logger.Info("Received UserRoleUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.String("role", string(newRole)),
				attr.String("requester_id", string(requestPayload.RequesterID)),
			)

			// Track operation attempt
			h.metrics.RecordOperationAttempt("UpdateUserRole", userID)

			// Call service function to update user role
			successPayload, failedPayload, err := h.userService.UpdateUserRoleInDatabase(ctx, msg, userID, newRole)

			// Determine appropriate payload
			var resultPayload interface{}
			var eventType string

			if err != nil || failedPayload != nil {
				h.logger.Error("Failed to update user role",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)

				// Track failure
				h.metrics.RecordOperationFailure("UpdateUserRole", userID)

				// Prefer explicit failed payload, or create a generic one
				if failedPayload != nil {
					resultPayload = failedPayload
					eventType = userevents.UserRoleUpdateFailed
				} else {
					resultPayload = &userevents.UserRoleUpdateFailedPayload{
						UserID: userID,
						Reason: err.Error(),
					}
					eventType = userevents.UserRoleUpdateFailed
				}
			} else {
				// Success scenario
				resultPayload = successPayload
				eventType = userevents.UserRoleUpdated
			}

			// Create result message
			resultMsg, createErr := h.helpers.CreateResultMessage(msg, resultPayload, eventType)
			if createErr != nil {
				// Log the failure to create the result message
				h.logger.Error("Failed to create result message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("event_type", eventType),
					attr.Error(createErr),
				)

				// Track the failure in metrics
				h.metrics.RecordOperationFailure("CreateResultMessage", userID)

				return nil, fmt.Errorf("failed to create result message: %w", createErr)
			}

			// Track duration and success
			h.metrics.RecordOperationSuccess("UpdateUserRole", userID)
			h.metrics.RecordUserRetrievalDuration(time.Since(startTime).Seconds())

			return []*message.Message{resultMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
