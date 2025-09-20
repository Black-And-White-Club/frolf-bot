package userhandlers

import (
	"context"
	"errors"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGetUserRequest Request handles the GetUser Request event.
func (h *UserHandlers) HandleGetUserRequest(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGetUserRequest",
		&userevents.GetUserRequestPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			getUserPayload := payload.(*userevents.GetUserRequestPayload)

			userID := getUserPayload.UserID
			guildID := getUserPayload.GuildID

			h.logger.InfoContext(ctx, "Received GetUserRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
				attr.String("guild_id", string(guildID)),
			)

			result, err := h.userService.GetUser(ctx, guildID, userID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Technical error from GetUser service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("technical error during GetUser service call: %w", err)
			}

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.GetUserFailedPayload)
				if !ok {
					h.logger.ErrorContext(ctx, "Unexpected type for failure payload from GetUser",
						attr.CorrelationIDFromMsg(msg),
					)
					return nil, errors.New("unexpected type for failure payload from GetUser")
				}

				h.logger.InfoContext(ctx, "User retrieval failed (domain failure)",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				failureMsg, createMsgErr := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.GetUserFailed,
				)
				if createMsgErr != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after service returned failure",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createMsgErr),
					)
					return nil, fmt.Errorf("failed to create failure message: %w", createMsgErr)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.GetUserResponsePayload)
				if !ok {
					h.logger.ErrorContext(ctx, "Unexpected type for success payload from GetUser",
						attr.CorrelationIDFromMsg(msg),
					)
					return nil, errors.New("unexpected type for success payload from GetUser")
				}

				h.logger.InfoContext(ctx, "User retrieval succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
				)

				successMsg, createMsgErr := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.GetUserResponse,
				)
				if createMsgErr != nil {
					h.logger.ErrorContext(ctx, "Failed to create success message after service returned success",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createMsgErr),
					)
					return nil, fmt.Errorf("failed to create success message: %w", createMsgErr)
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "GetUser service returned unexpected result: err is nil but no success or failure payload",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			return nil, errors.New("get user service returned unexpected result structure")
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
			getUserRolePayload := payload.(*userevents.GetUserRoleRequestPayload)

			userID := getUserRolePayload.UserID
			guildID := getUserRolePayload.GuildID

			h.logger.InfoContext(ctx, "Received GetUserRoleRequest event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)

			result, err := h.userService.GetUserRole(ctx, guildID, userID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Technical error from GetUserRole service call",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("technical error during GetUserRole service call: %w", err)
			}

			if result.Failure != nil {
				failedPayload, ok := result.Failure.(*userevents.GetUserRoleFailedPayload)
				if !ok {
					h.logger.ErrorContext(ctx, "Unexpected type for failure payload from GetUserRole",
						attr.CorrelationIDFromMsg(msg),
					)
					return nil, errors.New("unexpected type for failure payload from GetUserRole")
				}

				h.logger.InfoContext(ctx, "User role retrieval failed (domain failure)",
					attr.CorrelationIDFromMsg(msg),
					attr.String("reason", failedPayload.Reason),
				)

				failureMsg, createMsgErr := h.helpers.CreateResultMessage(
					msg,
					failedPayload,
					userevents.GetUserRoleFailed,
				)
				if createMsgErr != nil {
					h.logger.ErrorContext(ctx, "Failed to create failure message after service returned failure (GetUserRole)",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createMsgErr),
					)
					return nil, fmt.Errorf("failed to create failure message (GetUserRole): %w", createMsgErr)
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				successPayload, ok := result.Success.(*userevents.GetUserRoleResponsePayload)
				if !ok {
					h.logger.ErrorContext(ctx, "Unexpected type for success payload from GetUserRole",
						attr.CorrelationIDFromMsg(msg),
					)
					return nil, errors.New("unexpected type for success payload from GetUserRole")
				}

				h.logger.InfoContext(ctx, "User role retrieval succeeded",
					attr.CorrelationIDFromMsg(msg),
					attr.String("user_id", string(userID)),
					attr.String("role", string(successPayload.Role)),
				)

				successMsg, createMsgErr := h.helpers.CreateResultMessage(
					msg,
					successPayload,
					userevents.GetUserRoleResponse,
				)
				if createMsgErr != nil {
					h.logger.ErrorContext(ctx, "Failed to create success message after service returned success (GetUserRole)",
						attr.CorrelationIDFromMsg(msg),
						attr.Error(createMsgErr),
					)
					return nil, fmt.Errorf("failed to create success message (GetUserRole): %w", createMsgErr)
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "GetUserRole service returned unexpected result: err is nil but no success or failure payload",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(userID)),
			)
			return nil, errors.New("get user role service returned unexpected result structure")
		},
	)

	return wrappedHandler(msg)
}
