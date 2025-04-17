package leaderboardservice

import (
	"context"
	"fmt"
	"slices"
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

	// Create a map for quick lookups of current leaderboard entries
	tagMap := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(currentLeaderboard.LeaderboardData))
	for _, entry := range currentLeaderboard.LeaderboardData {
		tagMap[entry.UserID] = entry.TagNumber
	}

	// Parse participant tags and extract user IDs in performance order
	participants := make([]sharedtypes.DiscordID, 0, len(sortedParticipantTags))
	availableTags := make([]sharedtypes.TagNumber, 0, len(sortedParticipantTags))

	for _, entry := range sortedParticipantTags {
		parts := strings.Split(entry, ":")
		if len(parts) != 2 {
			continue
		}

		userID := sharedtypes.DiscordID(parts[1])

		// Only include participants that exist in the current leaderboard
		if existingTag, exists := tagMap[userID]; exists {
			participants = append(participants, userID)
			availableTags = append(availableTags, existingTag)
		}
	}

	// Sort the available tags in ascending order
	slices.Sort(availableTags)

	// Create a map to store new tag assignments
	newTagAssignments := make(map[sharedtypes.DiscordID]sharedtypes.TagNumber, len(participants))

	// Assign tags based on performance (best performer gets lowest tag)
	for i, userID := range participants {
		newTagAssignments[userID] = availableTags[i]
	}

	// Update the leaderboard data
	newLeaderboardData := make([]leaderboardtypes.LeaderboardEntry, len(currentLeaderboard.LeaderboardData))
	copy(newLeaderboardData, currentLeaderboard.LeaderboardData)

	// Apply the new tag assignments
	for i, entry := range newLeaderboardData {
		if newTag, exists := newTagAssignments[entry.UserID]; exists {
			newLeaderboardData[i].TagNumber = newTag
		}
	}

	// Return the updated leaderboard
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
				attr.Int("tag", int(entry.TagNumber)),
				attr.String("user_id", string(userID)),
			)
			return int(entry.TagNumber), true
		}
	}

	s.logger.InfoContext(ctx, "Tag not found",
		attr.String("leaderboard", fmt.Sprintf("%+v", leaderboard)),
		attr.String("user_id", string(userID)),
	)
	s.metrics.RecordOperationFailure(ctx, operationName, "leaderboard")

	return 0, false
}

// PrepareTagAssignment takes the current leaderboard and assigns a new tag,
// returning the updated leaderboard data
func (s *LeaderboardService) PrepareTagAssignment(
	currentLeaderboard *leaderboarddb.Leaderboard,
	userID sharedtypes.DiscordID,
	tagNumber sharedtypes.TagNumber,
) (leaderboardtypes.LeaderboardData, error) {
	// Validate tag number (add this validation)
	if tagNumber < 0 {
		return nil, fmt.Errorf("invalid input: invalid tag number")
	}

	// Convert LeaderboardData to a map for easy tag lookup
	tagMap := make(map[int]string)
	for _, entry := range currentLeaderboard.LeaderboardData {
		tagMap[int(entry.TagNumber)] = string(entry.UserID)
	}

	// Check if the tag is already assigned
	if existingUser, exists := tagMap[int(tagNumber)]; exists {
		return nil, fmt.Errorf("tag %d is already assigned to user %s", tagNumber, existingUser)
	}

	// Add the new assignment to the map
	tagMap[int(tagNumber)] = string(userID)

	// Convert the map back to LeaderboardData
	updatedLeaderboardData := make(leaderboardtypes.LeaderboardData, 0, len(tagMap))
	for tag, uid := range tagMap {
		updatedLeaderboardData = append(updatedLeaderboardData, leaderboardtypes.LeaderboardEntry{
			TagNumber: sharedtypes.TagNumber(tag),
			UserID:    sharedtypes.DiscordID(uid),
		})
	}

	// Use slices.Sort for more efficient and ergonomic sorting
	slices.SortFunc(updatedLeaderboardData, func(a, b leaderboardtypes.LeaderboardEntry) int {
		return int(a.TagNumber - b.TagNumber)
	})

	return updatedLeaderboardData, nil
}
