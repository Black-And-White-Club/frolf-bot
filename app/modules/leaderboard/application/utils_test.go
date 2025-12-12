package leaderboardservice

import (
	// Import context for the serviceWrapper
	"fmt"
	"log/slog" // Use standard library slog for a basic logger
	"os"       // Import os for logger output
	"slices"
	"strconv"
	"strings"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// Helper function to sort LeaderboardData by TagNumber
func sortLeaderboardData(data leaderboardtypes.LeaderboardData) {
	slices.SortFunc(data, func(a, b leaderboardtypes.LeaderboardEntry) int {
		// Assuming 0 is the zero value for TagNumber and should be sorted appropriately.
		// Adjust logic if 0 has a special sorting requirement (e.g., always last).
		if a.TagNumber != 0 && b.TagNumber != 0 {
			return int(a.TagNumber - b.TagNumber)
		}
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0
		}
		if a.TagNumber == 0 {
			return 1 // Place 0 after non-zero tags
		}
		return -1 // Place non-zero tags before 0
	})
}

// func compareLeaderboardEntries(t *testing.T, expected, actual []leaderboardtypes.LeaderboardEntry, testName string) {
// 	if len(expected) != len(actual) {
// 		t.Errorf("Test '%s': Expected leaderboard length %d, Got %d", testName, len(expected), len(actual))
// 		return
// 	}

// 	sortLeaderboardEntries(expected)
// 	sortLeaderboardEntries(actual)

// 	for i := range expected {
// 		if !reflect.DeepEqual(expected[i], actual[i]) {
// 			t.Errorf("Test '%s': Entry mismatch at index %d: Expected %+v, Got %+v", testName, i, expected[i], actual[i])
// 		}
// 	}
// }

// func sortLeaderboardEntries(entries []leaderboardtypes.LeaderboardEntry) {
// 	slices.SortFunc(entries, func(a, b leaderboardtypes.LeaderboardEntry) int {
// 		if a.TagNumber != 0 && b.TagNumber != 0 {
// 			return int(a.TagNumber - b.TagNumber)
// 		}
// 		if a.TagNumber == 0 && b.TagNumber == 0 {
// 			return 0
// 		}
// 		if a.TagNumber == 0 {
// 			return 1
// 		}
// 		return -1
// 	})
// }

func TestGenerateUpdatedLeaderboardData(t *testing.T) {
	tag1 := sharedtypes.TagNumber(1)
	tag5 := sharedtypes.TagNumber(5)
	tag18 := sharedtypes.TagNumber(18)
	tag20 := sharedtypes.TagNumber(20)

	testLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	tests := []struct {
		name                    string
		currentLeaderboard      *leaderboarddb.Leaderboard
		sortedParticipantTags   []string
		expectedLeaderboardData leaderboardtypes.LeaderboardData
		expectError             bool
	}{
		{
			name: "Basic redistribution with existing users",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: tag1},
					{UserID: "user2", TagNumber: tag5},
					{UserID: "user3", TagNumber: tag18},
					{UserID: "user4", TagNumber: tag20},
				},
			},
			sortedParticipantTags: []string{
				"18:user3", // user3 gets tag 18
				"5:user2",  // user2 gets tag 5
				"1:user1",  // user1 gets tag 1
			},
			// Expected data based on sortedParticipantTags and including non-participant user4
			expectedLeaderboardData: []leaderboardtypes.LeaderboardEntry{
				{UserID: "user3", TagNumber: 18}, // Should have tag 18 from input
				{UserID: "user2", TagNumber: 5},  // Should have tag 5 from input
				{UserID: "user1", TagNumber: 1},  // Should have tag 1 from input
				{UserID: "user4", TagNumber: 20}, // Non-participant keeps original tag
			},
			expectError: false,
		},
		{
			name: "Scenario with gaps in tag numbers and non-participants",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "userA", TagNumber: tag1},
					{UserID: "userB", TagNumber: tag5},
					{UserID: "userC", TagNumber: tag18},
					{UserID: "userD", TagNumber: tag20},
				},
			},
			sortedParticipantTags: []string{
				"18:userC", // userC gets tag 18
				"5:userB",  // userB gets tag 5
			},
			// Expected data based on sortedParticipantTags and including non-participants userA and userD
			expectedLeaderboardData: []leaderboardtypes.LeaderboardEntry{
				{UserID: "userC", TagNumber: 18}, // Should have tag 18 from input
				{UserID: "userB", TagNumber: 5},  // Should have tag 5 from input
				{UserID: "userA", TagNumber: 1},  // Non-participant keeps original tag
				{UserID: "userD", TagNumber: 20}, // Non-participant keeps original tag
			},
			expectError: false,
		},
		{
			name: "Empty participant tags",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: tag1},
					{UserID: "user2", TagNumber: tag5},
				},
			},
			sortedParticipantTags:   []string{},
			expectedLeaderboardData: nil,
			expectError:             true,
		},
		{
			name: "Invalid tag format in participant tags",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: tag1},
				},
			},
			sortedParticipantTags: []string{
				"invalid-tag-user",
			},
			expectedLeaderboardData: nil,
			expectError:             true,
		},
		{
			name: "Invalid tag number in participant tags",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: tag1},
				},
			},
			sortedParticipantTags: []string{
				"abc:user1",
			},
			expectedLeaderboardData: nil,
			expectError:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &LeaderboardService{
				logger: testLogger,
			}

			updatedLeaderboardData, err := service.GenerateUpdatedLeaderboard(tt.currentLeaderboard.LeaderboardData, tt.sortedParticipantTags)

			if (err != nil) != tt.expectError {
				t.Errorf("Test '%s': Expected error: %v, Got error: %v", tt.name, tt.expectError, err)
				return
			}
			if tt.expectError {
				return
			}

			if len(updatedLeaderboardData) != len(tt.expectedLeaderboardData) {
				t.Fatalf("Test '%s': Expected %d leaderboard entries, got %d", tt.name, len(tt.expectedLeaderboardData), len(updatedLeaderboardData))
			}

			// Sort both actual and expected before comparison
			sortLeaderboardData(updatedLeaderboardData)
			sortLeaderboardData(tt.expectedLeaderboardData)

			for i := range updatedLeaderboardData {
				actual := updatedLeaderboardData[i]
				expected := tt.expectedLeaderboardData[i]

				if actual.UserID != expected.UserID || actual.TagNumber != expected.TagNumber {
					t.Errorf("Test '%s': Entry mismatch at index %d: Expected %+v, Got %+v", tt.name, i, expected, actual)
				}
			}
		})
	}
}

func TestGenerateUpdatedLeaderboardData_LargeNumberOfParticipants(t *testing.T) {
	testLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	currentLeaderboardData := createBenchmarkLeaderboardData(50)

	sortedParticipantTags := make([]string, 40)
	for i := range sortedParticipantTags {
		originalTag := i + 1
		userID := fmt.Sprintf("existinguser%d", i)
		// Using the original tag in the input string as per the function's current logic
		sortedParticipantTags[i] = fmt.Sprintf("%d:%s", originalTag, userID)
	}

	service := &LeaderboardService{
		logger: testLogger,
	}

	updatedLeaderboardData, err := service.GenerateUpdatedLeaderboard(currentLeaderboardData, sortedParticipantTags)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// The number of entries should be the number of participants + number of non-participants
	expectedLen := len(sortedParticipantTags) + (len(currentLeaderboardData) - len(sortedParticipantTags)) // Assuming all users in currentData are either participants or non-participants
	if len(updatedLeaderboardData) != expectedLen {
		t.Errorf("Expected %d entries, got %d", expectedLen, len(updatedLeaderboardData))
	}

	participatingUsersMap := make(map[sharedtypes.DiscordID]bool)
	for _, tagUserStr := range sortedParticipantTags {
		parts := strings.Split(tagUserStr, ":")
		if len(parts) == 2 {
			participatingUsersMap[sharedtypes.DiscordID(parts[1])] = true
		}
	}

	expectedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, expectedLen)

	// Add participants with tags from sortedParticipantTags
	for _, tagUserStr := range sortedParticipantTags {
		parts := strings.Split(tagUserStr, ":")
		if len(parts) == 2 {
			tag, _ := strconv.Atoi(parts[0])
			userID := sharedtypes.DiscordID(parts[1])
			expectedLeaderboardData = append(expectedLeaderboardData, leaderboardtypes.LeaderboardEntry{
				UserID:    userID,
				TagNumber: sharedtypes.TagNumber(tag), // Use the tag from the input string
			})
		}
	}

	// Add non-participants with their original tags
	for _, originalEntry := range currentLeaderboardData {
		if !participatingUsersMap[originalEntry.UserID] {
			expectedLeaderboardData = append(expectedLeaderboardData, originalEntry)
		}
	}

	sortLeaderboardData(updatedLeaderboardData)
	sortLeaderboardData(expectedLeaderboardData)

	for i := range updatedLeaderboardData {
		actual := updatedLeaderboardData[i]
		expected := expectedLeaderboardData[i]

		if actual.UserID != expected.UserID || actual.TagNumber != expected.TagNumber {
			t.Errorf("Entry mismatch at index %d: Expected %+v, Got %+v", i, expected, actual)
		}
	}
}
