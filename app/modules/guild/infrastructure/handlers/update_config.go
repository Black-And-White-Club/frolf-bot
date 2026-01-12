package guildhandlers

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/handlerwrapper"
)

// HandleUpdateGuildConfig handles the GuildConfigUpdateRequested event.
func (h *GuildHandlers) HandleUpdateGuildConfig(ctx context.Context, payload *guildevents.GuildConfigUpdateRequestedPayloadV1) ([]handlerwrapper.Result, error) {
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

	result, err := h.guildService.UpdateGuildConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if result.Failure != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigUpdateFailedV1, Payload: result.Failure},
		}, nil
	}

	if result.Success != nil {
		return []handlerwrapper.Result{
			{Topic: guildevents.GuildConfigUpdatedV1, Payload: result.Success},
		}, nil
	}

	return nil, errors.New("unexpected empty result from UpdateGuildConfig service")
}
