package guildservice

import (
	"context"
	"errors"
	"fmt"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// GetGuildConfig retrieves the current guild configuration.
func (s *GuildService) GetGuildConfig(ctx context.Context, guildID sharedtypes.GuildID) (GuildOperationResult, error) {
	return s.serviceWrapper(ctx, "GetGuildConfig", guildID, func(ctx context.Context) (GuildOperationResult, error) {
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

		config, err := s.GuildDB.GetConfig(ctx, guildID)
		if err != nil {
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigRetrievalFailedPayloadV1{
					GuildID: guildID,
					Reason:  err.Error(),
				},
				Error: err,
			}, err
		}
		if config == nil {
			// Wrap sentinel so higher-level wrappers still produce enriched context while errors.Is matches.
			notFoundErr := fmt.Errorf("%w", ErrGuildConfigNotFound)
			return GuildOperationResult{
				Failure: &guildevents.GuildConfigRetrievalFailedPayloadV1{
					GuildID: guildID,
					Reason:  ErrGuildConfigNotFound.Error(),
				},
				Error: notFoundErr,
			}, nil
		}

		return GuildOperationResult{
			Success: &guildevents.GuildConfigRetrievedPayloadV1{
				GuildID: guildID,
				Config:  *config,
			},
		}, nil
	})
}
