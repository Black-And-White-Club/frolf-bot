package guildservice

import (
	"context"
	"errors"
	"time"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/go-cmp/cmp"
)

// CreateGuildConfig creates a new guild configuration.
// Common domain errors for guild config
var (
	ErrGuildConfigConflict = errors.New("guild config already exists with different settings - use update instead")
	ErrInvalidGuildID      = errors.New("invalid guild ID")
	ErrNilContext          = errors.New("context cannot be nil")
)

// CreateGuildConfig creates a new guild configuration.
func (s *GuildService) CreateGuildConfig(ctx context.Context, config *guildtypes.GuildConfig) (GuildOperationResult, error) {
	if ctx == nil {
		return GuildOperationResult{
			Error: ErrNilContext,
		}, ErrNilContext
	}

	if config == nil {
		return createGuildConfigFailureResult("", config, errors.New("config payload is nil")), errors.New("config payload is nil")
	}
	guildID := config.GuildID
	if guildID == "" {
		return createGuildConfigFailureResult(guildID, config, ErrInvalidGuildID), nil
	}
	// Validation: require key config fields
	if config.SignupChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("signup channel ID required")), nil
	}
	if config.EventChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("event channel ID required")), nil
	}
	if config.LeaderboardChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("leaderboard channel ID required")), nil
	}
	if config.UserRoleID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("user role ID required")), nil
	}
	if config.SignupEmoji == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("signup emoji required")), nil
	}

	// Check if config already exists
	existing, err := s.GuildDB.GetConfig(ctx, guildID)
	if err != nil {
		// Database error occurred during lookup
		return createGuildConfigFailureResult(guildID, config, err), err
	}
	if existing != nil {
		if guildConfigsEqual(existing, config) {
			successPayload := &guildevents.GuildConfigCreatedPayload{
				GuildID: guildID,
				Config:  *existing,
			}
			return GuildOperationResult{
				Success: successPayload,
			}, nil
		}

		return createGuildConfigFailureResult(guildID, config, ErrGuildConfigConflict), nil
	}

	// Save config (repository handles conversion to DB model)
	err = s.GuildDB.SaveConfig(ctx, config)
	if err != nil {
		return createGuildConfigFailureResult(guildID, config, err), err
	}

	// Success payload (return the canonical config)
	successPayload := &guildevents.GuildConfigCreatedPayload{
		GuildID: guildID,
		Config:  *config,
	}
	return GuildOperationResult{
		Success: successPayload,
	}, nil
}

// createGuildConfigFailureResult is a helper to create standardized failure results
func createGuildConfigFailureResult(guildID sharedtypes.GuildID, config *guildtypes.GuildConfig, err error) GuildOperationResult {
	return GuildOperationResult{
		Success: nil,
		Failure: &guildevents.GuildConfigCreationFailedPayload{
			GuildID: guildID,
			Reason:  err.Error(),
		},
		Error: err,
	}
}

var guildConfigCmpOptions = []cmp.Option{
	cmp.Comparer(func(a, b *time.Time) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Equal(*b)
	}),
}

func guildConfigsEqual(a, b *guildtypes.GuildConfig) bool {
	if a == nil || b == nil {
		return a == b
	}

	return cmp.Equal(*a, *b, guildConfigCmpOptions...)
}
