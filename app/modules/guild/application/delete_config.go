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

// DeleteGuildConfig performs a soft delete and returns the final state of the config.
// Returns a GuildConfigResult for domain failures or an error for infrastructure issues.
func (s *GuildService) DeleteGuildConfig(
	ctx context.Context,
	guildID sharedtypes.GuildID,
) (GuildConfigResult, error) {

	if guildID == "" {
		// Return as a domain failure since the ID itself is invalid
		return results.FailureResult[*guildtypes.GuildConfig, error](ErrInvalidGuildID), nil
	}

	// Named transaction function
	deleteGuildConfigTx := func(ctx context.Context, db bun.IDB) (GuildConfigResult, error) {
		return s.executeDeleteGuildConfig(ctx, db, guildID)
	}

	// Wrap with telemetry & transaction
	result, err := withTelemetry(s, ctx, "DeleteGuildConfig", guildID, func(ctx context.Context) (GuildConfigResult, error) {
		return runInTx(s, ctx, deleteGuildConfigTx)
	})

	if err != nil {
		// Infrastructure error — handler can retry
		return GuildConfigResult{}, fmt.Errorf("DeleteGuildConfig failed for %s: %w", guildID, err)
	}

	return result, nil
}

// executeDeleteGuildConfig contains the core logic for the soft delete.
func (s *GuildService) executeDeleteGuildConfig(
	ctx context.Context,
	db bun.IDB,
	guildID sharedtypes.GuildID,
) (GuildConfigResult, error) {

	// Perform the soft delete (typically updates a deleted_at column or state)
	if err := s.repo.DeleteConfig(ctx, db, guildID); err != nil {
		// Infrastructure error
		return GuildConfigResult{}, fmt.Errorf("failed to delete config in DB: %w", err)
	}

	// Fetch the final state (useful for cleanup events that need the IDs)
	deletedConfig, err := s.repo.GetConfigIncludeDeleted(ctx, db, guildID)
	if err != nil {
		if errors.Is(err, guilddb.ErrNotFound) {
			// If never existed, return success with just the GuildID for idempotency
			return results.SuccessResult[*guildtypes.GuildConfig, error](&guildtypes.GuildConfig{
				GuildID: guildID,
			}), nil
		}
		// Infrastructure error
		return GuildConfigResult{}, fmt.Errorf("failed to fetch deleted config state: %w", err)
	}

	return results.SuccessResult[*guildtypes.GuildConfig, error](deletedConfig), nil
}
