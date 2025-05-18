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
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// InsertLeaderboard creates and inserts a leaderboard with the given data
func InsertLeaderboard(t *testing.T, db *bun.DB, data leaderboardtypes.LeaderboardData) (*leaderboarddb.Leaderboard, error) {
	t.Helper() // Mark this as a helper function
	leaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: data,
		IsActive:        true,
		UpdateSource:    leaderboarddb.ServiceUpdateSourceManual,
		UpdateID:        sharedtypes.RoundID(uuid.New()),
	}
	_, err := db.NewInsert().Model(leaderboard).Exec(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to insert leaderboard: %w", err)
	}
	return leaderboard, nil
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

// ParseRequestPayload extracts a BatchTagAssignmentRequestedPayload from a message
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
		if entry.TagNumber != nil {
			tagVal = fmt.Sprintf("%d", *entry.TagNumber)
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
		if entry.TagNumber != nil {
			result[entry.UserID] = *entry.TagNumber
		}
	}
	return result
}

// MergeLeaderboardWithAssignments combines initial leaderboard data with new assignments
func MergeLeaderboardWithAssignments(initialLeaderboard *leaderboarddb.Leaderboard, assignments []sharedevents.TagAssignmentInfo) map[sharedtypes.DiscordID]sharedtypes.TagNumber {
	result := ExtractLeaderboardDataMap(initialLeaderboard.LeaderboardData)

	for _, assignment := range assignments {
		if assignment.TagNumber >= 0 {
			result[assignment.UserID] = assignment.TagNumber
		}
	}

	return result
}
