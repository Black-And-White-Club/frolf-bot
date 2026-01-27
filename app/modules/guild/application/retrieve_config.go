package guildservice

import (
	"context"
	"errors"
	"fmt"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// GetGuildConfig retrieves a guild configuration.
func (s *GuildService) GetGuildConfig(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) (GuildConfigResult, error) {
	if guildID == "" {
		return results.FailureResult[*guildtypes.GuildConfig, error](ErrInvalidGuildID), nil
	}

	getTx := func(ctx context.Context, db bun.IDB) (GuildConfigResult, error) {
		return s.executeGetGuildConfig(ctx, db, guildID)
	}

	result, err := withTelemetry(s, ctx, "GetGuildConfig", guildID, func(ctx context.Context) (GuildConfigResult, error) {
		return runInTx(s, ctx, getTx)
	})

	if err != nil {
		return GuildConfigResult{}, fmt.Errorf("GetGuildConfig failed for %s: %w", guildID, err)
	}

	return result, nil
}

func (s *GuildService) executeGetGuildConfig(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
) (GuildConfigResult, error) {
	config, err := s.repo.GetConfig(ctx, db, guildID)
	if err != nil {
		if errors.Is(err, guilddb.ErrNotFound) {
			// Domain failure: The config simply doesn't exist
			return results.FailureResult[*guildtypes.GuildConfig, error](ErrGuildConfigNotFound), nil
		}
		// Infra failure: DB is likely down
		return GuildConfigResult{}, err
	}
	return results.SuccessResult[*guildtypes.GuildConfig, error](config), nil
}
