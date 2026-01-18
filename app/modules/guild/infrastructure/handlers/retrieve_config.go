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

	result, err := h.service.GetGuildConfig(ctx, payload.GuildID)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		guildevents.GuildConfigRetrievedV1,
		guildevents.GuildConfigRetrievalFailedV1,
	), nil
}
