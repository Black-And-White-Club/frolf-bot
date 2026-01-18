package guildservice

import (
	"context"
	"errors"

	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guilddb "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/infrastructure/repositories"
)

// GetGuildConfig retrieves the current guild configuration.
func (s *GuildService) GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (results.OperationResult, error) {
	return s.withTelemetry(ctx, "GetGuildConfig", guildID, func(ctx context.Context) (results.OperationResult, error) {
		// Validate
		if guildID == "" {
			return retrievalFailure(guildID, ErrInvalidGuildID), nil
		}

		// Fetch from DB
		config, err := s.repo.GetConfig(ctx, guildID)
		if err != nil {
			if errors.Is(err, guilddb.ErrNotFound) {
				// Not found is a domain failure, not an infrastructure error
				return retrievalFailure(guildID, ErrGuildConfigNotFound), nil
			}
			// Infrastructure error - should retry
			return retrievalFailure(guildID, err), err
		}

		// Success
		return results.SuccessResult(&guildevents.GuildConfigRetrievedPayloadV1{
			GuildID: guildID,
			Config:  *config,
		}), nil
	})
}

// retrievalFailure creates a failure result for config retrieval.
func retrievalFailure(guildID sharedtypes.GuildID, err error) results.OperationResult {
	return results.FailureResult(&guildevents.GuildConfigRetrievalFailedPayloadV1{
		GuildID: guildID,
		Reason:  err.Error(),
	})
}
