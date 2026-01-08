package guildintegrationtests

import (
	"testing"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
)

// TestDeleteGuildConfig_ResourceState verifies that deleting a guild config
// snapshots the explicit resource ID columns into the ResourceState JSONB and
// clears the explicit ID columns.
func TestDeleteGuildConfig_ResourceState(t *testing.T) {
    deps := SetupTestGuildService(t)
    defer deps.Cleanup()

    guildID := sharedtypes.GuildID("888888888888888888")
    config := &guildtypes.GuildConfig{
        GuildID:              guildID,
        SignupChannelID:      "111111111111111111",
        SignupMessageID:      "222222222222222222",
        EventChannelID:       "333333333333333333",
        LeaderboardChannelID: "444444444444444444",
        UserRoleID:           "555555555555555555",
        EditorRoleID:         "666666666666666666",
        AdminRoleID:          "777777777777777777",
        SignupEmoji:          "✅",
    }

    // Create the config
    _, err := deps.Service.CreateGuildConfig(deps.Ctx, config)
    if err != nil {
        t.Fatalf("setup: CreateGuildConfig failed: %v", err)
    }

    // Delete the config (this should snapshot resources into ResourceState)
    result, err := deps.Service.DeleteGuildConfig(deps.Ctx, guildID)
    if err != nil {
        t.Fatalf("DeleteGuildConfig returned unexpected error: %v", err)
    }
    if result.Success == nil {
        t.Fatalf("expected success deleting config, got failure: %+v", result.Failure)
    }

    // Retrieve raw DB row to inspect stored fields regardless of is_active flag.
    var row struct {
        ResourceState        []byte `bun:"resource_state"`
        SignupChannelID      string `bun:"signup_channel_id"`
        SignupMessageID      string `bun:"signup_message_id"`
        EventChannelID       string `bun:"event_channel_id"`
        LeaderboardChannelID string `bun:"leaderboard_channel_id"`
        UserRoleID           string `bun:"user_role_id"`
        EditorRoleID         string `bun:"editor_role_id"`
        AdminRoleID          string `bun:"admin_role_id"`
    }

    if err := deps.BunDB.NewSelect().Table("guild_configs").
        Column("resource_state", "signup_channel_id", "signup_message_id", "event_channel_id", "leaderboard_channel_id", "user_role_id", "editor_role_id", "admin_role_id").
        Where("guild_id = ?", guildID).
        Scan(deps.Ctx, &row); err != nil {
        t.Fatalf("DB raw select failed: %v", err)
    }

    if len(row.ResourceState) == 0 {
        t.Fatalf("expected ResourceState JSONB to be populated after delete")
    }

    // The explicit ID columns should have been cleared (NULLs read as empty strings)
    if row.SignupChannelID != "" || row.SignupMessageID != "" || row.EventChannelID != "" || row.LeaderboardChannelID != "" || row.UserRoleID != "" || row.EditorRoleID != "" || row.AdminRoleID != "" {
        t.Fatalf("expected explicit resource ID columns to be cleared after delete; got %+v", row)
    }
}
