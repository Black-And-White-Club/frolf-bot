package userhandlers

import (
	"context"
	"fmt"

	userevents "github.com/Black-And-White-Club/frolf-bot-shared/events/user"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUpdateUDiscIdentityRequest handles UDisc identity update requests.
func (h *UserHandlers) HandleUpdateUDiscIdentityRequest(msg *message.Message) ([]*message.Message, error) {
	return h.handlerWrapper(
		"HandleUpdateUDiscIdentityRequest",
		&userevents.UpdateUDiscIdentityRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatePayload := payload.(*userevents.UpdateUDiscIdentityRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received UpdateUDiscIdentity request",
				attr.CorrelationIDFromMsg(msg),
				attr.String("user_id", string(updatePayload.UserID)),
				attr.String("guild_id", string(updatePayload.GuildID)),
			)

			result, err := h.userService.UpdateUDiscIdentity(
				ctx,
				updatePayload.GuildID,
				updatePayload.UserID,
				updatePayload.Username,
				updatePayload.Name,
			)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to update UDisc identity",
					attr.CorrelationIDFromMsg(msg),
					attr.Error(err),
				)
				return nil, fmt.Errorf("failed to update UDisc identity: %w", err)
			}

			if result.Failure != nil {
				failureMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					userevents.UDiscIdentityUpdateFailedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", err)
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					userevents.UDiscIdentityUpdatedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				return []*message.Message{successMsg}, nil
			}

			return nil, fmt.Errorf("unexpected result from UpdateUDiscIdentity")
		},
	)(msg)
}
