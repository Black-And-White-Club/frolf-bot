package leaderboardservice

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/Black-And-White-Club/frolf-bot-shared/observability/attr"
	leaderboardtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/leaderboard"
	sharedtypes "github.com/Black-And-White-Club/frolf-bot-shared/types/shared"
	leaderboarddb "github.com/Black-And-White-Club/frolf-bot/app/modules/leaderboard/infrastructure/repositories"
)

// GenerateUpdatedLeaderboard creates a new LeaderboardData slice based on the current data
// and a list of sorted participant tags in "tag:user" format.
// The participants will be assigned the tags provided in the sortedParticipantTags list
// in the order they appear. Non-participants from the current leaderboard are included
// with their existing tags.
func (s *LeaderboardService) GenerateUpdatedLeaderboard(currentLeaderboardData leaderboardtypes.LeaderboardData, sortedParticipantTags []string) (leaderboardtypes.LeaderboardData, error) {
	if len(sortedParticipantTags) == 0 {
		// This is already checked in UpdateLeaderboard, but keeping for GenerateUpdatedLeaderboard's contract
		return nil, fmt.Errorf("cannot generate updated leaderboard data with empty participant tags")
	}

	// Create a map to store the current leaderboard entries by user ID for quick lookup
	currentEntries := make(map[sharedtypes.DiscordID]leaderboardtypes.LeaderboardEntry, len(currentLeaderboardData))
	for _, entry := range currentLeaderboardData {
		currentEntries[entry.UserID] = entry
	}

	// Create a map to track participant user IDs for quick lookup
	participantUsers := make(map[sharedtypes.DiscordID]bool, len(sortedParticipantTags))

	// Create a slice for the new leaderboard data, starting with participants
	newLeaderboardData := make([]leaderboardtypes.LeaderboardEntry, 0, len(sortedParticipantTags))

	// Process sortedParticipantTags to get the new tag assignments for participants
	for _, entryStr := range sortedParticipantTags {
		parts := strings.Split(entryStr, ":")
		if len(parts) != 2 {
			s.logger.Error("Invalid sorted participant tag format", "tag_user_string", entryStr)
			return nil, fmt.Errorf("invalid sorted participant tag format: %s", entryStr)
		}

		tagNumberInt, err := strconv.Atoi(parts[0])
		if err != nil {
			s.logger.Error("Failed to parse tag number from sorted participant tag", "tag_string", parts[0], "error", err)
			return nil, fmt.Errorf("invalid tag number format: %s", parts[0])
		}
		tagNumber := sharedtypes.TagNumber(tagNumberInt)
		userID := sharedtypes.DiscordID(parts[1])

		// Add the participant with their new tag number
		newLeaderboardData = append(newLeaderboardData, leaderboardtypes.LeaderboardEntry{
			UserID:    userID,
			TagNumber: tagNumber,
		})

		// Mark user as participant
		participantUsers[userID] = true
	}

	// Add non-participating users from the current leaderboard to the new data
	// Their tags remain unchanged
	for userID, entry := range currentEntries {
		if !participantUsers[userID] {
			newLeaderboardData = append(newLeaderboardData, entry)
		}
	}

	// Sort the final list of leaderboard entries by tag number
	slices.SortFunc(newLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		// Standard comparison for non-zero tags
		if a.TagNumber != 0 && b.TagNumber != 0 {
			return int(a.TagNumber - b.TagNumber)
		}
		// Handle cases with tag 0. Assuming 0 is the zero value and should be sorted appropriately
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0 // Both are 0, consider them equal for sorting purposes
		}
		if a.TagNumber == 0 {
			return 1 // Place 0 after non-zero tags
		}
		return -1 // Place non-zero tags before 0
	})

	return newLeaderboardData, nil
}

// FindTagByUserID is a helper function to find the tag associated with a Discord ID in the leaderboard data.
func (s *LeaderboardService) FindTagByUserID(ctx context.Context, leaderboard *leaderboarddb.Leaderboard, userID sharedtypes.DiscordID) (int, bool) {
	operationName := "FindTagByUserID"
	s.logger.InfoContext(ctx, "Starting "+operationName,
		attr.String("user_id", string(userID)),
	)

	s.metrics.RecordOperationAttempt(ctx, operationName, "leaderboard")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		s.metrics.RecordOperationDuration(ctx, operationName, "leaderboard", duration)
	}()

	for _, entry := range leaderboard.LeaderboardData {
		if entry.UserID == userID {
			s.logger.InfoContext(ctx, "Tag found",
				attr.Int("tag", int(entry.TagNumber)),
				attr.String("user_id", string(userID)),
			)
			return int(entry.TagNumber), true
		}
	}

	s.logger.InfoContext(ctx, "Tag not found",
		attr.String("user_id", string(userID)),
	)
	s.metrics.RecordOperationFailure(ctx, operationName, "leaderboard")

	return 0, false
}

type TagSwapNeededError struct {
	RequestorID sharedtypes.DiscordID
	TargetID    sharedtypes.DiscordID
	TagNumber   sharedtypes.TagNumber
}

func (e *TagSwapNeededError) Error() string {
	return fmt.Sprintf("tag %d is already assigned to user %s, swap needed", e.TagNumber, e.TargetID)
}

func (s *LeaderboardService) PrepareTagAssignment(
	currentLeaderboard *leaderboarddb.Leaderboard,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
) (leaderboardtypes.LeaderboardData, error) {
	if tagNumber < 0 {
		return nil, fmt.Errorf("invalid input: tag number cannot be negative")
	}

	var userHasTag bool
	for _, entry := range currentLeaderboard.LeaderboardData {
		if entry.UserID == userID {
			userHasTag = true
			break
		}
	}

	for _, entry := range currentLeaderboard.LeaderboardData {
		if entry.TagNumber != 0 && entry.TagNumber == tagNumber {
			if entry.UserID == userID {
				// User is re-claiming their own tag, allow as no-op or success
				return nil, nil
			}
			if userHasTag {
				// User has a different tag, swap needed
				return nil, &TagSwapNeededError{
					RequestorID: userID,
					TargetID:    entry.UserID,
					TagNumber:   tagNumber,
				}
			}
			// User doesn't have a tag, fail
			return nil, fmt.Errorf("tag %d is already assigned to user %s", tagNumber, entry.UserID)
		}
	}

	// Create a copy of the leaderboard data with the new assignment
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, len(currentLeaderboard.LeaderboardData))
	copy(updatedLeaderboardData, currentLeaderboard.LeaderboardData)

	// Add the new entry
	updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
		TagNumber: tagNumber,
		UserID:    userID,
	})

	// Sort by tag number
	slices.SortFunc(updatedLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0
		}
		if a.TagNumber == 0 {
			return 1
		}
		if b.TagNumber == 0 {
			return -1
		}
		return int(a.TagNumber - b.TagNumber)
	})

	return updatedLeaderboardData, nil
}

// PrepareTagUpdateForExistingUser handles validation for updating a tag for a user
// that already exists in the leaderboard
func (s *LeaderboardService) PrepareTagUpdateForExistingUser(
	currentLeaderboard *leaderboarddb.Leaderboard,
	userID sharedtypes.DiscordID,
	newTagNumber sharedtypes.TagNumber,
) (leaderboardtypes.LeaderboardData, error) {
	// Validate tag number
	if newTagNumber < 0 {
		return nil, fmt.Errorf("invalid input: tag number cannot be negative")
	}

	// Find the user's current entry
	var userCurrentEntry *leaderboardtypes.LeaderboardEntry
	for _, entry := range currentLeaderboard.LeaderboardData {
		if entry.UserID == userID {
			entryClone := entry // Create a copy to avoid modifying the original
			userCurrentEntry = &entryClone
			break
		}
	}

	if userCurrentEntry == nil {
		return nil, fmt.Errorf("user %s not found in leaderboard", userID)
	}

	// Check if the new tag is already assigned to another user
	for _, entry := range currentLeaderboard.LeaderboardData {
		if entry.UserID != userID && entry.TagNumber != 0 && entry.TagNumber == newTagNumber {
			// Return TagSwapNeededError instead of generic error
			return nil, &TagSwapNeededError{
				RequestorID: userID,
				TargetID:    entry.UserID,
				TagNumber:   newTagNumber,
			}
		}
	}

	// Create a copy of the leaderboard data for the update
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(currentLeaderboard.LeaderboardData))

	// Add all entries except the user's entry
	for _, entry := range currentLeaderboard.LeaderboardData {
		if entry.UserID != userID {
			updatedLeaderboardData = append(updatedLeaderboardData, entry)
		}
	}

	// Add the updated entry for the user
	updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
		TagNumber: newTagNumber,
		UserID:    userID,
	})

	// Sort by tag number
	slices.SortFunc(updatedLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == 0 && b.TagNumber == 0 {
			return 0
		}
		if a.TagNumber == 0 {
			return 1
		}
		if b.TagNumber == 0 {
			return -1
		}
		return int(a.TagNumber - b.TagNumber)
	})

	return updatedLeaderboardData, nil
}
