package guildhandlers

import (
	"context"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	"github.com/ThreeDotsLabs/watermill/message"
)

// HandleRetrieveGuildConfig handles the GuildConfigRetrievalRequested event.
func (h *GuildHandlers) HandleRetrieveGuildConfig(msg *message.Message) ([]*message.Message, error) {
	wrappedHandler := h.handlerWrapper(
		"HandleRetrieveGuildConfig",
		&guildevents.GuildConfigRetrievalRequestedPayload{},
		func(ctx context.Context, msg *message.Message, payload interface{}) ([]*message.Message, error) {
			retrievePayload := payload.(*guildevents.GuildConfigRetrievalRequestedPayload)

			h.logger.InfoContext(ctx, "Received GuildConfigRetrievalRequested event",
				attr.CorrelationIDFromMsg(msg),
				attr.String("guild_id", string(retrievePayload.GuildID)),
			)

			result, err := h.guildService.GetGuildConfig(ctx, retrievePayload.GuildID)
			if err != nil {
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
					guildevents.GuildConfigRetrievalFailed,
				)
				if errMsg != nil {
					return nil, fmt.Errorf("failed to create failure message: %w", errMsg)
				}
				return []*message.Message{failureMsg}, nil
			}

			if result.Success != nil {
				h.logger.InfoContext(ctx, "Retrieve guild config request successful", attr.CorrelationIDFromMsg(msg))
				successMsg, err := h.helpers.CreateResultMessage(
					msg,
					result.Success,
					guildevents.GuildConfigRetrieved,
				)
				if err != nil {
					return nil, fmt.Errorf("failed to create success message: %w", err)
				}
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
