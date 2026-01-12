package guildintegrationtests

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
)

func TestCreateGuildConfig(t *testing.T) {
	tests := []struct {
		name             string
		setupFn          func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig)
		validateFn       func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error)
		expectedSuccess  bool
		expectedErrorMsg string
	}{
		{
			name: "Success - Valid guild config creation",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
				GuildID:              "123456789012345678",
				SignupChannelID:      "234567890123456789",
				EventChannelID:       "345678901234567890",
				LeaderboardChannelID: "456789012345678901",
				UserRoleID:           "567890123456789012",
				SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err != nil {
					t.Fatalf("CreateGuildConfig returned unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Result contained unexpected Error: %v", result.Error)
				}
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*guildevents.GuildConfigCreatedPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *guildevents.GuildConfigCreatedPayload")
				}

				// Verify the config was saved to the database
				retrievedConfig, dbErr := deps.DB.GetConfig(deps.Ctx, guildID)
				if dbErr != nil {
					t.Fatalf("Failed to retrieve guild config %q from DB: %v", guildID, dbErr)
				}
				if retrievedConfig == nil {
					t.Fatalf("Guild config %q not found in database after creation", guildID)
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Success payload GuildID mismatch: expected %q, got %q", guildID, successPayload.GuildID)
				}

				if retrievedConfig.GuildID != guildID {
					t.Errorf("Retrieved config GuildID mismatch: expected %q, got %q", guildID, retrievedConfig.GuildID)
				}
			if retrievedConfig.SignupChannelID != "234567890123456789" {
				t.Errorf("Retrieved config SignupChannelID mismatch: expected %q, got %q", "234567890123456789", retrievedConfig.SignupChannelID)
				}
			if retrievedConfig.EventChannelID != "345678901234567890" {
				t.Errorf("Retrieved config EventChannelID mismatch: expected %q, got %q", "345678901234567890", retrievedConfig.EventChannelID)
				}
		if retrievedConfig.LeaderboardChannelID != "456789012345678901" {
			t.Errorf("Retrieved config LeaderboardChannelID mismatch: expected %q, got %q", "456789012345678901", retrievedConfig.LeaderboardChannelID)
				}
			if retrievedConfig.UserRoleID != "567890123456789012" {
				t.Errorf("Retrieved config UserRoleID mismatch: expected %q, got %q", "567890123456789012", retrievedConfig.UserRoleID)
				}
				if retrievedConfig.SignupEmoji != "✅" {
					t.Errorf("Retrieved config SignupEmoji mismatch: expected %q, got %q", "✅", retrievedConfig.SignupEmoji)
				}
			},
			expectedSuccess: true,
		},
		{
			name: "Success - Idempotent when config matches",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
			GuildID:              "223456789012345678",
			SignupChannelID:      "334567890123456789",
			EventChannelID:       "445678901234567890",
			LeaderboardChannelID: "556789012345678901",
			UserRoleID:           "667890123456789012",
				SignupEmoji:          "✅",
				}
				// Create the config first time
				_, err := deps.Service.CreateGuildConfig(deps.Ctx, config)
				if err != nil {
					t.Fatalf("Initial CreateGuildConfig failed: %v", err)
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err != nil {
					t.Fatalf("CreateGuildConfig returned unexpected error: %v", err)
				}
				if result.Error != nil {
					t.Fatalf("Result contained unexpected Error: %v", result.Error)
				}
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
			},
			expectedSuccess: true,
		},
		{
			name: "Failure - Missing required field (signup channel)",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
			GuildID:              "323456789012345678",
			SignupChannelID:      "", // Missing
			EventChannelID:       "445678901234567890",
			LeaderboardChannelID: "556789012345678901",
			UserRoleID:           "667890123456789012",
				SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil && result.Error == nil && result.Failure == nil {
					t.Fatalf("Expected error or failure for missing signup channel but got none")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}

				failurePayload, ok := result.Failure.(*guildevents.GuildConfigCreationFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type")
				}

				if failurePayload.GuildID != guildID {
					t.Errorf("Failure payload GuildID mismatch: expected %q, got %q", guildID, failurePayload.GuildID)
				}
			},
			expectedSuccess: false,
		},
		{
			name: "Failure - Missing required field (event channel)",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
			GuildID:              "423456789012345678",
			SignupChannelID:      "534567890123456789",
			EventChannelID:       "", // Missing
			LeaderboardChannelID: "656789012345678901",
			UserRoleID:           "767890123456789012",
				SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil && result.Error == nil && result.Failure == nil {
					t.Fatalf("Expected error or failure for missing event channel but got none")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}
			},
			expectedSuccess: false,
		},
		{
			name: "Failure - Empty guild ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				config := &guildtypes.GuildConfig{
				GuildID:              "", // Empty
			SignupChannelID:      "534567890123456789",
			EventChannelID:       "545678901234567890",
			LeaderboardChannelID: "556789012345678901",
			UserRoleID:           "567890123456789012",
				SignupEmoji:          "✅",
				}
				return deps.Ctx, config
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil && result.Error == nil && result.Failure == nil {
					t.Fatalf("Expected error or failure for empty guild ID but got none")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}
			},
			expectedSuccess: false,
		},
		{
			name: "Failure - Nil config",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, *guildtypes.GuildConfig) {
				return deps.Ctx, nil
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil {
					t.Fatalf("Expected error for nil config but got nil")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}
			},
			expectedSuccess: false,
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

			result, err := deps.Service.CreateGuildConfig(ctx, config)

			tt.validateFn(t, deps, guildID, result, err)
		})
	}
}

func TestCreateGuildConfig_AlreadyExists(t *testing.T) {
	deps := SetupTestGuildService(t)
	defer deps.Cleanup()

	// Create initial config
	initialConfig := &guildtypes.GuildConfig{
		GuildID:              "623456789012345678",
		SignupChannelID:      "634567890123456789",
		EventChannelID:       "645678901234567890",
		LeaderboardChannelID: "656789012345678901",
		UserRoleID:           "667890123456789012",
		SignupEmoji:          "🔥",
	}

	_, err := deps.Service.CreateGuildConfig(deps.Ctx, initialConfig)
	if err != nil {
		t.Fatalf("Initial CreateGuildConfig failed: %v", err)
	}

	// Try to create with different settings
	differentConfig := &guildtypes.GuildConfig{
		GuildID:              "623456789012345678",
		SignupChannelID:      "724567890123456789",
		EventChannelID:       "735678901234567890",
		LeaderboardChannelID: "746789012345678901",
		UserRoleID:           "757890123456789012",
		SignupEmoji:          "✨",
	}

	result, err := deps.Service.CreateGuildConfig(deps.Ctx, differentConfig)
	if err == nil && result.Error == nil && result.Failure == nil {
		t.Fatalf("Expected error or failure when config already exists with different settings but got none")
	}

	if result.Success != nil {
		t.Fatalf("Expected failure when config already exists with different settings, but got success")
	}

	if result.Failure == nil {
		t.Fatalf("Expected failure payload but got nil")
	}

	failurePayload, ok := result.Failure.(*guildevents.GuildConfigCreationFailedPayload)
	if !ok {
		t.Fatalf("Failure payload was not of expected type")
	}

	if failurePayload.GuildID != "623456789012345678" {
		t.Errorf("Failure payload GuildID mismatch: expected %q, got %q", "623456789012345678", failurePayload.GuildID)
	}
}
