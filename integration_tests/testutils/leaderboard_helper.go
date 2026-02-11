package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
)

const defaultTestGuildID = "test_guild"

// SetupLeaderboardWithEntries seeds league_members for tests and returns the seeded snapshot.
// Deprecated args are kept for compatibility with existing test call sites.
func SetupLeaderboardWithEntries(
	t *testing.T,
	db *bun.DB,
	entries []leaderboardtypes.LeaderboardEntry,
	_ bool,
	_ sharedtypes.RoundID,
) leaderboardtypes.LeaderboardData {
	t.Helper()
	ctx := context.Background()

	if _, err := db.NewDelete().
		Model((*leaderboarddb.LeagueMember)(nil)).
		Where("guild_id = ?", defaultTestGuildID).
		Exec(ctx); err != nil {
		t.Fatalf("failed to clear league members: %v", err)
	}

	members := make([]leaderboarddb.LeagueMember, 0, len(entries))
	for _, entry := range entries {
		tag := int(entry.TagNumber)
		members = append(members, leaderboarddb.LeagueMember{
			GuildID:    defaultTestGuildID,
			MemberID:   string(entry.UserID),
			CurrentTag: &tag,
		})
	}
	if len(members) > 0 {
		if _, err := db.NewInsert().Model(&members).Exec(ctx); err != nil {
			t.Fatalf("failed to seed league members: %v", err)
		}
	}

	return entries
}

// QueryLeaderboardData returns current tagged members from league_members for a guild.
func QueryLeaderboardData(
	t *testing.T,
	ctx context.Context,
	db *bun.DB,
	guildID sharedtypes.GuildID,
) (leaderboardtypes.LeaderboardData, error) {
	t.Helper()
	var members []leaderboarddb.LeagueMember
	err := db.NewSelect().
		Model(&members).
		Where("guild_id = ?", string(guildID)).
		Where("current_tag IS NOT NULL").
		OrderExpr("current_tag ASC").
		OrderExpr("member_id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query league members: %w", err)
	}

	data := make(leaderboardtypes.LeaderboardData, 0, len(members))
	for _, m := range members {
		if m.CurrentTag == nil || *m.CurrentTag <= 0 {
			continue
		}
		data = append(data, leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(m.MemberID),
			TagNumber: sharedtypes.TagNumber(*m.CurrentTag),
		})
	}
	return data, nil
}

// ParsePayload extracts a payload struct from a message.
func ParsePayload[T any](msg *message.Message) (*T, error) {
	var payload T
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return &payload, nil
}

// DebugLeaderboardData prints leaderboard data for troubleshooting.
func DebugLeaderboardData(t *testing.T, label string, data leaderboardtypes.LeaderboardData) {
	t.Helper()
	t.Logf("--- %s ---", label)
	for i, entry := range data {
		tagVal := "nil"
		if entry.TagNumber != 0 {
			tagVal = fmt.Sprintf("%d", entry.TagNumber)
		}
		t.Logf("Entry %d: UserID=%s, TagNumber=%s", i, entry.UserID, tagVal)
	}
	t.Logf("------------------")
}

// ValidateSuccessResponse checks that a response has the expected properties.
func ValidateSuccessResponse(t *testing.T, requestPayload *sharedevents.BatchTagAssignmentRequestedPayloadV1, responsePayload *leaderboardevents.LeaderboardBatchTagAssignedPayloadV1) {
	t.Helper()
	if responsePayload.RequestingUserID != requestPayload.RequestingUserID {
		t.Errorf("Success payload RequestingUserID mismatch: expected %q, got %q",
			requestPayload.RequestingUserID, responsePayload.RequestingUserID)
	}
	if responsePayload.BatchID != requestPayload.BatchID {
		t.Errorf("Success payload BatchID mismatch: expected %q, got %q",
			requestPayload.BatchID, responsePayload.BatchID)
	}
}

// ValidateLeaderboardData compares leaderboard data against expected assignments.
func ValidateLeaderboardData(t *testing.T, expectedData map[sharedtypes.DiscordID]sharedtypes.TagNumber, actualData map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
	t.Helper()
	if len(actualData) != len(expectedData) {
		t.Errorf("Expected %d entries in leaderboard, got %d. Actual: %+v", len(expectedData), len(actualData), actualData)
		return
	}

	for userID, expectedTag := range expectedData {
		actualTag, ok := actualData[userID]
		if !ok {
			t.Errorf("Expected user %q not found in leaderboard data", userID)
		} else if actualTag != expectedTag {
			t.Errorf("Tag mismatch for user %q: expected %d, got %d", userID, expectedTag, actualTag)
		}
	}
}

// ExtractLeaderboardDataMap converts leaderboard data to a map for easier comparison.
func ExtractLeaderboardDataMap(leaderboardData leaderboardtypes.LeaderboardData) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	result := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, entry := range leaderboardData {
		if entry.TagNumber != 0 {
			result[entry.UserID] = entry.TagNumber
		}
	}
	return result
}

// MergeLeaderboardWithAssignments combines initial leaderboard data with new assignments.
func MergeLeaderboardWithAssignments(initial leaderboardtypes.LeaderboardData, assignments []sharedevents.TagAssignmentInfoV1) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	result := ExtractLeaderboardDataMap(initial)
	for _, assignment := range assignments {
		if assignment.TagNumber >= 0 {
			result[assignment.UserID] = assignment.TagNumber
		}
	}
	return result
}

// SortedLeaderboard returns a stable ordering for direct equality checks in tests.
func SortedLeaderboard(data leaderboardtypes.LeaderboardData) leaderboardtypes.LeaderboardData {
	out := append(leaderboardtypes.LeaderboardData(nil), data...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].TagNumber != out[j].TagNumber {
			return out[i].TagNumber < out[j].TagNumber
		}
		return out[i].UserID < out[j].UserID
	})
	return out
}
