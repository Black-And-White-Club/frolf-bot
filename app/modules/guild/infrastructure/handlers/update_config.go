package guildhandlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleUpdateGuildConfig handles the GuildConfigUpdateRequested event.
func (h *GuildHandlers) HandleUpdateGuildConfig(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleUpdateGuildConfig",
		&guildevents.GuildConfigUpdateRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			updatePayload := payload.(*guildevents.GuildConfigUpdateRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received GuildConfigUpdateRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(updatePayload.GuildID)),
			)

			// Convert payload to shared config type
			config := &guildtypes.GuildConfig{
				GuildID:              updatePayload.GuildID,
				SignupChannelID:      updatePayload.SignupChannelID,
				SignupMessageID:      updatePayload.SignupMessageID,
				EventChannelID:       updatePayload.EventChannelID,
				LeaderboardChannelID: updatePayload.LeaderboardChannelID,
				UserRoleID:           updatePayload.UserRoleID,
				EditorRoleID:         updatePayload.EditorRoleID,
				AdminRoleID:          updatePayload.AdminRoleID,
				SignupEmoji:          updatePayload.SignupEmoji,
				AutoSetupCompleted:   updatePayload.AutoSetupCompleted,
				SetupCompletedAt:     updatePayload.SetupCompletedAt,
				// Add more fields as needed
			}

			result, err := h.guildService.UpdateGuildConfig(ctx, config)
			if err != nil {
				h.logger.ErrorContext(ctx, "Failed to handle GuildConfigUpdateRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle GuildConfigUpdateRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Update guild config request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					guildevents.GuildConfigUpdateFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					failureMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if updatePayload != nil {
					failureMsg.Metadata.Set("guild_id", string(updatePayload.GuildID))
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Update guild config request successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigUpdatedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					successMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if updatePayload != nil {
					successMsg.Metadata.Set("guild_id", string(updatePayload.GuildID))
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from UpdateGuildConfig service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)
	return wrappedHandler(msg)
}
