package leaderboarddomain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	leaderboardevents "github.com/Black-And-White-Club/frolf-bot-shared/events/leaderboard"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GenerateUpdatedLeaderboard generates the updated leaderboard data based on the current leaderboard and the new tag order from the round.
func GenerateUpdatedLeaderboard(currentLeaderboard *leaderboarddb.Leaderboard, newTagOrder []string) leaderboardtypes.LeaderboardData {
	updatedLeaderboard := make(leaderboardtypes.LeaderboardData)
	currentEntries := currentLeaderboard.LeaderboardData

	// 1. Create a map of tag number (as string) to Discord ID from the new tag order.
	newTagMap := createNewTagMap(newTagOrder)

	// 2. Iterate through the current leaderboard entries.
	usedTags := make(map[int]bool)
	nextAvailableTag := 1

	// Sort the current leaderboard entries by tag number (rank).
	sortedCurrentEntries := sortCurrentEntries(currentEntries)

	// Iterate through the sorted current entries, adding them to the updated leaderboard if they exist in the new tag map.
	for _, entry := range sortedCurrentEntries {
		tag := entry.TagNumber
		if _, exists := newTagMap[*tag]; exists {
			for {
				if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
					updatedLeaderboard[nextAvailableTag] = string(entry.UserID)
					usedTags[*tag] = true
					nextAvailableTag++
					break
				}
				nextAvailableTag++
			}
		}
	}

	// 3. Add any new tags from the round results that were not in the current leaderboard.
	for tag, userID := range newTagMap {
		if !usedTags[tag] {
			for {
				if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
					updatedLeaderboard[nextAvailableTag] = userID
					nextAvailableTag++
					break
				}
				nextAvailableTag++
			}
		}
	}

	// 4. Add back any tags that were in the current leaderboard but not in the new tag order, preserving their relative order.
	for _, entry := range sortedCurrentEntries {
		tag := entry.TagNumber
		if _, exists := updatedLeaderboard[*tag]; !exists {
			updatedLeaderboard[*tag] = string(entry.UserID)
		}
	}

	return updatedLeaderboard
}

// FindTagByUserID is a helper function to find the tag associated with a Discord ID in the leaderboard data.
func FindTagByUserID(leaderboard *leaderboarddb.Leaderboard, userID leaderboardtypes.UserID) (int, bool) {
	for tag, id := range leaderboard.LeaderboardData {
		if leaderboardtypes.UserID(id) == userID {
			return tag, true
		}
	}
	return 0, false
}

// --- Helper functions for generateUpdatedLeaderboard ---

func parseTagUserIDPair(tagUserIDPair string) (int, string, error) {
	parts := strings.Split(tagUserIDPair, ":")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid tag-UserID pair format: %s", tagUserIDPair)
	}
	tag, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid tag number: %s", parts[0])
	}
	return tag, parts[1], nil
}

func createNewTagMap(newTagOrder []string) map[int]string {
	newTagMap := make(map[int]string)
	for _, tagUserIDPair := range newTagOrder {
		tag, userID, err := parseTagUserIDPair(tagUserIDPair)
		if err == nil { // Only add to map if parsing was successful
			newTagMap[tag] = userID
		}
	}
	return newTagMap
}

func sortCurrentEntries(currentEntries map[int]string) []leaderboardevents.LeaderboardEntry {
	var sortedCurrentEntries []leaderboardevents.LeaderboardEntry
	for tag, userID := range currentEntries {
		sortedCurrentEntries = append(sortedCurrentEntries, leaderboardevents.LeaderboardEntry{
			TagNumber: &tag,
			UserID:    leaderboardtypes.UserID(userID),
		})
	}
	sort.Slice(sortedCurrentEntries, func(i, j int) bool {
		// Sort by tag number (ascending).
		tagI := sortedCurrentEntries[i].TagNumber
		tagJ := sortedCurrentEntries[j].TagNumber
		return *tagI < *tagJ
	})
	return sortedCurrentEntries
}

// InsertTag inserts a new tag/Discord ID into the correct position in the leaderboard, maintaining order.
func InsertTag(currentLeaderboard *leaderboarddb.Leaderboard, tagNumber leaderboardtypes.Tag, userID leaderboardtypes.UserID) leaderboardtypes.LeaderboardData {
	updatedLeaderboard := make(leaderboardtypes.LeaderboardData)
	inserted := false

	// Sort current leaderboard entries for ordered insertion
	sortedCurrentEntries := sortCurrentEntries(currentLeaderboard.LeaderboardData)

	// Find the correct position to insert the new tag
	nextAvailableTag := 1
	for _, entry := range sortedCurrentEntries {
		currentTag := entry.TagNumber

		if !inserted && int(tagNumber) < *currentTag {
			updatedLeaderboard[int(tagNumber)] = string(userID)
			inserted = true
		}

		for {
			if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
				updatedLeaderboard[nextAvailableTag] = string(entry.UserID)
				nextAvailableTag++
				break
			}
			nextAvailableTag++
		}
	}

	// If not inserted yet, assign the tag at the end
	if !inserted {
		for {
			if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
				updatedLeaderboard[nextAvailableTag] = string(userID)
				break
			}
			nextAvailableTag++
		}
	}

	return updatedLeaderboard
}
