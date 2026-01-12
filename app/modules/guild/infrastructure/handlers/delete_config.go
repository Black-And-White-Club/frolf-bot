package guildhandlers

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleDeleteGuildConfig handles the GuildConfigDeletionRequested event.
func (h *GuildHandlers) HandleDeleteGuildConfig(ctx context.Context, payload *guildevents.GuildConfigDeletionRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload cannot be nil")
	}

	result, err := h.guildService.DeleteGuildConfig(ctx, sharedtypes.GuildID(payload.GuildID))
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigDeletionFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigDeletedV1, Payload: result.Success},
		}, nil
	}

	return nil, errors.New("unexpected empty result from DeleteGuildConfig service")
}
