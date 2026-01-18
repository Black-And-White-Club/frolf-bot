package guildintegrationtests

import (
	"context"
	"testing"

	guildevents "github.com/Black-And-White-Club/frolf-bot-shared/events/guild"
	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot-shared/utils/results"
)

func TestGetGuildConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFn    func(t *testing.T, deps TestDeps) (context.Context, sharedtypes.GuildID)
		validateFn func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error)
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
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				if err != nil {
					t.Fatalf("GetGuildConfig returned unexpected error: %v", err)
				}
				// No system error expected; checked via err above
				if result.Success == nil {
					t.Fatalf("Result contained nil Success payload. Failure payload: %+v", result.Failure)
				}
				if result.Failure != nil {
					t.Fatalf("Result contained non-nil Failure payload: %+v", result.Failure)
				}

				successPayload, ok := result.Success.(*guildevents.GuildConfigRetrievedPayloadV1)
				if !ok {
					t.Fatalf("Success payload was not of expected type *guildevents.GuildConfigRetrievedPayloadV1")
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
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
				if err != nil {
					t.Fatalf("Expected business failure but got system error: %v", err)
				}
				if result.Success != nil {
					t.Fatalf("Expected failure but got success: %+v", result.Success)
				}
				if result.Failure == nil {
					t.Fatalf("Expected failure payload but got nil")
				}

				failurePayload, ok := result.Failure.(*guildevents.GuildConfigRetrievalFailedPayloadV1)
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
			validateFn: func(t *testing.T, deps TestDeps, guildID sharedtypes.GuildID, result results.OperationResult, err error) {
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
				failurePayload, ok := result.Failure.(*guildevents.GuildConfigRetrievalFailedPayloadV1)
				if !ok {
					t.Fatalf("Failure payload was not of expected type")
				}
				if failurePayload.GuildID != guildID {
					t.Errorf("Failure payload GuildID mismatch: expected %q, got %q", guildID, failurePayload.GuildID)
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
