package leaderboarddomain

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/events"
	leaderboardtypes "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/domain/types"
	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GenerateUpdatedLeaderboard generates the updated leaderboard data based on the current leaderboard and the new tag order from the round.
func GenerateUpdatedLeaderboard(currentLeaderboard *leaderboarddb.Leaderboard, newTagOrder []string) leaderboardtypes.LeaderboardData {
	updatedLeaderboard := make(leaderboardtypes.LeaderboardData)
	currentEntries := currentLeaderboard.LeaderboardData

	// 1. Create a map of tag number (as string) to Discord ID from the new tag order.
	newTagMap := createNewTagMap(newTagOrder)

	// 2. Iterate through the current leaderboard entries.
	usedTags := make(map[string]bool)
	nextAvailableTag := 1

	// Sort the current leaderboard entries by tag number (rank).
	sortedCurrentEntries := sortCurrentEntries(currentEntries)

	// Iterate through the sorted current entries, adding them to the updated leaderboard if they exist in the new tag map.
	for _, entry := range sortedCurrentEntries {
		tag := entry.TagNumber
		if _, exists := newTagMap[tag]; exists {
			for {
				if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
					updatedLeaderboard[nextAvailableTag] = string(entry.DiscordID)
					usedTags[tag] = true
					nextAvailableTag++
					break
				}
				nextAvailableTag++
			}
		}
	}

	// 3. Add any new tags from the round results that were not in the current leaderboard.
	for tag, discordID := range newTagMap {
		if !usedTags[tag] {
			for {
				if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
					updatedLeaderboard[nextAvailableTag] = discordID
					nextAvailableTag++
					break
				}
				nextAvailableTag++
			}
		}
	}

	// 4. Add back any tags that were in the current leaderboard but not in the new tag order, preserving their relative order.
	for _, entry := range sortedCurrentEntries {
		tag, _ := strconv.Atoi(entry.TagNumber)
		if _, exists := updatedLeaderboard[tag]; !exists {
			updatedLeaderboard[tag] = string(entry.DiscordID)
		}
	}

	return updatedLeaderboard
}

// FindTagByDiscordID is a helper function to find the tag associated with a Discord ID in the leaderboard data.
func FindTagByDiscordID(leaderboard *leaderboarddb.Leaderboard, discordID leaderboardtypes.DiscordID) (int, bool) {
	for tag, id := range leaderboard.LeaderboardData {
		if leaderboardtypes.DiscordID(id) == discordID {
			return tag, true
		}
	}
	return 0, false
}

// --- Helper functions for generateUpdatedLeaderboard ---

func parseTagDiscordIDPair(tagDiscordIDPair string) (string, string, error) {
	parts := strings.Split(tagDiscordIDPair, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid tag-discordID pair format: %s", tagDiscordIDPair)
	}
	return parts[0], parts[1], nil
}

func createNewTagMap(newTagOrder []string) map[string]string {
	newTagMap := make(map[string]string)
	for _, tagDiscordIDPair := range newTagOrder {
		tag, discordID, err := parseTagDiscordIDPair(tagDiscordIDPair)
		if err == nil { // Only add to map if parsing was successful
			newTagMap[tag] = discordID
		}
	}
	return newTagMap
}

func sortCurrentEntries(currentEntries map[int]string) []leaderboardevents.LeaderboardEntry {
	var sortedCurrentEntries []leaderboardevents.LeaderboardEntry
	for tag, discordID := range currentEntries {
		sortedCurrentEntries = append(sortedCurrentEntries, leaderboardevents.LeaderboardEntry{
			TagNumber: strconv.Itoa(tag),
			DiscordID: leaderboardtypes.DiscordID(discordID),
		})
	}
	sort.Slice(sortedCurrentEntries, func(i, j int) bool {
		// Sort by tag number (ascending).
		tagI, _ := strconv.Atoi(sortedCurrentEntries[i].TagNumber)
		tagJ, _ := strconv.Atoi(sortedCurrentEntries[j].TagNumber)
		return tagI < tagJ
	})
	return sortedCurrentEntries
}

// InsertTag inserts a new tag/Discord ID into the correct position in the leaderboard, maintaining order.
func InsertTag(currentLeaderboard *leaderboarddb.Leaderboard, tagNumber leaderboardtypes.Tag, discordID leaderboardtypes.DiscordID) leaderboardtypes.LeaderboardData {
	updatedLeaderboard := make(leaderboardtypes.LeaderboardData)
	inserted := false

	// Sort current leaderboard entries for ordered insertion
	sortedCurrentEntries := sortCurrentEntries(currentLeaderboard.LeaderboardData)

	// Find the correct position to insert the new tag
	nextAvailableTag := 1
	for _, entry := range sortedCurrentEntries {
		currentTag, _ := strconv.Atoi(entry.TagNumber)

		if !inserted && int(tagNumber) < currentTag {
			updatedLeaderboard[int(tagNumber)] = string(discordID)
			inserted = true
		}

		for {
			if _, exists := updatedLeaderboard[nextAvailableTag]; !exists {
				updatedLeaderboard[nextAvailableTag] = string(entry.DiscordID)
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
				updatedLeaderboard[nextAvailableTag] = string(discordID)
				break
			}
			nextAvailableTag++
		}
	}

	return updatedLeaderboard
}
