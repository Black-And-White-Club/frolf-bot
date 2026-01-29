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

	if result.Success != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigRetrievedV1,
			Payload: guildevents.GuildConfigRetrievedPayloadV1{
				GuildID: payload.GuildID,
				Config:  **result.Success,
			},
		}}, nil
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{{
			Topic: guildevents.GuildConfigRetrievalFailedV1,
			Payload: guildevents.GuildConfigRetrievalFailedPayloadV1{
				GuildID: payload.GuildID,
				Reason:  (*result.Failure).Error(),
			},
		}}, nil
	}

	return nil, nil
}
