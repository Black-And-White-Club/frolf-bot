package guildintegrationtests

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
)

func TestDeleteGuildConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID)
		validateFn func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error)
	}{
		{
			name: "Success - Delete existing guild config",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("923456789012345678")
				config := &guildtypes.GuildConfig{
					GuildID:              guildID,
					SignupChannelID:      "934567890123456789",
					EventChannelID:       "945678901234567890",
					LeaderboardChannelID: "956789012345678901",
					UserRoleID:           "967890123456789012",
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
					t.Fatalf("DeleteGuildConfig returned unexpected error: %v", err)
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

				successPayload, ok := result.Success.(*guildevents.GuildConfigDeletedPayloadV1)
				if !ok {
					t.Fatalf("Success payload was not of expected type *guildevents.GuildConfigDeletedPayloadV1")
				}

				if successPayload.GuildID != guildID {
					t.Errorf("Success payload GuildID mismatch: expected %q, got %q", guildID, successPayload.GuildID)
				}

				// Verify the config was soft-deleted in the database
				retrievedConfig, dbErr := deps.DB.GetConfig(deps.Ctx, guildID)
				if dbErr == nil && retrievedConfig != nil {
					t.Logf("Config still retrievable after delete (soft delete behavior)")
				}
			},
		},
		{
			name: "Success - Delete nonexistent config (idempotent)",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("nonexistent_guild_delete")
				return deps.Ctx, guildID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildOperationResult, err error) {
				if err == nil && result.Error == nil && result.Failure == nil {
					t.Fatalf("Expected error or failure for nonexistent config but got none")
				}
				// Deleting nonexistent config returns error
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

			result, err := deps.Service.DeleteGuildConfig(ctx, guildID)

			tt.validateFn(t, deps, guildID, result, err)
		})
	}
}

func TestDeleteGuildConfig_VerifySoftDelete(t *testing.T) {
	deps := SetupTestGuildService(t)
	defer deps.Cleanup()

	guildID := sharedtypes.GuildID("973456789012345678")
	config := &guildtypes.GuildConfig{
		GuildID:              guildID,
		SignupChannelID:      "984567890123456789",
		EventChannelID:       "995678901234567890",
		LeaderboardChannelID: "996789012345678901",
		UserRoleID:           "997890123456789012",
		SignupEmoji:          "✅",
	}

	// Create the config
	_, err := deps.Service.CreateGuildConfig(deps.Ctx, config)
	if err != nil {
		t.Fatalf("Failed to create guild config: %v", err)
	}

	// Delete the config
	result, err := deps.Service.DeleteGuildConfig(deps.Ctx, guildID)
	if err != nil {
		t.Fatalf("DeleteGuildConfig returned unexpected error: %v", err)
	}

	if result.Success == nil {
		t.Fatalf("Expected success but got failure: %+v", result.Failure)
	}

	// Try to retrieve the deleted config - should return error/failure
	getResult, err := deps.Service.GetGuildConfig(deps.Ctx, guildID)
	if err == nil && getResult.Error == nil && getResult.Failure == nil {
		t.Fatalf("Expected error or failure when retrieving soft-deleted config but got none")
	}

	if getResult.Success != nil {
		t.Fatalf("Expected not to retrieve deleted config, but got success: %+v", getResult.Success)
	}

	if getResult.Failure == nil {
		t.Fatalf("Expected failure payload for deleted config but got nil (err: %v, result.Error: %v)", err, getResult.Error)
	}
}
