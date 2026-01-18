package guildservice

import (
	"context"

	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// DeleteGuildConfig performs a soft delete on a guild configuration.
// Idempotent: deleting a non-existent or already-deleted config succeeds.
func (s *GuildService) DeleteGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "DeleteGuildConfig", guildID, func(ctx context.Context) (results.OperationResult, error) {
		// Validate
		if guildID == "" {
			return deletionFailure(guildID, ErrInvalidGuildID), nil
		}

		// Execute soft delete
		// The repository handles idempotency (returns nil if already deleted)
		if err := s.repo.DeleteConfig(ctx, guildID); err != nil {
			// Infrastructure error - should retry
			return deletionFailure(guildID, err), err
		}

		// Success
		// The event signals deletion intent. Discord worker uses ResourceState
		// snapshot to clean up physical resources.
		return results.SuccessResult(&guildevents.GuildConfigDeletedPayloadV1{
			GuildID: guildID,
		}), nil
	})
}

// deletionFailure creates a failure result for config deletion.
func deletionFailure(guildID sharedtypes.GuildID, err error) results.OperationResult {
	return results.FailureResult(&guildevents.GuildConfigDeletionFailedPayloadV1{
		GuildID: guildID,
		Reason:  err.Error(),
	})
}
