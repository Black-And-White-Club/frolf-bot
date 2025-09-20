package guildservice

import (
	"context"
	"errors"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// CreateGuildConfig creates a new guild configuration.
// Common domain errors for guild config
var (
	ErrGuildConfigAlreadyExists = errors.New("guild config already exists")
	ErrInvalidGuildID           = errors.New("invalid guild ID")
	ErrNilContext               = errors.New("context cannot be nil")
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
		return createGuildConfigFailureResult(guildID, config, ErrInvalidGuildID), ErrInvalidGuildID
	}
	// Validation: require key config fields
	if config.SignupChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("signup channel ID required")), errors.New("signup channel ID required")
	}
	if config.EventChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("event channel ID required")), errors.New("event channel ID required")
	}
	if config.LeaderboardChannelID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("leaderboard channel ID required")), errors.New("leaderboard channel ID required")
	}
	if config.UserRoleID == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("user role ID required")), errors.New("user role ID required")
	}
	if config.SignupEmoji == "" {
		return createGuildConfigFailureResult(guildID, config, errors.New("signup emoji required")), errors.New("signup emoji required")
	}

	// Check if config already exists
	existing, err := s.GuildDB.GetConfig(ctx, guildID)
	if err != nil {
		// Database error occurred during lookup
		return createGuildConfigFailureResult(guildID, config, err), err
	}
	if existing != nil {
		// Config already exists
		return createGuildConfigFailureResult(guildID, config, ErrGuildConfigAlreadyExists), ErrGuildConfigAlreadyExists
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
