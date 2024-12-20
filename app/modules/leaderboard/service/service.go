package leaderboardservice

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
	leaderboardevents "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/events"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// LeaderboardService handles leaderboard logic.
type LeaderboardService struct {
	LeaderboardDB leaderboarddb.LeaderboardDB
	Publisher     message.Publisher
	logger        watermill.LoggerAdapter
}

// NewLeaderboardService creates a new LeaderboardService.
func NewLeaderboardService(publisher message.Publisher, db leaderboarddb.LeaderboardDB, logger watermill.LoggerAdapter) *LeaderboardService {
	return &LeaderboardService{
		LeaderboardDB: db,
		Publisher:     publisher,
		logger:        logger,
	}
}

// UpdateLeaderboard processes a LeaderboardUpdateEvent.
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, event leaderboardevents.LeaderboardUpdateEvent) error {

	// Sort scores
	sortedScores, err := s.sortScores(event.Scores)
	if err != nil {
		return fmt.Errorf("failed to sort scores: %w", err)
	}

	// Update the leaderboard in the database
	if err := s.LeaderboardDB.UpdateLeaderboard(ctx, sortedScores); err != nil {
		return fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return nil
}

// AssignTag assigns a tag to a user.
func (s *LeaderboardService) AssignTag(ctx context.Context, event leaderboardevents.TagAssigned) error {
	// Add the tag to the leaderboard in the database
	if err := s.LeaderboardDB.AssignTag(ctx, event.DiscordID, event.TagNumber); err != nil {
		return fmt.Errorf("failed to assign tag: %w", err)
	}

	return nil
}

// SwapTags swaps tags between two users.
func (s *LeaderboardService) SwapTags(ctx context.Context, requestorID, targetID string) error {
	// Perform the tag swap in the database
	if err := s.LeaderboardDB.SwapTags(ctx, requestorID, targetID); err != nil {
		return fmt.Errorf("failed to swap tags: %w", err)
	}

	return nil
}

// GetLeaderboard returns the current leaderboard.
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) ([]leaderboardevents.LeaderboardEntry, error) {
	// Fetch the leaderboard from the database
	leaderboard, err := s.LeaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}

	// Access the LeaderboardData map from the pointer
	leaderboardData := leaderboard.LeaderboardData

	// Convert to leaderboardevents.LeaderboardEntry
	eventLeaderboard := make([]leaderboardevents.LeaderboardEntry, len(leaderboardData)) // Use len(leaderboardData)
	i := 0
	for tagNumber, discordID := range leaderboardData { // Range over leaderboardData
		eventLeaderboard[i] = leaderboardevents.LeaderboardEntry{
			TagNumber: strconv.Itoa(tagNumber),
			DiscordID: discordID,
		}
		i++
	}

	return eventLeaderboard, nil
}

// GetTagByDiscordID returns the tag number associated with a Discord ID.
func (s *LeaderboardService) GetTagByDiscordID(ctx context.Context, discordID string) (int, error) { // Changed return type to int
	// Fetch the tag number from the database
	tagNumber, err := s.LeaderboardDB.GetTagByDiscordID(ctx, discordID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tag by Discord ID: %w", err)
	}

	return tagNumber, nil // Return the tag number as an integer
}

// CheckTagAvailability checks if a tag number is available.
func (s *LeaderboardService) CheckTagAvailability(ctx context.Context, tagNumber int) (bool, error) {
	// Check tag availability in the database
	isAvailable, err := s.LeaderboardDB.CheckTagAvailability(ctx, tagNumber)
	if err != nil {
		return false, fmt.Errorf("failed to check tag availability: %w", err)
	}

	return isAvailable, nil
}

// sortScores sorts the scores according to your specific criteria.
func (s *LeaderboardService) sortScores(scores []leaderboardevents.Score) ([]leaderboardevents.Score, error) {
	// Sort by score (ascending), then by tag number (descending)
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].Score == scores[j].Score {
			// Convert tag numbers to integers for comparison
			tagI, errI := strconv.Atoi(scores[i].TagNumber)
			tagJ, errJ := strconv.Atoi(scores[j].TagNumber)
			if errI != nil || errJ != nil {
				// Handle the error (e.g., log it or return an error)
				// For now, assuming valid tag numbers
				return scores[i].TagNumber > scores[j].TagNumber
			}
			return tagI > tagJ // Sort by tag number in descending order
		}
		return scores[i].Score < scores[j].Score // Sort by score in ascending order
	})

	return scores, nil
}
