package leaderboardqueries

import (
	"context"
	"errors"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
)

// LeaderboardQueryService defines methods for querying leaderboard data.
type LeaderboardQueryService interface {
	GetUserTag(ctx context.Context, query GetUserTagQuery) (int, error)
	IsTagTaken(ctx context.Context, tagNumber int) (bool, error)
	GetParticipantTag(ctx context.Context, participantID string) (int, error)
	// Add other query methods as needed
}

type leaderboardQueryService struct {
	leaderboardDB leaderboarddb.LeaderboardDB
}

// NewLeaderboardQueryService creates a new LeaderboardQueryService.
func NewLeaderboardQueryService(leaderboardDB leaderboarddb.LeaderboardDB) LeaderboardQueryService {
	return &leaderboardQueryService{leaderboardDB: leaderboardDB}
}

// GetUserTag retrieves the tag number for a specific user.
func (s *leaderboardQueryService) GetUserTag(ctx context.Context, query GetUserTagQuery) (int, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetUserTag: failed to get leaderboard: %w", err) // Wrap the error with context
	}

	for tagNumber, discordID := range leaderboard.LeaderboardData {
		if discordID == query.DiscordID {
			return tagNumber, nil
		}
	}

	return 0, errors.New("GetUserTag: user not found on leaderboard") // Add context to the error
}

// IsTagTaken checks if a tag number is already taken.
func (s *leaderboardQueryService) IsTagTaken(ctx context.Context, tagNumber int) (bool, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return false, fmt.Errorf("IsTagTaken: failed to get leaderboard: %w", err) // Wrap the error
	}

	_, taken := leaderboard.LeaderboardData[tagNumber]
	return taken, nil
}

// GetParticipantTag retrieves a tag number for a participant.
func (s *leaderboardQueryService) GetParticipantTag(ctx context.Context, participantID string) (int, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetParticipantTag: failed to get leaderboard: %w", err) // Wrap the error
	}

	// Check if the participant already has a tag
	for tagNumber, discordID := range leaderboard.LeaderboardData {
		if discordID == participantID {
			return tagNumber, nil
		}
	}

	// If not, return an error indicating no tag assigned
	return 0, errors.New("GetParticipantTag: no tag assigned to participant") // Add context
}
