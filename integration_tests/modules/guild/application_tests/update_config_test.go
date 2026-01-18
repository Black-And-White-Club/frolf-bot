package guildintegrationtests

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

func TestUpdateGuildConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig)
		validateFn func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error)
	}{
		{
			name: "Success - Update existing guild config",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				guildID := sharedtypes.GuildID("423456789012345678")
				initialConfig := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "424567890123456789",
					EventChannelID:       "425678901234567890",
					LeaderboardChannelID: "426789012345678901",
					UserRoleID:           "427890123456789012",
					SignupEmoji:          "🔥",
				}
				_, err := deps.Service.CreateGuildConfig(deps.Ctx, initialConfig)
				if err != nil {
					t.Fatalf("Setup: Failed to create guild config: %v", err)
				}

				// Updated config
				updatedConfig := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "524567890123456789",
					EventChannelID:       "525678901234567890",
					LeaderboardChannelID: "526789012345678901",
					UserRoleID:           "527890123456789012",
					SignupEmoji:          "✅",
				}

				return deps.Ctx, updatedConfig
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				if err != nil {
					t.Fatalf("UpdateGuildConfig returned unexpected error: %v", err)
				}
				// No system error expected; checked via err above
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*guildevents.GuildConfigUpdatedPayloadV1)
				if !ok {
					t.Fatalf("Success payload was not of expected type *guildevents.GuildConfigUpdatedPayloadV1")
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Success payload GuildID mismatch: expected %q, got %q", guildID, successPayload.GuildID)
				}

				// Verify the config was updated in the database
				retrievedConfig, dbErr := deps.DB.GetConfig(deps.Ctx, guildID)
				if dbErr != nil {
					t.Fatalf("Failed to retrieve guild config %q from DB: %v", guildID, dbErr)
				}

				if retrievedConfig.SignupChannelID != "524567890123456789" {
					t.Errorf("Updated config SignupChannelID mismatch: expected %q, got %q", "524567890123456789", retrievedConfig.SignupChannelID)
				}
				if retrievedConfig.EventChannelID != "525678901234567890" {
					t.Errorf("Updated config EventChannelID mismatch: expected %q, got %q", "525678901234567890", retrievedConfig.EventChannelID)
				}
				if retrievedConfig.LeaderboardChannelID != "526789012345678901" {
					t.Errorf("Updated config LeaderboardChannel mismatch: expected %q, got %q", "526789012345678901", retrievedConfig.LeaderboardChannelID)
				}
				if retrievedConfig.UserRoleID != "527890123456789012" {
					t.Errorf("Updated config UserRole mismatch: expected %q, got %q", "527890123456789012", retrievedConfig.UserRoleID)
				}
				if retrievedConfig.SignupEmoji != "✅" {
					t.Errorf("Updated config SignupEmoji mismatch: expected %q, got %q", "✅", retrievedConfig.SignupEmoji)
				}
			},
		},
		{
			name: "Failure - Config not found",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
					GuildID:              "623456789012345678",
					SignupChannelID:      "624567890123456789",
					EventChannelID:       "625678901234567890",
					LeaderboardChannelID: "626789012345678901",
					UserRoleID:           "627890123456789012",
					SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected business failure but got system error: %v", err)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}

				failurePayload, ok := result.Failure.(*guildevents.GuildConfigUpdateFailedPayloadV1)
				if !ok {
					t.Fatalf("Failure payload was not of expected type")
				}

				if failurePayload.GuildID != guildID {
					t.Errorf("Failure payload GuildID mismatch: expected %q, got %q", guildID, failurePayload.GuildID)
				}
			},
		},
		{
			name: "Failure - Nil config",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				return deps.Ctx, nil
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				// Service returns a business failure payload for nil config (no system error)
				if err != nil {
					t.Fatalf("Expected business failure but got system error: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}
			},
		},
		{
			name: "Failure - Empty guild ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
					GuildID:              "",
					SignupChannelID:      "724567890123456789",
					EventChannelID:       "725678901234567890",
					LeaderboardChannelID: "726789012345678901",
					UserRoleID:           "727890123456789012",
					SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				// Service returns a business failure payload for empty guild ID (no system error)
				if err != nil {
					t.Fatalf("Expected business failure but got system error: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestGuildService(t)
			defer deps.Cleanup()

			ctx, config := tt.setupFn(t, deps)

			var guildID sharedtypes.GuildID
			if config != nil {
				guildID = config.GuildID
			}

			result, err := deps.Service.UpdateGuildConfig(ctx, config)

			tt.validateFn(t, deps, guildID, result, err)
		})
	}
}
