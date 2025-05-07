package leaderboardservice

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

func TestGenerateUpdatedLeaderboard(t *testing.T) {
	tag1 := sharedtypes.TagNumber(1)
	tag2 := sharedtypes.TagNumber(5)
	tag3 := sharedtypes.TagNumber(18)
	tag4 := sharedtypes.TagNumber(20)
	tests := []struct {
		name                   string
		currentLeaderboard     *leaderboarddb.Leaderboard
		sortedParticipantTags  []string
		expectedTagAssignments map[sharedtypes.DiscordID]sharedtypes.TagNumber
		expectedRemainingUsers int
	}{
		{
			name: "Basic redistribution with existing users",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: &tag1},
					{UserID: "user2", TagNumber: &tag2},
					{UserID: "user3", TagNumber: &tag3},
					{UserID: "user4", TagNumber: &tag4},
				},
			},
			sortedParticipantTags: []string{
				"18:user3", // best performer
				"5:user2",  // second
				"1:user1",  // worst performer
			},
			expectedTagAssignments: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
				"user3": 1,  // best performer gets lowest tag
				"user2": 5,  // keeps original tag
				"user1": 18, // worst performer gets highest tag
			},
			expectedRemainingUsers: 1, // user4 should remain unchanged
		},
		{
			name: "Scenario with gaps in tag numbers",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: &tag1},
					{UserID: "user2", TagNumber: &tag2},
					{UserID: "user3", TagNumber: &tag3},
				},
			},
			sortedParticipantTags: []string{
				"10:user3",
				"1:user1",
			},
			expectedTagAssignments: map[sharedtypes.DiscordID]sharedtypes.TagNumber{
				"user3": 1,  // best performer gets lowest tag
				"user1": 10, // worst performer gets highest tag
			},
			expectedRemainingUsers: 1, // user2 should remain unchanged
		},
		{
			name: "Empty participant tags",
			currentLeaderboard: &leaderboarddb.Leaderboard{
				LeaderboardData: []leaderboardtypes.LeaderboardEntry{
					{UserID: "user1", TagNumber: &tag1},
					{UserID: "user2", TagNumber: &tag2},
				},
			},
			sortedParticipantTags:  []string{},
			expectedTagAssignments: map[sharedtypes.DiscordID]sharedtypes.TagNumber{},
			expectedRemainingUsers: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &LeaderboardService{} // You might need to mock dependencies if required

			// Handle empty participant tags case
			if len(tt.sortedParticipantTags) == 0 {
				_, err := service.GenerateUpdatedLeaderboard(tt.currentLeaderboard, tt.sortedParticipantTags)
				if err == nil {
					t.Errorf("Expected error for empty participant tags, got nil")
				}
				return
			}

			updatedLeaderboard, err := service.GenerateUpdatedLeaderboard(tt.currentLeaderboard, tt.sortedParticipantTags)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if updatedLeaderboard == nil {
				t.Fatal("Updated leaderboard is nil")
			}

			// Verify tag assignments
			for _, tag := range tt.sortedParticipantTags {
				parts := strings.Split(tag, ":")
				userID := parts[1]

				// Find the user in the updated leaderboard
				found := false
				for _, entry := range updatedLeaderboard.LeaderboardData {
					if string(entry.UserID) == userID {
						expectedTag := tt.expectedTagAssignments[sharedtypes.DiscordID(userID)]
						if entry.TagNumber != &expectedTag {
							t.Errorf("For user %s, expected tag %d, got %d",
								userID, expectedTag, entry.TagNumber)
						}
						found = true
						break
					}
				}
				if !found {
					t.Errorf("User %s not found in updated leaderboard", userID)
				}
			}

			// Verify remaining users
			nonParticipatingUsers := 0
			for _, entry := range updatedLeaderboard.LeaderboardData {
				isParticipating := false
				for _, tag := range tt.sortedParticipantTags {
					parts := strings.Split(tag, ":")
					if string(entry.UserID) == parts[1] {
						isParticipating = true
						break
					}
				}
				if !isParticipating {
					nonParticipatingUsers++
				}
			}
			if nonParticipatingUsers != tt.expectedRemainingUsers {
				t.Errorf("Expected %d non-participating users, got %d",
					tt.expectedRemainingUsers, nonParticipatingUsers)
			}
		})
	}
}

func TestGenerateUpdatedLeaderboard_LargeNumberOfParticipants(t *testing.T) {
	// Create a large initial leaderboard
	currentLeaderboard := &leaderboarddb.Leaderboard{
		LeaderboardData: make([]leaderboardtypes.LeaderboardEntry, 50),
	}
	for i := range currentLeaderboard.LeaderboardData {
		tag := sharedtypes.TagNumber(i + 1)
		currentLeaderboard.LeaderboardData[i] = leaderboardtypes.LeaderboardEntry{
			UserID:    sharedtypes.DiscordID(fmt.Sprintf("existinguser%d", i)),
			TagNumber: &tag,
		}
	}

	// Create sorted participant tags using only existing users
	sortedParticipantTags := make([]string, 40)
	for i := range sortedParticipantTags {
		sortedParticipantTags[i] = fmt.Sprintf("%d:existinguser%d", i+1, i)
	}

	// Run the function
	service := &LeaderboardService{}
	updatedLeaderboard, err := service.GenerateUpdatedLeaderboard(currentLeaderboard, sortedParticipantTags)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(updatedLeaderboard.LeaderboardData) != 50 {
		t.Errorf("Expected 50 entries, got %d", len(updatedLeaderboard.LeaderboardData))
	}

	// Verify participants got new tags in order
	participantTags := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)
	for _, entry := range updatedLeaderboard.LeaderboardData {
		if strings.HasPrefix(string(entry.UserID), "existinguser") {
			participantTags[entry.UserID] = *entry.TagNumber
		}
	}

	// Verify tags are assigned in ascending order for participants
	assignedTags := make([]sharedtypes.TagNumber, 0, len(participantTags))
	for range sortedParticipantTags {
		found := false
		for userID, tag := range participantTags {
			if !slices.Contains(assignedTags, tag) {
				assignedTags = append(assignedTags, tag)
				delete(participantTags, userID)
				found = true
				break
			}
		}
		if !found {
			t.Error("Should find an unassigned tag")
		}
	}

	slices.Sort(assignedTags)
	for i := 1; i < len(assignedTags); i++ {
		if assignedTags[i-1] >= assignedTags[i] {
			t.Errorf("Tags should be assigned in strictly ascending order")
		}
	}

	// Verify existing users remain unchanged
	for _, entry := range currentLeaderboard.LeaderboardData {
		found := false
		for _, updatedEntry := range updatedLeaderboard.LeaderboardData {
			if entry.UserID == updatedEntry.UserID {
				if entry.TagNumber != updatedEntry.TagNumber {
					t.Errorf("Existing user %s tag changed from %d to %d",
						entry.UserID, entry.TagNumber, updatedEntry.TagNumber)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Existing user %s not found in leaderboard", entry.UserID)
		}
	}
}
