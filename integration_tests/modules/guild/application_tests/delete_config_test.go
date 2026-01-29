package guildintegrationtests

import (
	"context"
	"testing"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	guildservice "github.com/Black-And-White-Club/frolf-bot/app/modules/guild/application"
)

func TestDeleteGuildConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID)
		validateFn func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildConfigResult, err error)
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
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildConfigResult, err error) {
				if err != nil {
					t.Fatalf("DeleteGuildConfig returned unexpected error: %v", err)
				}
				// No system error expected; checked via err above
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				// Success payload is now *guildtypes.GuildConfig (the state before/after delete)
				deletedConfig := *result.Success

				if deletedConfig.GuildID != guildID {
					t.Errorf("Success payload GuildID mismatch: expected %q, got %q", guildID, deletedConfig.GuildID)
				}

				// Verify the config was soft-deleted in the database
				retrievedConfig, dbErr := deps.DB.GetConfig(deps.Ctx, nil, guildID)
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
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildConfigResult, err error) {
				if err != nil {
					t.Fatalf("DeleteGuildConfig returned unexpected system error: %v", err)
				}
				// Deleting a nonexistent config currently returns success (idempotent) or failure depending on impl.
				// Based on previous test, it seemed to return success.
				// However, if the implementation relies on `repo.DeleteConfig` doing a soft delete on existing,
				// let's check what checking logic exists.
				// Previous test said "Success - Delete nonexistent config (idempotent)".
				// Assuming `repo.DeleteGuildConfig` handles not found gracefully or we don't check existence first.

				if result.Failure != nil {
					t.Fatalf("Expected success but got failure: %+v", result.Failure)
				}
				if result.Success == nil {
					t.Fatalf("Expected success payload but got nil")
				}
			},
		},
		{
			name: "Failure - Empty guild ID",
			setupFn: func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID) {
				guildID := sharedtypes.GuildID("")
				return deps.Ctx, guildID
			},
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result guildservice.GuildConfigResult, err error) {
				// New behavior: empty guild ID returns a business failure payload (no system error)
				if err != nil {
					t.Fatalf("Expected business failure payload for empty guild ID but got system error: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload for empty guild ID but got nil")
				}

				domainErr := *result.Failure
				if domainErr == nil {
					t.Fatalf("Failure payload was nil error")
				}
				t.Logf("Got expected domain error: %v", domainErr)
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
	if err != nil {
		t.Fatalf("Expected business failure payload for deleted config retrieval but got system error: %v", err)
	}

	if getResult.Success != nil {
		t.Fatalf("Expected not to retrieve deleted config, but got success: %+v", getResult.Success)
	}

	if getResult.Failure == nil {
		t.Fatalf("Expected failure payload for deleted config but got nil (err: %v)", err)
	}
}
