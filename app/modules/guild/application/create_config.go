package guildservice

import (
	"context"
	"errors"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
	"github.com/google/go-cmp/cmp"
)

// CreateGuildConfig creates a new guild configuration.
// Idempotent: re-creating an identical config returns success.
func (s *GuildService) CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (results.OperationResult, error) {
	// Pre-validation (before telemetry)
	if config == nil {
		return creationFailure("", ErrNilConfig), nil
	}

	guildID := config.GuildID

	return s.withTelemetry(ctx, "CreateGuildConfig", guildID, func(ctx context.Context) (results.OperationResult, error) {
		// Validate guild ID
		if guildID == "" {
			return creationFailure(guildID, ErrInvalidGuildID), nil
		}

		// Domain-level validation
		if err := config.Validate(); err != nil {
			return creationFailure(guildID, err), nil
		}

		// Check for existing configuration (idempotency)
		existing, err := s.repo.GetConfig(ctx, guildID)
		if err != nil && !errors.Is(err, guilddb.ErrNotFound) {
			// Infrastructure error - should retry
			return creationFailure(guildID, err), err
		}

		if existing != nil {
			// Config exists - check if identical (idempotent success)
			if configsEqual(existing, config) {
				return results.SuccessResult(&guildevents.GuildConfigCreatedPayloadV1{
					GuildID: guildID,
					Config:  *existing,
				}), nil
			}
			// Conflict - exists but differs
			return creationFailure(guildID, ErrGuildConfigConflict), nil
		}

		// Save new configuration
		if err := s.repo.SaveConfig(ctx, config); err != nil {
			return creationFailure(guildID, err), err
		}

		return results.SuccessResult(&guildevents.GuildConfigCreatedPayloadV1{
			GuildID: guildID,
			Config:  *config,
		}), nil
	})
}

// creationFailure creates a failure result for config creation.
func creationFailure(guildID sharedtypes.GuildID, err error) results.OperationResult {
	return results.FailureResult(&guildevents.GuildConfigCreationFailedPayloadV1{
		GuildID: guildID,
		Reason:  err.Error(),
	})
}

// configCmpOptions defines how to compare GuildConfig structs for idempotency checks.
var configCmpOptions = []cmp.Option{
	cmp.Comparer(func(a, b *time.Time) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Equal(*b)
	}),
}

// configsEqual performs a deep comparison between two configurations.
func configsEqual(a, b *guildtypes.GuildConfig) bool {
	if a == nil || b == nil {
		return a == b
	}
	return cmp.Equal(*a, *b, configCmpOptions...)
}
