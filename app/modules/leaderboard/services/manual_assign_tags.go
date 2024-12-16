package leaderboardservices

import (
	"context"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
)

// ManualTagAssignmentService handles manual tag assignment.
type ManualTagAssignmentService struct {
	leaderboardDB leaderboarddb.LeaderboardDB
}

// NewManualTagAssignmentService creates a new ManualTagAssignmentService.
func NewManualTagAssignmentService(leaderboardDB leaderboarddb.LeaderboardDB) *ManualTagAssignmentService {
	return &ManualTagAssignmentService{leaderboardDB: leaderboardDB}
}

// AssignTagToUser assigns the given tag number to the user.
func (s *ManualTagAssignmentService) AssignTagToUser(ctx context.Context, discordID string, tagNumber int) error {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return fmt.Errorf("AssignTagToUser: failed to get leaderboard: %w", err)
	}

	// Check if the tag is already taken
	if _, taken := leaderboard.LeaderboardData[tagNumber]; taken {
		return fmt.Errorf("AssignTagToUser: tag number %d is already taken", tagNumber)
	}

	// Update the leaderboard data
	leaderboard.LeaderboardData[tagNumber] = discordID

	// Update the leaderboard in the database
	err = s.leaderboardDB.UpdateLeaderboardWithTransaction(ctx, leaderboard.LeaderboardData)
	if err != nil {
		return fmt.Errorf("AssignTagToUser: failed to update leaderboard: %w", err)
	}

	return nil
}
