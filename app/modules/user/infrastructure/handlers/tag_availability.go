package userhandlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

			userID := tagAvailablePayload.UserID
			guildID := tagAvailablePayload.GuildID
			tagNumber := tagAvailablePayload.TagNumber

			h.logger.InfoContext(ctx, "Received TagAvailable event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Int("tag_number", int(tagNumber)),
			)

			ctx, span := h.tracer.Start(ctx, "CreateUserWithTag")
			defer span.End()

			result, err := h.userService.CreateUser(ctx, guildID, userID, &tagNumber, nil, nil)

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					span.RecordError(errors.New("unexpected type for failure payload"))
					return nil, errors.New("unexpected type for failure payload")
				}

				h.logger.InfoContext(ctx, "User creation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.String("reason", failedPayload.Reason),
				)

				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.UserCreationFailed,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}

				h.metrics.RecordUserCreationFailure(ctx, failedPayload.Reason, "user_already_exists")

				return []*message.Message{failureMsg}, nil
			}

			// Now check for service error after handling any failure payload
			if err != nil {
				span.RecordError(err)
				h.logger.ErrorContext(ctx, "Failed to call CreateUser service",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to create user with tag: %w", err)
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.UserCreatedPayload)
				if !ok {
					span.RecordError(errors.New("unexpected type for success payload"))
					return nil, errors.New("unexpected type for success payload")
				}

				h.logger.InfoContext(ctx, "User creation with tag succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Int("tag_number", int(tagNumber)),
				)

				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.UserCreated,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				h.metrics.RecordUserCreationSuccess(ctx, string(successPayload.UserID), "discord")

				return []*message.Message{successMsg}, nil
			}

			h.logger.WarnContext(ctx, "CreateUser returned no success or failure payload when error was nil",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			return nil, errors.New("user creation service returned unexpected result")
		},
	)

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

			h.logger.InfoContext(ctx, "Received TagUnavailable event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.Int("tag_number", int(tagNumber)),
			)

			// Ensure a default reason when none is provided
			reason := strings.TrimSpace(tagUnavailablePayload.Reason)
			if reason == "" {
				reason = "tag not available"
			}

			// Create the UserCreationFailed payload directly in the handler
			failedPayload := &userevents.UserCreationFailedPayload{
				GuildID:   tagUnavailablePayload.GuildID,
				UserID:    userID,
				TagNumber: &tagNumber,
				Reason:    reason,
			}

			// Trace message creation
			ctx, span := h.tracer.Start(ctx, "CreateResultMessage")
			defer span.End()

			// Create message to publish the UserCreationFailed event
			failedMsg, err := h.helpers.CreateResultMessage(msg, failedPayload, userevents.UserCreationFailed)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to create UserCreationFailed message: %w", err)
			}

			h.logger.InfoContext(ctx, "TagUnavailable event processed", attr.CorrelationIDFromMsg(msg))
			// Record the failure metric
			h.metrics.RecordTagAvailabilityCheck(ctx, false, tagNumber)

			// Record handler success metric

			return []*message.Message{failedMsg}, nil
		},
	)

	// Execute the wrapped handler with the message
	return wrappedHandler(msg)
}
