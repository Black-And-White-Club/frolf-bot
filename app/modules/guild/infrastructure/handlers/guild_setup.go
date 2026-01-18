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

	return mapOperationResult(result,
		guildevents.GuildConfigCreatedV1,
		guildevents.GuildConfigCreationFailedV1,
	), nil
}
