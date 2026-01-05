package guildhandlers

import (
	"context"
	"errors"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRetrieveGuildConfig handles the GuildConfigRetrievalRequested event.
func (h *GuildHandlers) HandleRetrieveGuildConfig(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRetrieveGuildConfig",
		&guildevents.GuildConfigRetrievalRequestedPayloadV1{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			retrievePayload := payload.(*guildevents.GuildConfigRetrievalRequestedPayloadV1)

			h.logger.InfoContext(ctx, "Received GuildConfigRetrievalRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(retrievePayload.GuildID)),
			)

			result, err := h.guildService.GetGuildConfig(ctx, retrievePayload.GuildID)
			if err != nil {
				// Domain not-found: publish failure event, ACK (no retry)
				if errors.Is(err, guildservice.ErrGuildConfigNotFound) {
					h.logger.InfoContext(ctx, "Guild config not found (domain failure, no retry)",
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", string(retrievePayload.GuildID)),
						attr.Any("error", err),
					)
					failurePayload := result.Failure
					if failurePayload == nil { // safety net
						failurePayload = &guildevents.GuildConfigRetrievalFailedPayloadV1{ //nolint:exhaustruct
							GuildID: retrievePayload.GuildID,
							Reason:  guildservice.ErrGuildConfigNotFound.Error(),
						}
					}
					failureMsg, errMsg := h.helpers.CreateResultMessage(
						msg,
						failurePayload,
						guildevents.GuildConfigRetrievalFailedV1,
					)
					if errMsg != nil {
						return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
					}
					// Preserve guild_id metadata
					if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
						failureMsg.Metadata.Set("guild_id", guildID)
					}
					failureMsg.Metadata.Set("guild_id", string(retrievePayload.GuildID))
					return []*message.Message{failureMsg}, nil
				}

				// System / unknown error -> retry (nack)
				h.logger.ErrorContext(ctx, "Failed to handle GuildConfigRetrievalRequested event",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("error", err),
				)
				return nil, fmt.Errorf("failed to handle GuildConfigRetrievalRequested event: %w", err)
			}

			if result.Failure != nil {
				h.logger.InfoContext(ctx, "Retrieve guild config request failed",
					attr.CorrelationIDFromMsg(msg),
					attr.Any("failure_payload", result.Failure),
				)
				failureMsg, errMsg := h.helpers.CreateResultMessage(
					msg,
					result.Failure,
					guildevents.GuildConfigRetrievalFailedV1,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}

				// Preserve guild_id metadata for Discord side
				if guildID := msg.Metadata.Get("guild_id"); guildID != "" {
					failureMsg.Metadata.Set("guild_id", guildID)
				}
				// Also set it from the payload as a fallback
				if retrievePayload != nil {
					failureMsg.Metadata.Set("guild_id", string(retrievePayload.GuildID))
				}

				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Retrieve guild config request successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigRetrievedV1,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}

				// Preserve guild_id metadata for Discord side
				originalGuildID := msg.Metadata.Get("guild_id")
				h.logger.InfoContext(ctx, "Preserving guild_id metadata",
					attr.CorrelationIDFromMsg(msg),
					attr.String("original_guild_id_from_metadata", originalGuildID),
					attr.String("guild_id_from_payload", string(retrievePayload.GuildID)),
				)

				if originalGuildID != "" {
					successMsg.Metadata.Set("guild_id", originalGuildID)
					h.logger.InfoContext(ctx, "Set guild_id from original metadata",
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", originalGuildID),
					)
				}
				// Also set it from the payload as a fallback
				if retrievePayload != nil {
					successMsg.Metadata.Set("guild_id", string(retrievePayload.GuildID))
					h.logger.InfoContext(ctx, "Set guild_id from payload",
						attr.CorrelationIDFromMsg(msg),
						attr.String("guild_id", string(retrievePayload.GuildID)),
					)
				}

				// Set Discord-specific metadata for proper routing
				successMsg.Metadata.Set("domain", "discord")
				successMsg.Metadata.Set("handler_name", "HandleGuildConfigRetrieved")

				h.logger.InfoContext(ctx, "Set Discord routing metadata",
					attr.CorrelationIDFromMsg(msg),
					attr.String("domain", "discord"),
					attr.String("handler_name", "HandleGuildConfigRetrieved"),
				)

				return []*message.Message{successMsg}, nil
			}

			h.logger.ErrorContext(ctx, "Unexpected result from GetGuildConfig service",
				attr.CorrelationIDFromMsg(msg),
			)
			return nil, fmt.Errorf("unexpected result from service")
		},
	)
	return wrappedHandler(msg)
}
