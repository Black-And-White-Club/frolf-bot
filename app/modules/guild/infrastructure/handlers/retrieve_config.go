package guildhandlers

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleRetrieveGuildConfig handles the GuildConfigRetrievalRequested event.
func (h *GuildHandlers) HandleRetrieveGuildConfig(ctx context.Context, payload *guildevents.GuildConfigRetrievalRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload cannot be nil")
	}

	result, err := h.guildService.GetGuildConfig(ctx, payload.GuildID)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigRetrievalFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigRetrievedV1, Payload: result.Success},
		}, nil
	}

	return nil, errors.New("unexpected empty result from GetGuildConfig service")
}
