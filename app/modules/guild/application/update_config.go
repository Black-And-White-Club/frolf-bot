package guildservice

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
)

// UpdateGuildConfig updates an existing guild configuration.
func (s *GuildService) UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildOperationResult, error) {
	if ctx == nil {
		return GuildOperationResult{
			Error: errors.New("context cannot be nil"),
		}, errors.New("context cannot be nil")
	}
	if config == nil {
		return GuildOperationResult{
			Error: errors.New("config cannot be nil"),
		}, errors.New("config cannot be nil")
	}
	guildID := config.GuildID
	if guildID == "" {
		return GuildOperationResult{
			Error: errors.New("invalid guild ID"),
		}, errors.New("invalid guild ID")
	}

	return s.serviceWrapper(ctx, "UpdateGuildConfig", guildID, func(ctx context.Context) (GuildOperationResult, error) {
		existing, err := s.GuildDB.GetConfig(ctx, guildID)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: guildID,
					Reason:  "could not fetch existing config: " + err.Error(),
				},
				Error: err,
			}, err
		}
		if existing == nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: guildID,
					Reason:  "guild config not found",
				},
				Error: errors.New("guild config not found"),
			}, errors.New("guild config not found")
		}

		// Build updates map from config
		updates := map[string]interface{}{
			"SignupChannelID":      config.SignupChannelID,
			"SignupMessageID":      config.SignupMessageID,
			"EventChannelID":       config.EventChannelID,
			"LeaderboardChannelID": config.LeaderboardChannelID,
			"UserRoleID":           config.UserRoleID,
			"EditorRoleID":         config.EditorRoleID,
			"AdminRoleID":          config.AdminRoleID,
			"SignupEmoji":          config.SignupEmoji,
			"AutoSetupCompleted":   config.AutoSetupCompleted,
			"SetupCompletedAt":     config.SetupCompletedAt,
			// Add more fields as needed
		}

		err = s.GuildDB.UpdateConfig(ctx, guildID, updates)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigUpdateFailedPayload{
					GuildID: guildID,
					Reason:  err.Error(),
				},
				Error: err,
			}, err
		}

		return GuildOperationResult{
			Success: &guildevents.GuildConfigUpdatedPayload{
				GuildID: guildID,
				Config:  *config,
			},
		}, nil
	})
}
