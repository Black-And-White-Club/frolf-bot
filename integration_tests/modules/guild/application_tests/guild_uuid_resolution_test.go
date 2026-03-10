package guildintegrationtests

import (
	"context"
	"testing"

	guildtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/guild"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fullConfig returns a GuildConfig with all required fields populated.
func fullConfig(guildID sharedtypes.GuildID) *guildtypes.GuildConfig {
	return &guildtypes.GuildConfig{
		GuildID:              guildID,
		SignupChannelID:      "111111111111111111",
		EventChannelID:       "222222222222222222",
		LeaderboardChannelID: "333333333333333333",
		UserRoleID:           "444444444444444444",
		SignupEmoji:          "✅",
	}
}

// TestGetConfig_UUIDResolution covers the backfill bug fix where GetConfig must
// resolve a club UUID to its Discord guild snowflake before querying guild_configs,
// which is keyed on Discord guild IDs.
func TestGetConfig_UUIDResolution(t *testing.T) {
	t.Run("club UUID maps to known discord guild ID - config is found", func(t *testing.T) {
		deps := SetupTestGuildService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		discordGuildID := sharedtypes.GuildID("555000111222333444")
		clubUUID := uuid.New()

		// Clubs is not in the standard cleanup list.
		_, _ = deps.BunDB.ExecContext(ctx, "DELETE FROM clubs WHERE discord_guild_id = ?", discordGuildID)

		// Create a guild config keyed on the Discord guild ID.
		_, err := deps.Service.CreateGuildConfig(ctx, fullConfig(discordGuildID))
		require.NoError(t, err)

		// Map the club UUID to the Discord guild ID.
		_, err = deps.BunDB.ExecContext(ctx,
			"INSERT INTO clubs (uuid, name, discord_guild_id) VALUES (?, 'Test Club', ?)",
			clubUUID, discordGuildID)
		require.NoError(t, err)

		// GetGuildConfig with the club UUID — must resolve to discordGuildID.
		result, err := deps.Service.GetGuildConfig(ctx, sharedtypes.GuildID(clubUUID.String()))
		require.NoError(t, err)
		require.Nil(t, result.Failure, "should not have a failure payload")
		require.NotNil(t, result.Success)
		assert.Equal(t, discordGuildID, (*result.Success).GuildID)
	})

	t.Run("club UUID with no matching clubs row - returns not-found gracefully", func(t *testing.T) {
		deps := SetupTestGuildService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		orphanUUID := uuid.New() // not present in the clubs table

		// resolvedGuildID stays as the UUID; guild_configs has no matching row → not-found.
		result, err := deps.Service.GetGuildConfig(ctx, sharedtypes.GuildID(orphanUUID.String()))
		require.NoError(t, err)
		require.NotNil(t, result.Failure, "should return a not-found failure payload")
		assert.Nil(t, result.Success)
	})

	t.Run("standard Discord snowflake - UUID resolution skipped, config found directly", func(t *testing.T) {
		deps := SetupTestGuildService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		discordGuildID := sharedtypes.GuildID("444000111222333444")

		_, err := deps.Service.CreateGuildConfig(ctx, fullConfig(discordGuildID))
		require.NoError(t, err)

		// Passing a Discord snowflake directly: uuid.Parse fails, resolution is
		// skipped, and guild_configs is queried as-is.
		result, err := deps.Service.GetGuildConfig(ctx, discordGuildID)
		require.NoError(t, err)
		require.NotNil(t, result.Success)
		require.Nil(t, result.Failure)
		assert.Equal(t, discordGuildID, (*result.Success).GuildID)
	})
}
