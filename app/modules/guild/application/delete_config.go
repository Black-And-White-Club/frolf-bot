package guildservice

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// DeleteGuildConfig performs a soft delete on a guild configuration.
func (s *GuildService) DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildOperationResult, error) {
	return s.serviceWrapper(ctx, "DeleteGuildConfig", guildID, func(ctx context.Context) (GuildOperationResult, error) {
		if ctx == nil {
			return GuildOperationResult{
				Error: errors.New("context cannot be nil"),
			}, errors.New("context cannot be nil")
		}
		if guildID == "" {
			return GuildOperationResult{
				Error: errors.New("invalid guild ID"),
			}, errors.New("invalid guild ID")
		}

		// Try to get the config first
		existing, err := s.GuildDB.GetConfig(ctx, guildID)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigDeletionFailedPayload{
					GuildID: guildID,
					Reason:  err.Error(),
				},
				Error: err,
			}, err
		}
		if existing == nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigDeletionFailedPayload{
					GuildID: guildID,
					Reason:  "guild config not found",
				},
				Error: errors.New("guild config not found"),
			}, errors.New("guild config not found")
		}

		// Build resource snapshot for the event from the existing config.
		rs := guildtypes.ResourceState{
			SignupChannelID:      existing.SignupChannelID,
			SignupMessageID:      existing.SignupMessageID,
			EventChannelID:       existing.EventChannelID,
			LeaderboardChannelID: existing.LeaderboardChannelID,
			UserRoleID:           existing.UserRoleID,
			EditorRoleID:         existing.EditorRoleID,
			AdminRoleID:          existing.AdminRoleID,
			Results:              map[string]guildtypes.DeletionResult{},
		}

		// Soft delete: set IsActive = false and snapshot resource_state.
		err = s.GuildDB.DeleteConfig(ctx, guildID)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigDeletionFailedPayload{
					GuildID: guildID,
					Reason:  err.Error(),
				},
				Error: err,
			}, err
		}

		// Success payload includes the resource snapshot so consumers can act.
		return GuildOperationResult{
			Success: &guildevents.GuildConfigDeletedPayload{
				GuildID:       guildID,
				ResourceState: rs,
			},
		}, nil
	})
}
