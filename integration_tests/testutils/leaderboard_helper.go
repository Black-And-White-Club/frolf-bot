package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	sharedevents "github.com/Black-And-White-Club/frolf-bot-shared/events/shared"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/uptrace/bun"
)

// InsertLeaderboard creates and inserts a leaderboard with the given data and allows custom IsActive and UpdateID.
func InsertLeaderboard(t *testing.T, db *bun.DB, data leaderboardtypes.LeaderboardData, isActive bool, updateID sharedtypes.RoundID) (*leaderboarddb.Leaderboard, error) {
	t.Helper()
	leaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: data,
		IsActive:        isActive,
		UpdateSource:    sharedtypes.ServiceUpdateSourceManual,
		UpdateID:        updateID,
		GuildID:         sharedtypes.GuildID("test_guild"),
	}
	_, err := db.NewInsert().Model(leaderboard).Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to insert leaderboard: %w", err)
	}
	return leaderboard, nil
}

// SetupLeaderboardWithEntries inserts a leaderboard with the specified entries and returns the refreshed leaderboard.
func SetupLeaderboardWithEntries(t *testing.T, db *bun.DB, entries []leaderboardtypes.LeaderboardEntry, isActive bool, roundID sharedtypes.RoundID) *leaderboarddb.Leaderboard {
	t.Helper()
	leaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(entries))
	for _, entry := range entries {
		leaderboardData = append(leaderboardData, entry)
	}
	leaderboard, err := InsertLeaderboard(t, db, leaderboardData, isActive, roundID)
	if err != nil {
		t.Fatalf("Failed to insert leaderboard: %v", err)
	}
	updatedLeaderboard := &leaderboarddb.Leaderboard{}
	err = db.NewSelect().
		Model(updatedLeaderboard).
		Where("id = ?", leaderboard.ID).
		Scan(context.Background())
	if err != nil {
		t.Fatalf("Failed to refresh leaderboard from database: %v", err)
	}
	return updatedLeaderboard
}

// AssertLeaderboardState validates the leaderboard state in the database.
func AssertLeaderboardState(t *testing.T, db *bun.DB, expectedLeaderboard *leaderboarddb.Leaderboard, expectedCount int, expectedActive bool) {
	t.Helper()
	leaderboards, err := QueryLeaderboards(t, context.Background(), db)
	if err != nil {
		t.Fatalf("Failed to query leaderboards: %v", err)
	}

	activeCount := 0
	var activeLeaderboard *leaderboarddb.Leaderboard
	for _, lb := range leaderboards {
		if lb.IsActive {
			activeCount++
			activeLeaderboard = &lb
		}
	}

	if expectedCount > 0 && expectedActive {
		if activeCount != expectedCount {
			t.Errorf("Expected %d active leaderboards, got %d", expectedCount, activeCount)
		}
		if expectedLeaderboard != nil && activeLeaderboard != nil && expectedLeaderboard.ID != activeLeaderboard.ID {
			t.Errorf("Active leaderboard ID mismatch: expected %d, got %d", expectedLeaderboard.ID, activeLeaderboard.ID)
		}
	} else if expectedActive && activeCount == 0 {
		t.Error("Expected an active leaderboard but found none")
	} else if !expectedActive && activeCount > 0 {
		t.Errorf("Expected no active leaderboards but found %d", activeCount)
	}
}

// QueryLeaderboards retrieves all leaderboards from the database
func QueryLeaderboards(t *testing.T, ctx context.Context, db *bun.DB) ([]leaderboarddb.Leaderboard, error) {
	t.Helper() // Mark this as a helper function
	var leaderboards []leaderboarddb.Leaderboard
	err := db.NewSelect().
		Model(&leaderboards).
		Order("id ASC").
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboards: %w", err)
	}
	return leaderboards, nil
}

// ParseRequestPayload extracts a Payload Struct from a message
func ParsePayload[T any](msg *message.Message) (*T, error) {
	var payload T
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return &payload, nil
}

// DebugLeaderboardData prints leaderboard data for troubleshooting
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

// ValidateSuccessResponse checks that a response has the expected properties
func ValidateSuccessResponse(t *testing.T, requestPayload *sharedevents.BatchTagAssignmentRequestedPayload, responsePayload *leaderboardevents.BatchTagAssignedPayload) {
	t.Helper() // Mark this as a helper function
	if responsePayload.RequestingUserID != requestPayload.RequestingUserID {
		t.Errorf("Success payload RequestingUserID mismatch: expected %q, got %q",
			requestPayload.RequestingUserID, responsePayload.RequestingUserID)
	}
	if responsePayload.BatchID != requestPayload.BatchID {
		t.Errorf("Success payload BatchID mismatch: expected %q, got %q",
			requestPayload.BatchID, responsePayload.BatchID)
	}
}

// ValidateLeaderboardData compares leaderboard data against expected assignments
func ValidateLeaderboardData(t *testing.T, expectedData map[sharedtypes.DiscordID]sharedtypes.TagNumber, actualData map[sharedtypes.DiscordID]sharedtypes.TagNumber) {
	t.Helper() // Mark this as a helper function
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

// ExtractLeaderboardDataMap converts leaderboard data to a map for easier comparison
func ExtractLeaderboardDataMap(leaderboardData leaderboardtypes.LeaderboardData) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	result := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, entry := range leaderboardData {
		if entry.TagNumber != 0 {
			result[entry.UserID] = entry.TagNumber
		}
	}
	return result
}

// MergeLeaderboardWithAssignments combines initial leaderboard data with new assignments
func MergeLeaderboardWithAssignments(initialLeaderboard *leaderboarddb.Leaderboard, assignments []sharedevents.TagAssignmentInfoV1) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	result := ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)

	for _, assignment := range assignments {
		if assignment.TagNumber >= 0 {
			result[assignment.UserID] = assignment.TagNumber
		}
	}

	return result
}
