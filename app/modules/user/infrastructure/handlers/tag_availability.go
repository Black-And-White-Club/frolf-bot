package userhandlers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleTagAvailable handles the TagAvailable event.
func (h *UserHandlers) HandleTagAvailable(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagAvailable",
		&userevents.TagAvailablePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagAvailablePayload := payload.(*userevents.TagAvailablePayload)

			// Ensure UserID is of type sharedtypes.DiscordID
			userID := tagAvailablePayload.UserID
			tagNumber := tagAvailablePayload.TagNumber

			h.logger.Info("Received TagAvailable event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)), // Ensure this matches
				attr.Int("tag_number", int(tagNumber)),
			)

			// Call the service function to create the user
			userCreatedPayload, userCreationFailedPayload, err := h.userService.CreateUser(ctx, msg, userID, &tagNumber)
			if err != nil {
				h.logger.Error("Failed to create user",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create user: %w", err)
			}

			if userCreationFailedPayload != nil {
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					userCreationFailedPayload,
					userevents.UserCreationFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				h.metrics.RecordTagAvailabilityCheck(false, tagNumber)

				// Even if `CreateUser` returned an error, we should still return the failure message.
				return []*message.Message{failureMsg}, nil
			}

			// Create success message to publish
			successMsg, err := h.helpers.CreateResultMessage(
				msg,
				userCreatedPayload,
				userevents.UserCreated,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create success message: %w", err)
			}

			// Record the success metric
			h.metrics.RecordTagAvailabilityCheck(true, tagNumber)

			return []*message.Message{successMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}

// HandleTagUnavailable handles the TagUnavailable event.
func (h *UserHandlers) HandleTagUnavailable(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleTagUnavailable",
		&userevents.TagUnavailablePayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			tagUnavailablePayload := payload.(*userevents.TagUnavailablePayload)

			// Create convenient variables for frequently used fields
			userID := tagUnavailablePayload.UserID
			tagNumber := tagUnavailablePayload.TagNumber

			h.logger.Info("Received TagUnavailable event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Int("tag_number", int(tagNumber)),
			)

			// Create the UserCreationFailed payload directly in the handler
			failedPayload := &userevents.UserCreationFailedPayload{
				UserID:    userID,
				TagNumber: &tagNumber,
				Reason:    "tag not available",
			}

			// Trace message creation
			_, span := h.tracer.StartSpan(ctx, "CreateResultMessage", msg)
			defer span.End()

			// Create message to publish the UserCreationFailed event
			failedMsg, err := h.helpers.CreateResultMessage(msg, failedPayload, userevents.UserCreationFailed)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to create UserCreationFailed message: %w", err)
			}

			h.logger.Info("TagUnavailable event processed", attr.CorrelationIDFromMsg(msg))
			// Record the failure metric
			h.metrics.RecordTagAvailabilityCheck(false, tagNumber)

			// Record handler success metric
			h.metrics.RecordHandlerSuccess("HandleTagUnavailable")

			return []*message.Message{failedMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
