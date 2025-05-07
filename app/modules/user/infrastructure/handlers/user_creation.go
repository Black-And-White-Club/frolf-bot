package userhandlers

import (
	"context"
	"errors"
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

			userID := userSignupPayload.UserID

			h.logger.InfoContext(ctx, "Received UserSignupRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			if userSignupPayload.TagNumber != nil {
				tagNumber := *userSignupPayload.TagNumber

				h.logger.InfoContext(ctx, "Tag availability check requested",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.Int("tag_number", int(tagNumber)),
				)

				ctx, span := h.tracer.Start(ctx, "TagAvailabilityCheck")
				defer span.End()

				eventPayload := &userevents.TagAvailabilityCheckRequestedPayload{
					TagNumber: tagNumber,
					UserID:    userID,
				}

				tagAvailabilityMsg, err := h.helpers.CreateResultMessage(
					msg,
					eventPayload,
					userevents.TagAvailabilityCheckRequested,
				)
				if err != nil {
					span.RecordError(err)
					return nil, fmt.Errorf("failed to create tag availability check message: %w", err)
				}

				h.metrics.RecordTagAvailabilityCheck(ctx, true, tagNumber)

				return []*message.Message{tagAvailabilityMsg}, nil
			}

			ctx, span := h.tracer.Start(ctx, "CreateUser")
			defer span.End()

			result, err := h.userService.CreateUser(ctx, userID, nil)

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.UserCreationFailedPayload)
				if !ok {
					span.RecordError(errors.New("unexpected type for failure payload"))
					return nil, errors.New("unexpected type for failure payload")
				}

				h.logger.InfoContext(ctx, "User creation failed",
					attr.CorrelationIDFromMsg(msg),
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

				h.metrics.RecordUserCreationFailure(ctx, failedPayload.Reason, "failed")

				return []*message.Message{failureMsg}, nil
			}

			// Now check for service error after handling any failure payload
			if err != nil {
				span.RecordError(err)
				h.logger.ErrorContext(ctx, "Failed to call CreateUser service",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to process UserSignupRequest service call: %w", err)
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.UserCreatedPayload)
				if !ok {
					span.RecordError(errors.New("unexpected type for success payload"))
					return nil, errors.New("unexpected type for success payload")
				}

				h.logger.InfoContext(ctx, "User creation succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
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
