package guildhandlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleGuildSetup handles the guild.setup event from Discord.
// This event is published when Discord completes guild setup and contains
// all the necessary configuration data to create a guild config in the backend.
func (h *GuildHandlers) HandleGuildSetup(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleGuildSetup",
		&guildevents.GuildConfigCreationRequestedPayloadV1{}, // Using the same payload structure as create config
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			setupPayload := payload.(*guildevents.GuildConfigCreationRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received guild.setup event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(setupPayload.GuildID)),
			)

			// Convert setup payload to shared config type
			config := &guildtypes.GuildConfig{
				GuildID:              setupPayload.GuildID,
				SignupChannelID:      setupPayload.SignupChannelID,
				SignupMessageID:      setupPayload.SignupMessageID,
				EventChannelID:       setupPayload.EventChannelID,
				LeaderboardChannelID: setupPayload.LeaderboardChannelID,
				UserRoleID:           setupPayload.UserRoleID,
				EditorRoleID:         setupPayload.EditorRoleID,
				AdminRoleID:          setupPayload.AdminRoleID,
				SignupEmoji:          setupPayload.SignupEmoji,
				AutoSetupCompleted:   setupPayload.AutoSetupCompleted,
				SetupCompletedAt:     setupPayload.SetupCompletedAt,
			}

			result, err := h.guildService.CreateGuildConfig(ctx, config)
			if err != nil {
				if result.Failure == nil {
					h.logger.ErrorContext(ctx, "Failed to handle guild.setup event",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("error", err),
					)
					return nil, fmt.Errorf("failed to handle guild.setup event: %w", err)
				}
				// If we have a failure payload, we log the error as info/warn and proceed to publish the failure event
				h.logger.InfoContext(ctx, "Service returned error with failure payload",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Guild setup config creation failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					guildevents.GuildConfigCreationFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					failureMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if setupPayload != nil {
					failureMsg.Metadata.Set("guild_id", string(setupPayload.GuildID))
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Guild setup config creation successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigCreatedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					successMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if setupPayload != nil {
					successMsg.Metadata.Set("guild_id", string(setupPayload.GuildID))
				}

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from CreateGuildConfig service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)
	return wrappedHandler(msg)
}
