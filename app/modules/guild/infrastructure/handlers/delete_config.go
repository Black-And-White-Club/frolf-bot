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
		&guildevents.GuildConfigDeletionRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			deletePayload := payload.(*guildevents.GuildConfigDeletionRequestedPayloadV1)

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
					guildevents.GuildConfigDeletionFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					failureMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if deletePayload != nil {
					failureMsg.Metadata.Set("guild_id", string(deletePayload.GuildID))
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Delete guild config request successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigDeletedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					successMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if deletePayload != nil {
					successMsg.Metadata.Set("guild_id", string(deletePayload.GuildID))
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
