package guildservice

import (
	"context"
	"errors"
	"fmt"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/uptrace/bun"
)

// CreateGuildConfig creates a new guild configuration.
// Returns a GuildConfigResult for domain-level failures (validation/conflict)
// or a non-nil error for infrastructure failures (DB down, network issues).
func (s *GuildService) CreateGuildConfig(
	ctx context.Context,
	config *guildtypes.GuildConfig,
) (GuildConfigResult, error) {

	if config == nil {
		return GuildConfigResult{}, ErrNilConfig
	}

	guildID := config.GuildID

	// Named transaction function for observability
	createGuildConfigTx := func(ctx context.Context, db bun.IDB) (GuildConfigResult, error) {
		return s.executeCreateGuildConfig(ctx, db, config)
	}

	// Wrap with telemetry & transaction
	result, err := withTelemetry(s, ctx, "CreateGuildConfig", guildID, func(ctx context.Context) (GuildConfigResult, error) {
		return runInTx(s, ctx, createGuildConfigTx)
	})

	if err != nil {
		// Infrastructure error (DB/network/etc.) — handler can retry
		return GuildConfigResult{}, fmt.Errorf("CreateGuildConfig failed: %w", err)
	}

	// Domain result contains success/failure payload
	return result, nil
}

// executeCreateGuildConfig contains the core business logic for creating a guild config.
// Returns a domain FailureResult for validation/conflict, or a GuildConfigResult + error for infrastructure errors.
func (s *GuildService) executeCreateGuildConfig(
	ctx context.Context,
	db bun.IDB,
	config *guildtypes.GuildConfig,
) (GuildConfigResult, error) {

	if config.GuildID == "" {
		return results.FailureResult[*guildtypes.GuildConfig, error](ErrInvalidGuildID), nil
	}

	if err := config.Validate(); err != nil {
		return results.FailureResult[*guildtypes.GuildConfig, error](err), nil
	}

	existing, err := s.repo.GetConfig(ctx, db, config.GuildID)
	if err != nil && !errors.Is(err, guilddb.ErrNotFound) {
		return GuildConfigResult{}, fmt.Errorf("failed to get guild config for %s: %w", config.GuildID, err)
	}

	if existing != nil {
		return results.FailureResult[*guildtypes.GuildConfig, error](ErrGuildConfigConflict), nil
	}

	if err := s.repo.SaveConfig(ctx, db, config); err != nil {
		return GuildConfigResult{}, fmt.Errorf("failed to save guild config for %s: %w", config.GuildID, err)
	}

	return results.SuccessResult[*guildtypes.GuildConfig, error](config), nil
}
