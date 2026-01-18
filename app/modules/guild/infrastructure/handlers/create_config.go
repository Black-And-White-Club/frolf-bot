package guildhandlers

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleCreateGuildConfig handles the CreateGuildConfigRequested event.
func (h *GuildHandlers) HandleCreateGuildConfig(ctx context.Context, payload *guildevents.GuildConfigCreationRequestedPayloadV1) ([]handlerwrapper.Result, error) {
	if payload == nil {
		return nil, errors.New("payload cannot be nil")
	}

	// Convert event payload to domain model
	config := &guildtypes.GuildConfig{
		GuildID:              payload.GuildID,
		SignupChannelID:      payload.SignupChannelID,
		SignupMessageID:      payload.SignupMessageID,
		EventChannelID:       payload.EventChannelID,
		LeaderboardChannelID: payload.LeaderboardChannelID,
		UserRoleID:           payload.UserRoleID,
		EditorRoleID:         payload.EditorRoleID,
		AdminRoleID:          payload.AdminRoleID,
		SignupEmoji:          payload.SignupEmoji,
		AutoSetupCompleted:   payload.AutoSetupCompleted,
		SetupCompletedAt:     payload.SetupCompletedAt,
	}

	result, err := h.service.CreateGuildConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return mapOperationResult(result,
		guildevents.GuildConfigCreatedV1,
		guildevents.GuildConfigCreationFailedV1,
	), nil
}
