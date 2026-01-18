package guildservice

import (
	"context"
	"errors"

	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/ptr"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
)

// UpdateGuildConfig updates an existing guild configuration using partial updates.
func (s *GuildService) UpdateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (results.OperationResult, error) {
	// Pre-validation (before telemetry)
	if config == nil {
		return updateFailure("", ErrNilConfig), nil
	}

	guildID := config.GuildID

	return s.withTelemetry(ctx, "UpdateGuildConfig", guildID, func(ctx context.Context) (results.OperationResult, error) {
		// Validate
		if guildID == "" {
			return updateFailure(guildID, ErrInvalidGuildID), nil
		}

		// Build partial update fields
		updates := &guilddb.UpdateFields{
			SignupChannelID:      ptr.IfNonEmpty(config.SignupChannelID),
			SignupMessageID:      ptr.IfNonEmpty(config.SignupMessageID),
			EventChannelID:       ptr.IfNonEmpty(config.EventChannelID),
			LeaderboardChannelID: ptr.IfNonEmpty(config.LeaderboardChannelID),
			UserRoleID:           ptr.IfNonEmpty(config.UserRoleID),
			EditorRoleID:         ptr.IfNonEmpty(config.EditorRoleID),
			AdminRoleID:          ptr.IfNonEmpty(config.AdminRoleID),
			SignupEmoji:          ptr.IfNonEmpty(config.SignupEmoji),
			AutoSetupCompleted:   ptr.IfTrue(config.AutoSetupCompleted),
			SetupCompletedAt:     ptr.TimeToUnixNano(config.SetupCompletedAt),
		}

		// Perform the update
		if err := s.repo.UpdateConfig(ctx, guildID, updates); err != nil {
			if errors.Is(err, guilddb.ErrNoRowsAffected) {
				// No active config found - domain failure
				return updateFailure(guildID, ErrGuildConfigNotFound), nil
			}
			// Infrastructure error - should retry
			return updateFailure(guildID, err), err
		}

		// Success
		return results.SuccessResult(&guildevents.GuildConfigUpdatedPayloadV1{
			GuildID: guildID,
			Config:  *config,
		}), nil
	})
}

// updateFailure creates a failure result for config update.
func updateFailure(guildID sharedtypes.GuildID, err error) results.OperationResult {
	return results.FailureResult(&guildevents.GuildConfigUpdateFailedPayloadV1{
		GuildID: guildID,
		Reason:  err.Error(),
	})
}
