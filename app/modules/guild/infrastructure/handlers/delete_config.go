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

	result, err := h.service.DeleteGuildConfig(ctx, sharedtypes.GuildID(payload.GuildID))
	if err != nil {
		return nil, err
	}

	if result.Success != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigDeletedV1,
			Payload: guildevents.GuildConfigDeletedPayloadV1{
				GuildID:       sharedtypes.GuildID(payload.GuildID),
				ResourceState: (*result.Success).ResourceState,
			},
		}}, nil
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigDeletionFailedV1,
			Payload: guildevents.GuildConfigDeletionFailedPayloadV1{
				GuildID: sharedtypes.GuildID(payload.GuildID),
				Reason:  (*result.Failure).Error(),
			},
		}}, nil
	}

	return nil, nil
}
