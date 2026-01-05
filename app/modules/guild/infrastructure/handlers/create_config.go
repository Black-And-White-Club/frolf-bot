package guildhandlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleCreateGuildConfig handles the CreateGuildConfigRequested event.
func (h *GuildHandlers) HandleCreateGuildConfig(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleCreateGuildConfig",
		&guildevents.GuildConfigCreationRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			createPayload := payload.(*guildevents.GuildConfigCreationRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received CreateGuildConfigRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(createPayload.GuildID)),
			)

			// Convert payload to shared config type
			config := &guildtypes.GuildConfig{
				GuildID:              createPayload.GuildID,
				SignupChannelID:      createPayload.SignupChannelID,
				SignupMessageID:      createPayload.SignupMessageID,
				EventChannelID:       createPayload.EventChannelID,
				LeaderboardChannelID: createPayload.LeaderboardChannelID,
				UserRoleID:           createPayload.UserRoleID,
				EditorRoleID:         createPayload.EditorRoleID,
				AdminRoleID:          createPayload.AdminRoleID,
				SignupEmoji:          createPayload.SignupEmoji,
				AutoSetupCompleted:   createPayload.AutoSetupCompleted,
				SetupCompletedAt:     createPayload.SetupCompletedAt,
				// Add more fields as needed
			}

			result, err := h.guildService.CreateGuildConfig(ctx, config)
			if err != nil && result.Failure == nil {
				h.logger.ErrorContext(ctx, "Failed to handle CreateGuildConfigRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle CreateGuildConfigRequested event: %w", err)
			}

			if result.Failure != nil {
				if err != nil {
					h.logger.WarnContext(ctx, "Create guild config request failed with domain error",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("error", err),
						attr.Any("failure_payload", result.Failure),
					)
				} else {
					h.logger.InfoContext(ctx, "Create guild config request failed",
						attr.CorrelationIDFromMsg(msg),
						attr.Any("failure_payload", result.Failure),
					)
				}
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
				if createPayload != nil {
					failureMsg.Metadata.Set("guild_id", string(createPayload.GuildID))
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Create guild config request successful", attr.CorrelationIDFromMsg(msg))
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
				if createPayload != nil {
					successMsg.Metadata.Set("guild_id", string(createPayload.GuildID))
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
