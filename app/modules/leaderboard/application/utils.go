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

func (s *LeaderboardService) GenerateUpdatedLeaderboard(currentLeaderboard *leaderboarddb.Leaderboard, sortedParticipantTags []string) (*leaderboarddb.Leaderboard, error) {
	if len(sortedParticipantTags) == 0 {
		return nil, fmt.Errorf("cannot generate updated leaderboard with empty participant tags")
	}

	leaderboardMap := make(map[sharedtypes.DiscordID]*leaderboardtypes.LeaderboardEntry)
	for i := range currentLeaderboard.LeaderboardData {
		entry := &currentLeaderboard.LeaderboardData[i]
		leaderboardMap[entry.UserID] = entry
	}

	updatedLeaderboardMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber)

	for _, entryStr := range sortedParticipantTags {
		parts := strings.Split(entryStr, ":")
		if len(parts) != 2 {
			s.logger.Warn("Invalid sorted participant tag format", attr.String("tag_user_string", entryStr))
			continue
		}

		tagNumInt, err := strconv.Atoi(parts[0])
		if err != nil {
			s.logger.Warn("Invalid tag number format in sorted participant tag", attr.String("tag_string", parts[0]), attr.Error(err))
			continue
		}
		tagNum := sharedtypes.TagNumber(tagNumInt)
		userID := sharedtypes.DiscordID(parts[1])

		updatedLeaderboardMap[userID] = tagNum
	}

	newLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(updatedLeaderboardMap))
	for userID, tagNum := range updatedLeaderboardMap {
		tag := tagNum
		newLeaderboardData = append(newLeaderboardData, leaderboardtypes.LeaderboardEntry{
			UserID:    userID,
			TagNumber: &tag,
		})
	}

	slices.SortFunc(newLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == nil && b.TagNumber == nil {
			return 0
		}
		if a.TagNumber == nil {
			return -1
		}
		if b.TagNumber == nil {
			return 1
		}
		return int(*a.TagNumber - *b.TagNumber)
	})

	return &leaderboarddb.Leaderboard{
		LeaderboardData: newLeaderboardData,
		IsActive:        true,
		UpdateSource:    leaderboarddb.ServiceUpdateSourceProcessScores,
	}, nil
}

// FindTagByUserID is a helper function to find the tag associated with a Discord ID in the leaderboard data.
func (s *LeaderboardService) FindTagByUserID(ctx context.Context, leaderboard *leaderboarddb.Leaderboard, userID sharedtypes.DiscordID) (int, bool) {
	operationName := "FindTagByUserID"
	s.logger.InfoContext(ctx, "Starting "+operationName,
		attr.String("leaderboard", fmt.Sprintf("%+v", leaderboard)),
		attr.String("user_id", string(userID)),
	)

	s.metrics.RecordOperationAttempt(ctx, operationName, "leaderboard")

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		s.metrics.RecordOperationDuration(ctx, operationName, "leaderboard", time.Duration(duration))
	}()

	for _, entry := range leaderboard.LeaderboardData {
		if entry.UserID == userID {
			s.logger.InfoContext(ctx, "Tag found",
				attr.Int("tag", int(*entry.TagNumber)),
				attr.String("user_id", string(userID)),
			)
			return int(*entry.TagNumber), true
		}
	}

	s.logger.InfoContext(ctx, "Tag not found",
		attr.String("leaderboard", fmt.Sprintf("%+v", leaderboard)),
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
		if entry.TagNumber != nil && *entry.TagNumber == tagNumber {
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
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, len(currentLeaderboard.LeaderboardData)+1)
	copy(updatedLeaderboardData, currentLeaderboard.LeaderboardData)

	// Add the new entry
	updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
		TagNumber: &tagNumber,
		UserID:    userID,
	})

	// Sort by tag number
	slices.SortFunc(updatedLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == nil && b.TagNumber == nil {
			return 0
		}
		if a.TagNumber == nil {
			return -1
		}
		if b.TagNumber == nil {
			return 1
		}
		return int(*a.TagNumber - *b.TagNumber)
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
		if entry.UserID != userID && entry.TagNumber != nil && *entry.TagNumber == newTagNumber {
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
		TagNumber: &newTagNumber,
		UserID:    userID,
	})

	// Sort by tag number
	slices.SortFunc(updatedLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		if a.TagNumber == nil && b.TagNumber == nil {
			return 0
		}
		if a.TagNumber == nil {
			return -1
		}
		if b.TagNumber == nil {
			return 1
		}
		return int(*a.TagNumber - *b.TagNumber)
	})

	return updatedLeaderboardData, nil
}
