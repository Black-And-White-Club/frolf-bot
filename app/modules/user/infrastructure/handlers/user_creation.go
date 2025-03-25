package userhandlers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserSignupRequest handles the UserSignupRequest event.
func (h *UserHandlers) HandleUserSignupRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleUserSignupRequest",
		&userevents.UserSignupRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			userSignupPayload := payload.(*userevents.UserSignupRequestPayload)

			// Create convenient variables for frequently used fields
			userID := userSignupPayload.UserID

			h.logger.Info("Received UserSignupRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			// If a tag is provided, check its availability
			if userSignupPayload.TagNumber != nil {
				tagNumber := *userSignupPayload.TagNumber

				h.logger.Info("Tag availability check requested",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Int("tag_number", tagNumber),
				)

				// Trace tag availability check
				_, span := h.tracer.StartSpan(ctx, "TagAvailabilityCheck", msg)
				defer span.End()

				// Prepare the event payload for the tag availability check request
				eventPayload := &userevents.TagAvailabilityCheckRequestedPayload{
					TagNumber: tagNumber,
					UserID:    userID,
				}

				// Create a new message for the tag availability check
				tagAvailabilityMsg, err := h.helpers.CreateResultMessage(
					msg,
					eventPayload,
					userevents.TagAvailabilityCheckRequested,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create tag availability check message: %w", err)
				}

				// Record metrics for tag availability check
				h.metrics.RecordTagAvailabilityCheck(true, tagNumber)

				// Return the tag availability check message
				return []*message.Message{tagAvailabilityMsg}, nil
			}

			// If no tag is provided, proceed with user creation
			ctx, span := h.tracer.StartSpan(ctx, "CreateUser", msg)
			defer span.End()

			successPayload, failedPayload, err := h.userService.CreateUser(ctx, msg, userID, nil)
			if err != nil {
				span.RecordError(err)
				h.logger.Error("Failed to create user",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process UserSignupRequest: %w", err)
			}

			if failedPayload != nil {
				// Log user creation failure
				h.logger.Info("User creation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				// Create failure message to publish
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.UserCreationFailed,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}

				// Record metrics for user creation failure
				h.metrics.RecordUserCreation(failedPayload.Reason, "user", "failed")

				return []*message.Message{failureMsg}, nil
			}

			// Log user creation success
			h.logger.Info("User creation succeeded",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			// Create success message to publish
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				successPayload,
				userevents.UserCreated,
			)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			// Record metrics for successful user creation
			h.metrics.RecordUserCreation("standard", "user", "success")

			return []*message.Message{successMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
