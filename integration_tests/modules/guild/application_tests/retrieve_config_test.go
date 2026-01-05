package guildintegrationtests

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
)

func TestGetGuildConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID)
		validateFn func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error)
	}{
		{
			name: "Success - Retrieve existing guild config",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("823456789012345678")
				config := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "834567890123456789",
					EventChannelID:       "845678901234567890",
					LeaderboardChannelID: "856789012345678901",
					UserRoleID:           "867890123456789012",
					SignupEmoji:          "✅",
				}

				// Create the config first
				_, err := deps.Service.CreateGuildConfig(deps.Ctx, config)
				if err != nil {
					t.Fatalf("Setup: Failed to create guild config: %v", err)
				}

				return deps.Ctx, guildID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err != nil {
					t.Fatalf("GetGuildConfig returned unexpected error: %v", err)
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

				successPayload, ok := result.Success.(*guildevents.GuildConfigRetrievedPayload)
				if !ok {
					t.Fatalf("Success payload was not of expected type *guildevents.GuildConfigRetrievedPayload")
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Success payload GuildID mismatch: expected %q, got %q", guildID, successPayload.GuildID)
				}

				if successPayload.Config.SignupChannelID != "834567890123456789" {
					t.Errorf("Config SignupChannelID mismatch: expected %q, got %q", "834567890123456789", successPayload.Config.SignupChannelID)
				}
				if successPayload.Config.EventChannelID != "845678901234567890" {
					t.Errorf("Config EventChannelID mismatch: expected %q, got %q", "845678901234567890", successPayload.Config.EventChannelID)
				}
			},
		},
		{
			name: "Failure - Guild config not found",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("nonexistent_guild")
				return deps.Ctx, guildID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil {
					t.Fatalf("Expected error for not found but got nil")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}

				failurePayload, ok := result.Failure.(*guildevents.GuildConfigRetrievalFailedPayload)
				if !ok {
					t.Fatalf("Failure payload was not of expected type")
				}

				if failurePayload.GuildID != guildID {
					t.Errorf("Failure payload GuildID mismatch: expected %q, got %q", guildID, failurePayload.GuildID)
				}
			},
		},
		{
			name: "Failure - Empty guild ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("")
				return deps.Ctx, guildID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil {
					t.Fatalf("Expected error for empty guild ID but got nil")
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				// Empty guild ID returns Error field, not Failure payload
				if result.Error == nil {
					t.Fatalf("Expected error in result.Error field but got nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := SetupTestGuildService(t)
			defer deps.Cleanup()

			ctx, guildID := tt.setupFn(t, deps)

			result, err := deps.Service.GetGuildConfig(ctx, guildID)

			tt.validateFn(t, deps, guildID, result, err)
		})
	}
}
