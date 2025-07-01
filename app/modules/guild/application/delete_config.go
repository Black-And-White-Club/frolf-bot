package guildservice

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
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

		// Soft delete: set IsActive = false
		updates := map[string]interface{}{"is_active": false}
		err = s.GuildDB.UpdateConfig(ctx, guildID, updates)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigDeletionFailedPayload{
					GuildID: guildID,
					Reason:  err.Error(),
				},
				Error: err,
			}, err
		}

		// Success payload
		return GuildOperationResult{
			Success: &guildevents.GuildConfigDeletedPayload{
				GuildID: guildID,
			},
		}, nil
	})
}
