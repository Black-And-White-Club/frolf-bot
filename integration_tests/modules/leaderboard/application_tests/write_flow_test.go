package leaderboardintegrationtests

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboardservice "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/application"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
)

func TestProcessRound_CoreFlow(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			deps := SetupTestLeaderboardService(t)
			defer deps.Cleanup()

			ctx := context.Background()
			guildID := sharedtypes.GuildID("test_guild_wf")
			seasonID := "season_wf_1"
			roundID := uuid.New()

			// 1. Setup Data
			_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)

			// Create a season
			resSeason, err := deps.Service.StartNewSeason(ctx, guildID, seasonID, "Test Season")
			require.NoError(t, err)
			require.True(t, resSeason.IsSuccess())

			// Seed members with tags
			// Use "test_guild" for helper consistency if needed, but we used "test_guild_wf".
			// SetupLeaderboardWithEntries writes to "test_guild".
			// We should manually insert for our custom guild to be safe, or use "test_guild".
			// Let's use "test_guild" (defaultTestGuildID) to leverage helper for initial state,
			// but helper clears "test_guild".
			// Our guildID is "test_guild_wf". SetupLeaderboardWithEntries won't help us unless we use that guild ID.
			// Since SetupLeaderboardWithEntries hardcodes "test_guild", I'll manually seed for "test_guild_wf".

			// Setup members for "test_guild_wf"
			membersToSeed := []leaderboarddb.LeagueMember{
				{GuildID: string(guildID), MemberID: "user_a", CurrentTag: ptr(1)},
				{GuildID: string(guildID), MemberID: "user_b", CurrentTag: ptr(2)},
				{GuildID: string(guildID), MemberID: "user_c", CurrentTag: ptr(3)},
				{GuildID: string(guildID), MemberID: "user_d", CurrentTag: nil},
			}
			_, err = deps.BunDB.NewInsert().Model(&membersToSeed).Exec(ctx)
			require.NoError(t, err)

			// 2. Execute ProcessRound
			cmd := leaderboardservice.ProcessRoundCommand{
				GuildID: string(guildID),
				RoundID: roundID,
				Participants: []leaderboardservice.RoundParticipantInput{
					{MemberID: "user_b", FinishRank: 1},
					{MemberID: "user_a", FinishRank: 2},
					{MemberID: "user_c", FinishRank: 3},
					{MemberID: "user_d", FinishRank: 4},
				},
			}

			// Use ProcessRoundCommand (interface method)
			output, err := deps.Service.ProcessRoundCommand(ctx, cmd)
			require.NoError(t, err)
			require.NotNil(t, output)

			// 3. Verify Tag Changes
			expectedTags := map[string]int{
				"user_b": 1,
				"user_a": 2,
				"user_c": 3,
				// user_d: no tag
			}

			// Verify output.FinalParticipantTags
			for memberID, expectedTag := range expectedTags {
				if expectedTag > 0 {
					assert.Equal(t, expectedTag, output.FinalParticipantTags[memberID], "Member %s tag mismatch", memberID)
				}
			}
			_, dHasTag := output.FinalParticipantTags["user_d"]
			assert.False(t, dHasTag, "user_d should not have a tag")

			// Verify DB state using GetLeaderboard
			resLb, err := deps.Service.GetLeaderboard(ctx, guildID, "")
			require.NoError(t, err)
			require.True(t, resLb.IsSuccess())

			tagMap := make(map[string]int)
			for _, entry := range *resLb.Success {
				tagMap[string(entry.UserID)] = int(entry.TagNumber)
			}
			assert.Equal(t, 1, tagMap["user_b"])
			assert.Equal(t, 2, tagMap["user_a"])
			assert.Equal(t, 3, tagMap["user_c"])
			_, dInLb := tagMap["user_d"]
			assert.False(t, dInLb)

			// 4. Verify Points (tagless participants like user_d are excluded from point awards)
			assert.False(t, output.PointsSkipped)
			assert.Equal(t, seasonID, output.SeasonID)
			assert.Len(t, output.PointAwards, 3)

			// 5. Test Idempotency
			output2, err := deps.Service.ProcessRoundCommand(ctx, cmd)
			require.NoError(t, err)
			require.True(t, output2.WasIdempotent)
			assert.Equal(t, output.FinalParticipantTags, output2.FinalParticipantTags)

			// 6. Test Recalculation
			cmdRecalc := leaderboardservice.ProcessRoundCommand{
				GuildID: string(guildID),
				RoundID: roundID,
				Participants: []leaderboardservice.RoundParticipantInput{
					{MemberID: "user_a", FinishRank: 1}, // A won
					{MemberID: "user_b", FinishRank: 2}, // B second
					{MemberID: "user_c", FinishRank: 3},
					{MemberID: "user_d", FinishRank: 4},
				},
			}

			output3, err := deps.Service.ProcessRoundCommand(ctx, cmdRecalc)
			require.NoError(t, err)
			assert.False(t, output3.WasIdempotent)

			// Verify Tags updated again (Revert to A:1, B:2, C:3)
			resLb2, err := deps.Service.GetLeaderboard(ctx, guildID, "")
			require.NoError(t, err)
			tagMapRecalc := make(map[string]int)
			for _, entry := range *resLb2.Success {
				tagMapRecalc[string(entry.UserID)] = int(entry.TagNumber)
			}
			assert.Equal(t, 1, tagMapRecalc["user_a"])
			assert.Equal(t, 2, tagMapRecalc["user_b"])
			assert.Equal(t, 3, tagMapRecalc["user_c"])

			// Verify Points updated (count history)
			var histories []leaderboarddb.PointHistory
			err = deps.BunDB.NewSelect().
				Model(&histories).
				Where("round_id = ?", roundID).
				Order("member_id ASC").
				Scan(ctx)
			require.NoError(t, err)
			assert.Equal(t, 3, len(histories), "Should have 3 history entries (tagless user_d excluded)")

			// Verify specific point values for recalculation (A=1st, B=2nd...)
			// Since we don't have the exact point calculation logic mocked or known here easily without duplication,
			// we assume standard points: 1st > 2nd.
			// Map member to points
			pointMap := make(map[string]int)
			for _, h := range histories {
				pointMap[string(h.MemberID)] = h.Points
			}
			assert.True(t, pointMap["user_a"] > pointMap["user_b"], "User A (1st) should have more points than User B (2nd)")
			assert.True(t, pointMap["user_b"] > pointMap["user_c"], "User B (2nd) should have more points than User C (3rd)")

			// 7. Verify Tagless Participant Member Creation
			// Ensure user_d exists in league_members even though they have no tag
			exists, err := deps.BunDB.NewSelect().
				Table("league_members").
				Where("guild_id = ? AND member_id = ?", guildID, "user_d").
				Exists(ctx)
			require.NoError(t, err)
			assert.True(t, exists, "Tagless participant user_d should exist in league_members")
		})
	}
}

// TestProcessRound_TiedFinishRank verifies end-to-end behaviour when two
// participants share the same FinishRank. The tied players must receive equal
// points, and a player ranked below them must receive fewer points.
func TestProcessRound_TiedFinishRank(t *testing.T) {
	__codexTDCases := []struct {
		name string
	}{
		{name: "default"},
	}

	for _, __codexTDCase := range __codexTDCases {
		t.Run(__codexTDCase.name, func(t *testing.T) {
			deps := SetupTestLeaderboardService(t)
			defer deps.Cleanup()

			ctx := context.Background()
			guildID := sharedtypes.GuildID("test_guild_ties")
			seasonID := "season_ties_1"
			roundID := uuid.New()

			_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)

			// Start a season
			resSeason, err := deps.Service.StartNewSeason(ctx, guildID, seasonID, "Tie Test Season")
			require.NoError(t, err)
			require.True(t, resSeason.IsSuccess())

			// Seed three members with tags
			membersToSeed := []leaderboarddb.LeagueMember{
				{GuildID: string(guildID), MemberID: "tie_user_a", CurrentTag: ptr(1)},
				{GuildID: string(guildID), MemberID: "tie_user_b", CurrentTag: ptr(2)},
				{GuildID: string(guildID), MemberID: "tie_user_c", CurrentTag: ptr(3)},
			}
			_, err = deps.BunDB.NewInsert().Model(&membersToSeed).Exec(ctx)
			require.NoError(t, err)

			// tie_user_a and tie_user_b both finish 1st (tied); tie_user_c finishes 3rd
			cmd := leaderboardservice.ProcessRoundCommand{
				GuildID: string(guildID),
				RoundID: roundID,
				Participants: []leaderboardservice.RoundParticipantInput{
					{MemberID: "tie_user_a", FinishRank: 1},
					{MemberID: "tie_user_b", FinishRank: 1},
					{MemberID: "tie_user_c", FinishRank: 3},
				},
			}

			output, err := deps.Service.ProcessRoundCommand(ctx, cmd)
			require.NoError(t, err)
			require.NotNil(t, output)
			assert.False(t, output.PointsSkipped)
			assert.Len(t, output.PointAwards, 3)

			// Build points map from output
			pointMap := make(map[string]int)
			for _, award := range output.PointAwards {
				pointMap[award.MemberID] = award.Points
			}

			// Tied finishers must earn the same points
			assert.Equal(t, pointMap["tie_user_a"], pointMap["tie_user_b"],
				"tied finishers (rank 1) must earn equal points")

			// Both tied players must earn more than the player ranked below them
			assert.Greater(t, pointMap["tie_user_a"], pointMap["tie_user_c"],
				"tied rank-1 finisher must earn more than rank-3 finisher")
			assert.Greater(t, pointMap["tie_user_b"], pointMap["tie_user_c"],
				"tied rank-1 finisher must earn more than rank-3 finisher")

			// Confirm points persisted correctly in point_history
			var histories []leaderboarddb.PointHistory
			err = deps.BunDB.NewSelect().
				Model(&histories).
				Where("round_id = ?", roundID).
				Order("member_id ASC").
				Scan(ctx)
			require.NoError(t, err)
			assert.Len(t, histories, 3, "all three tagged participants should have point history entries")

			dbPointMap := make(map[string]int)
			for _, h := range histories {
				dbPointMap[string(h.MemberID)] = h.Points
			}
			assert.Equal(t, dbPointMap["tie_user_a"], dbPointMap["tie_user_b"],
				"persisted points must be equal for tied finishers")
		})
	}
}

// TestProcessRound_ResolveGuildID verifies that resolveGuildID converts a club UUID
// to its corresponding Discord snowflake before any leaderboard data is written.
// This covers the backfill bug where the PWA sends a club UUID instead of a Discord guild ID.
func TestProcessRound_ResolveGuildID(t *testing.T) {
	deps := SetupTestLeaderboardService(t)
	defer deps.Cleanup()

	ctx := context.Background()

	// A real-looking Discord snowflake and its matching club UUID.
	discordGuildID := sharedtypes.GuildID("888000111222333444")
	clubUUID := uuid.New()
	seasonID := "season_resolve_1"
	roundID := uuid.New()

	_ = testutils.CleanLeaderboardIntegrationTables(ctx, deps.BunDB)

	// Clubs is not in the standard cleanup list — delete explicitly.
	_, err := deps.BunDB.ExecContext(ctx, "DELETE FROM clubs WHERE discord_guild_id = ?", discordGuildID)
	require.NoError(t, err)

	// Map the club UUID to the Discord guild ID.
	_, err = deps.BunDB.ExecContext(ctx,
		"INSERT INTO clubs (uuid, name, discord_guild_id) VALUES (?, 'Test Club', ?)",
		clubUUID, discordGuildID)
	require.NoError(t, err)

	// Season and league members must live under the Discord guild ID (not the UUID).
	resSeason, err := deps.Service.StartNewSeason(ctx, discordGuildID, seasonID, "UUID Resolve Test Season")
	require.NoError(t, err)
	require.True(t, resSeason.IsSuccess())

	membersToSeed := []leaderboarddb.LeagueMember{
		{GuildID: string(discordGuildID), MemberID: "resolve_user_a", CurrentTag: ptr(1)},
		{GuildID: string(discordGuildID), MemberID: "resolve_user_b", CurrentTag: ptr(2)},
	}
	_, err = deps.BunDB.NewInsert().Model(&membersToSeed).Exec(ctx)
	require.NoError(t, err)

	// Submit the command using the club UUID as GuildID (simulates a PWA request).
	cmd := leaderboardservice.ProcessRoundCommand{
		GuildID: clubUUID.String(),
		RoundID: roundID,
		Participants: []leaderboardservice.RoundParticipantInput{
			{MemberID: "resolve_user_b", FinishRank: 1},
			{MemberID: "resolve_user_a", FinishRank: 2},
		},
	}

	output, err := deps.Service.ProcessRoundCommand(ctx, cmd)
	require.NoError(t, err)
	require.NotNil(t, output)

	// Tags must be assigned correctly — resolveGuildID converted the UUID.
	assert.Equal(t, 1, output.FinalParticipantTags["resolve_user_b"])
	assert.Equal(t, 2, output.FinalParticipantTags["resolve_user_a"])

	// Leaderboard data must be stored under the Discord snowflake.
	resLb, err := deps.Service.GetLeaderboard(ctx, discordGuildID, "")
	require.NoError(t, err)
	require.True(t, resLb.IsSuccess())

	tagMap := make(map[string]int)
	for _, entry := range *resLb.Success {
		tagMap[string(entry.UserID)] = int(entry.TagNumber)
	}
	assert.Equal(t, 1, tagMap["resolve_user_b"], "tag for resolve_user_b should be under Discord snowflake")
	assert.Equal(t, 2, tagMap["resolve_user_a"], "tag for resolve_user_a should be under Discord snowflake")

	// Nothing should be stored under the club UUID.
	resLbUUID, err := deps.Service.GetLeaderboard(ctx, sharedtypes.GuildID(clubUUID.String()), "")
	require.NoError(t, err)
	require.True(t, resLbUUID.IsSuccess(), "lookup under the club UUID should succeed but return no leaderboard entries")
	assert.Empty(t, *resLbUUID.Success, "no leaderboard data should exist under the club UUID")
}

func ptr(i int) *int {
	return &i
}
