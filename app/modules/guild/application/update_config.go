package guildservice

import (
	"context"
	"errors"
	"fmt"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/ptr"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// UpdateGuildConfig updates an existing guild configuration.
func (s *GuildService) UpdateGuildConfig(
	ctx context.Context,
	config *guildtypes.GuildConfig,
) (GuildConfigResult, error) {
	if config == nil {
		return GuildConfigResult{}, ErrNilConfig
	}

	guildID := config.GuildID

	updateTx := func(ctx context.Context, db bun.IDB) (GuildConfigResult, error) {
		return s.executeUpdateGuildConfig(ctx, db, config)
	}

	result, err := withTelemetry(s, ctx, "UpdateGuildConfig", guildID, func(ctx context.Context) (GuildConfigResult, error) {
		return runInTx(s, ctx, updateTx)
	})

	if err != nil {
		return GuildConfigResult{}, fmt.Errorf("UpdateGuildConfig failed for %s: %w", guildID, err)
	}

	return result, nil
}

func (s *GuildService) executeUpdateGuildConfig(
	ctx context.Context,
	db bun.IDB,
	config *guildtypes.GuildConfig,
) (GuildConfigResult, error) {
	if config.GuildID == "" {
		return results.FailureResult[*guildtypes.GuildConfig, error](ErrInvalidGuildID), nil
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

	if err := s.repo.UpdateConfig(ctx, db, config.GuildID, updates); err != nil {
		if errors.Is(err, guilddb.ErrNoRowsAffected) {
			// Domain failure: Cannot update what doesn't exist
			return results.FailureResult[*guildtypes.GuildConfig, error](ErrGuildConfigNotFound), nil
		}
		// Infrastructure error
		return GuildConfigResult{}, fmt.Errorf("failed to update config in DB: %w", err)
	}

	return results.SuccessResult[*guildtypes.GuildConfig, error](config), nil
}
