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

	// Convert payload to shared config type
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

	result, err := h.guildService.CreateGuildConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigCreationFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigCreatedV1, Payload: result.Success},
		}, nil
	}

	return nil, errors.New("unexpected empty result from CreateGuildConfig service")
}
