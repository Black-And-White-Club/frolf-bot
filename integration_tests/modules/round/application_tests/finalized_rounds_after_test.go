package roundintegrationtests

import (
	"context"
	"testing"
	"time"

	roundtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/round"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	"github.com/Black-And-White-Club/frolf-bot/integration_tests/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetFinalizedRoundsAfter verifies the SQL filter in GetFinalizedRoundsAfter:
//   - only rounds with finalized=true AND start_time > cutoff are returned
//   - rounds at or before the cutoff are excluded
//   - non-finalized rounds are excluded regardless of start_time
//
// This covers the backfill bug fix where start_time was previously stored as
// JSONB text, causing SQLSTATE=22P02 when compared as a native timestamptz.
func TestGetFinalizedRoundsAfter(t *testing.T) {
	deps := SetupTestRoundService(t)
	defer deps.Cleanup()

	ctx := context.Background()
	guildID := sharedtypes.GuildID("test-guild-finalized-after")

	_ = testutils.CleanRoundIntegrationTables(ctx, deps.BunDB)

	generator := testutils.NewTestDataGenerator()
	cutoff := time.Now().Add(-2 * time.Hour)

	// r1: finalized, start_time one hour before cutoff — must NOT be returned.
	r1 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
		StartTime: testutils.StartTimePtr(cutoff.Add(-time.Hour)),
	})
	r1.GuildID = guildID
	r1.Finalized = roundtypes.Finalized(true)
	r1.State = roundtypes.RoundStateFinalized

	// r2: finalized, start_time exactly at cutoff — SQL uses >, so NOT returned.
	r2 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
		StartTime: testutils.StartTimePtr(cutoff),
	})
	r2.GuildID = guildID
	r2.Finalized = roundtypes.Finalized(true)
	r2.State = roundtypes.RoundStateFinalized

	// r3: finalized, start_time one hour after cutoff — MUST be returned.
	r3 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
		StartTime: testutils.StartTimePtr(cutoff.Add(time.Hour)),
	})
	r3.GuildID = guildID
	r3.Finalized = roundtypes.Finalized(true)
	r3.State = roundtypes.RoundStateFinalized

	// r4: finalized, start_time two hours after cutoff — MUST be returned.
	r4 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
		StartTime: testutils.StartTimePtr(cutoff.Add(2 * time.Hour)),
	})
	r4.GuildID = guildID
	r4.Finalized = roundtypes.Finalized(true)
	r4.State = roundtypes.RoundStateFinalized

	// r5: NOT finalized, start_time after cutoff — must NOT be returned.
	r5 := generator.GenerateRoundWithConstraints(testutils.RoundOptions{
		StartTime: testutils.StartTimePtr(cutoff.Add(3 * time.Hour)),
	})
	r5.GuildID = guildID
	r5.Finalized = roundtypes.Finalized(false)
	r5.State = roundtypes.RoundStateUpcoming

	for _, round := range []roundtypes.Round{r1, r2, r3, r4, r5} {
		r := round
		require.NoError(t, deps.DB.CreateRound(ctx, deps.BunDB, guildID, &r))
	}

	returned, err := deps.Service.GetFinalizedRoundsAfter(ctx, guildID, cutoff)
	require.NoError(t, err)

	assert.Len(t, returned, 2, "only finalized rounds with start_time strictly greater than cutoff should be returned")

	returnedIDs := make(map[sharedtypes.RoundID]bool, len(returned))
	for _, r := range returned {
		returnedIDs[r.ID] = true
	}

	assert.True(t, returnedIDs[r3.ID], "r3 (1h after cutoff, finalized) must be returned")
	assert.True(t, returnedIDs[r4.ID], "r4 (2h after cutoff, finalized) must be returned")
	assert.False(t, returnedIDs[r1.ID], "r1 (before cutoff) must not be returned")
	assert.False(t, returnedIDs[r2.ID], "r2 (at cutoff, not strictly greater) must not be returned")
	assert.False(t, returnedIDs[r5.ID], "r5 (not finalized) must not be returned")
}
