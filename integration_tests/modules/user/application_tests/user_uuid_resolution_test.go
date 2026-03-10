package userintegrationtests

import (
	"context"
	"testing"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	userservice "github.com/Black-And-White-Club/frolf-bot/app/modules/user/application"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetUserRole_UUIDResolution covers the backfill bug fix where GetUserRole
// must resolve a club UUID to its Discord guild snowflake before querying
// guild_memberships, which is keyed on Discord guild IDs.
func TestGetUserRole_UUIDResolution(t *testing.T) {
	t.Run("club UUID maps to known discord guild ID - role is found", func(t *testing.T) {
		deps := SetupTestUserService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		discordGuildID := sharedtypes.GuildID("777000111222333444")
		clubUUID := uuid.New()
		userID := sharedtypes.DiscordID("123456789012345001")

		if err := testutils.CleanUserIntegrationTables(ctx, deps.BunDB); err != nil {
			t.Fatalf("CleanUserIntegrationTables: %v", err)
		}
		// Clubs is not in the standard cleanup list.
		_, _ = deps.BunDB.ExecContext(ctx, "DELETE FROM clubs WHERE discord_guild_id = ?", discordGuildID)

		// Create user with a guild membership keyed on the Discord guild ID.
		createResult, err := deps.Service.CreateUser(ctx, discordGuildID, userID, nil, nil, nil)
		require.NoError(t, err)
		require.True(t, createResult.IsSuccess())

		// Map the club UUID to the Discord guild ID.
		_, err = deps.BunDB.ExecContext(ctx,
			"INSERT INTO clubs (uuid, name, discord_guild_id) VALUES (?, 'Test Club', ?)",
			clubUUID, discordGuildID)
		require.NoError(t, err)

		// GetUserRole with the club UUID — must resolve to discordGuildID.
		result, err := deps.Service.GetUserRole(ctx, sharedtypes.GuildID(clubUUID.String()), userID)
		require.NoError(t, err)
		require.True(t, result.IsSuccess(), "role should be found via UUID-to-snowflake resolution")
		assert.Equal(t, sharedtypes.UserRoleUser, *result.Success)
	})

	t.Run("club UUID with no matching clubs row - returns not-found gracefully", func(t *testing.T) {
		deps := SetupTestUserService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		orphanUUID := uuid.New() // not present in the clubs table
		userID := sharedtypes.DiscordID("123456789012345002")

		if err := testutils.CleanUserIntegrationTables(ctx, deps.BunDB); err != nil {
			t.Fatalf("CleanUserIntegrationTables: %v", err)
		}

		// resolvedGuildID stays as the UUID; guild_memberships has no row → not-found.
		result, err := deps.Service.GetUserRole(ctx, sharedtypes.GuildID(orphanUUID.String()), userID)
		require.NoError(t, err)
		require.True(t, result.IsFailure(), "should return a not-found failure, not a system error")
		require.NotNil(t, result.Failure)
		assert.ErrorIs(t, *result.Failure, userservice.ErrUserNotFound)
	})

	t.Run("standard Discord snowflake - UUID resolution skipped, works as before", func(t *testing.T) {
		deps := SetupTestUserService(t)
		defer deps.Cleanup()

		ctx := context.Background()
		discordGuildID := sharedtypes.GuildID("666000111222333444")
		userID := sharedtypes.DiscordID("123456789012345003")

		if err := testutils.CleanUserIntegrationTables(ctx, deps.BunDB); err != nil {
			t.Fatalf("CleanUserIntegrationTables: %v", err)
		}

		createResult, err := deps.Service.CreateUser(ctx, discordGuildID, userID, nil, nil, nil)
		require.NoError(t, err)
		require.True(t, createResult.IsSuccess())

		// Passing the Discord snowflake directly: uuid.Parse will fail, so resolution
		// is skipped and the query runs against guild_memberships as-is.
		result, err := deps.Service.GetUserRole(ctx, discordGuildID, userID)
		require.NoError(t, err)
		require.True(t, result.IsSuccess(), "role should be found directly via Discord snowflake")
		assert.Equal(t, sharedtypes.UserRoleUser, *result.Success)
	})
}
