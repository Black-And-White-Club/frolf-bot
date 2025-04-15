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

			h.logger.InfoContext(ctx, "Received UserRoleUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.String("role", string(newRole)),
				attr.String("requester_id", string(requestPayload.RequesterID)),
			)

			// Track operation attempt
			h.metrics.RecordHandlerAttempt(ctx, "HandleUpdateUserRole")

			// Call service function to update user role
			successPayload, failedPayload, err := h.userService.UpdateUserRoleInDatabase(ctx, userID, newRole)

			// Determine appropriate payload
			var resultPayload interface{}
			var eventType string

			if err != nil || failedPayload != nil {
				h.logger.ErrorContext(ctx, "Failed to update user role",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Error(err),
				)

				// Track failure
				h.metrics.RecordHandlerFailure(ctx, "HandleUpdateUserRole")

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
				h.logger.ErrorContext(ctx, "Failed to create result message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("event_type", eventType),
					attr.Error(createErr),
				)

				// Track the failure in metrics
				h.metrics.RecordHandlerFailure(ctx, "CreateResultMessage")

				return nil, fmt.Errorf("failed to create result message: %w", createErr)
			}

			// Track duration and success
			h.metrics.RecordHandlerSuccess(ctx, "HandleUpdateUserRole")
			h.metrics.RecordUserRetrievalDuration(ctx, userID, time.Duration(time.Since(startTime).Seconds()))

			return []*message.Message{resultMsg}, nil
		},
	)

	return wrappedHandler(msg)
}
