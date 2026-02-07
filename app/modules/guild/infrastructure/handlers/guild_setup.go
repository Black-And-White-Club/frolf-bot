package guildhandlers

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleGuildSetup handles the guild setup event from Discord.
// This event is published when Discord completes guild setup and contains
// all the necessary configuration data to create a guild config in the backend.
func (h *GuildHandlers) HandleGuildSetup(ctx context.Context, payload *guildtypes.GuildConfig) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload cannot be nil")
	}

	result, err := h.service.CreateGuildConfig(ctx, payload)
	if err != nil {
		return nil, err
	}

	if result.Success != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigCreatedV1,
			Payload: guildevents.GuildConfigCreatedPayloadV1{
				GuildID: payload.GuildID,
				Config:  **result.Success,
			},
		}}, nil
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigCreationFailedV1,
			Payload: guildevents.GuildConfigCreationFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  (*result.Failure).Error(),
			},
		}}, nil
	}

	return nil, nil
}
