package userhandlers

import (
	"context"
	"errors"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUserRoleUpdateRequest handles the UserRoleUpdateRequest event.
func (h *UserHandlers) HandleUserRoleUpdateRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleUserRoleUpdateRequest",
		&userevents.UserRoleUpdateRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			requestPayload := payload.(*userevents.UserRoleUpdateRequestedPayloadV1)
			userID := requestPayload.UserID
			guildID := requestPayload.GuildID
			newRole := requestPayload.Role

			h.logger.InfoContext(ctx, "Received UserRoleUpdateRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.String("role", string(newRole)),
				attr.String("requester_id", string(requestPayload.RequesterID)),
			)

			// Call service method
			result, err := h.userService.UpdateUserRoleInDatabase(ctx, guildID, userID, newRole)

			// Handle each outcome with a separate CreateResultMessage call
			if result.Failure != nil {
				failurePayload, ok := result.Failure.(*userevents.UserRoleUpdateResultPayloadV1)

				if !ok {
					// Unexpected payload type - create detailed log about the actual type
					h.logger.ErrorContext(ctx, "UNEXPECTED FAILURE PAYLOAD TYPE",
						attr.CorrelationIDFromMsg(msg),
						attr.String("actual_type", fmt.Sprintf("%T", result.Failure)),
						attr.String("actual_value", fmt.Sprintf("%+v", result.Failure)),
					)

					// Try to directly convert/extract the error
					var errorStr string
					switch actual := result.Failure.(type) {
					case string:
						errorStr = actual
					case error:
						errorStr = actual.Error()
					case fmt.Stringer:
						errorStr = actual.String()
					default:
						errorStr = fmt.Sprintf("%v", actual)
					}

					// Create a new payload manually
					errorPayload := userevents.UserRoleUpdateResultPayloadV1{
						GuildID: guildID,
						Success: false,
						UserID:  userID,
						Role:    newRole, // Always include the requested role
						Reason:  fmt.Sprintf("validation error: %s", errorStr),
					}

					resultMsg, createErr := h.helpers.CreateResultMessage(
						msg,
						errorPayload,
						userevents.UserRoleUpdateFailedV1,
					)

					if createErr != nil {
						h.logger.ErrorContext(ctx, "FAILED TO CREATE ERROR MESSAGE",
							attr.CorrelationIDFromMsg(msg),
							attr.Error(createErr),
						)
						return nil, fmt.Errorf("failed to create error message: %w", createErr)
					}

					h.logger.InfoContext(ctx, "Created failure message (unexpected type)",
						attr.CorrelationIDFromMsg(msg),
						attr.String("topic", userevents.UserRoleUpdateFailedV1),
						attr.String("msg_uuid", resultMsg.UUID),
					)

					return []*message.Message{resultMsg}, nil
				}

				// Known failure payload - ensure role is set
				if failurePayload.Role == "" {
					failurePayload.Role = newRole // Always return the requested role in failure responses
				}

				// Create specific failure message
				resultMsg, createErr := h.helpers.CreateResultMessage(
					msg,
					failurePayload,
					userevents.UserRoleUpdateFailedV1,
				)

				if createErr != nil {
					h.logger.ErrorContext(ctx, "FAILED TO CREATE FAILURE MESSAGE",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createErr),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", createErr)
				}

				h.logger.InfoContext(ctx, "Created failure message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("topic", userevents.UserRoleUpdateFailedV1),
					attr.String("msg_uuid", resultMsg.UUID),
				)

				return []*message.Message{resultMsg}, nil

			} else if err != nil {
				// Technical error from service - create specific error message
				h.logger.ErrorContext(ctx, "Technical error calling service",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)

				technicalErrorPayload := userevents.UserRoleUpdateResultPayloadV1{
					GuildID: guildID,
					Success: false,
					UserID:  userID,
					Role:    newRole, // Always include the requested role
					Reason:  fmt.Sprintf("internal service error: %v", err),
				}

				resultMsg, createErr := h.helpers.CreateResultMessage(
					msg,
					technicalErrorPayload,
					userevents.UserRoleUpdateFailedV1,
				)

				if createErr != nil {
					h.logger.ErrorContext(ctx, "FAILED TO CREATE TECHNICAL ERROR MESSAGE",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createErr),
					)
					return nil, fmt.Errorf("failed to create technical error message: %w", createErr)
				}

				h.logger.InfoContext(ctx, "Created technical error message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("topic", userevents.UserRoleUpdateFailedV1),
					attr.String("msg_uuid", resultMsg.UUID),
				)

				return []*message.Message{resultMsg}, nil

			} else if result.Success != nil {
				// Success case - create success message
				successPayload, ok := result.Success.(*userevents.UserRoleUpdateResultPayloadV1)
				if !ok {
					h.logger.ErrorContext(ctx, "UNEXPECTED SUCCESS PAYLOAD TYPE",
						attr.CorrelationIDFromMsg(msg),
						attr.String("actual_type", fmt.Sprintf("%T", result.Success)),
						attr.String("actual_value", fmt.Sprintf("%+v", result.Success)),
					)
					return nil, errors.New("unexpected success payload type from service")
				}

				resultMsg, createErr := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.UserRoleUpdatedV1,
				)

				if createErr != nil {
					h.logger.ErrorContext(ctx, "FAILED TO CREATE SUCCESS MESSAGE",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createErr),
					)
					return nil, fmt.Errorf("failed to create success message: %w", createErr)
				}

				h.logger.InfoContext(ctx, "Created success message",
					attr.CorrelationIDFromMsg(msg),
					attr.String("topic", userevents.UserRoleUpdatedV1),
					attr.String("msg_uuid", resultMsg.UUID),
				)

				return []*message.Message{resultMsg}, nil

			} else {
				// Unexpected result structure - create specific error message
				h.logger.ErrorContext(ctx, "Service returned unexpected result structure",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.String("role", string(newRole)),
				)

				unexpectedResultPayload := userevents.UserRoleUpdateResultPayloadV1{
					GuildID: guildID,
					Success: false,
					UserID:  userID,
					Role:    newRole, // Always include the requested role
					Reason:  "service returned unexpected result structure",
				}

				resultMsg, createErr := h.helpers.CreateResultMessage(
					msg,
					unexpectedResultPayload,
					userevents.UserRoleUpdateFailedV1,
				)

				if createErr != nil {
					h.logger.ErrorContext(ctx, "FAILED TO CREATE MESSAGE FOR UNEXPECTED RESULT",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createErr),
					)
					return nil, fmt.Errorf("failed to create message for unexpected result structure: %w", createErr)
				}

				h.logger.InfoContext(ctx, "Created message for unexpected result",
					attr.CorrelationIDFromMsg(msg),
					attr.String("topic", userevents.UserRoleUpdateFailedV1),
					attr.String("msg_uuid", resultMsg.UUID),
				)

				return []*message.Message{resultMsg}, nil
			}
		},
	)

	msgs, err := wrappedHandler(msg)

	return msgs, err
}
