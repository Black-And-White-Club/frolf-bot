package guildhandlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleDeleteGuildConfig handles the GuildConfigDeletionRequested event.
func (h *GuildHandlers) HandleDeleteGuildConfig(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleDeleteGuildConfig",
		&guildevents.GuildConfigDeletionRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			deletePayload := payload.(*guildevents.GuildConfigDeletionRequestedPayload)

			h.logger.InfoContext(ctx, "Received GuildConfigDeletionRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(deletePayload.GuildID)),
			)

			result, err := h.guildService.DeleteGuildConfig(ctx, deletePayload.GuildID)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle GuildConfigDeletionRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle GuildConfigDeletionRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Delete guild config request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					guildevents.GuildConfigDeletionFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Delete guild config request successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigDeleted,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from DeleteGuildConfig service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)
	return wrappedHandler(msg)
}
