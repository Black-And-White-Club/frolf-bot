package leaderboardqueries

import (
	"context"
	"errors"
	"fmt"

	leaderboarddb "github.com/Black-And-White-Club/tcr-bot/app/modules/leaderboard/db"
)

// GetUserTagQuery represents a query to get a user's tag number.
type GetUserTagQuery struct {
	UserID string `json:"user_id"`
}

// CheckTagTakenQuery represents a query to check if a tag number is taken.
type CheckTagTakenQuery struct {
	TagNumber int `json:"tag_number"`
}

// GetParticipantTagQuery represents a query to get a participant's tag number.
type GetParticipantTagQuery struct {
	ParticipantID string `json:"participant_id"`
}
type leaderboardQueryService struct {
	leaderboardDB leaderboarddb.LeaderboardDB
}

// NewLeaderboardQueryService creates a new LeaderboardQueryService.
func NewLeaderboardQueryService(leaderboardDB leaderboarddb.LeaderboardDB) QueryService { // Changed return type to LeaderboardQueryService
	return &leaderboardQueryService{leaderboardDB: leaderboardDB}
}

// GetUserTag retrieves the tag number for a specific user.
func (s *leaderboardQueryService) GetUserTag(ctx context.Context, query GetUserTagQuery) (int, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetUserTag: failed to get leaderboard: %w", err)
	}

	for tagNumber, discordID := range leaderboard.LeaderboardData {
		if discordID == query.UserID {
			return tagNumber, nil
		}
	}

	return 0, errors.New("GetUserTag: user not found on leaderboard")
}

// IsTagTaken checks if a tag number is already taken.
func (s *leaderboardQueryService) IsTagTaken(ctx context.Context, tagNumber int) (bool, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return false, fmt.Errorf("IsTagTaken: failed to get leaderboard: %w", err)
	}

	_, taken := leaderboard.LeaderboardData[tagNumber]
	return taken, nil
}

// GetParticipantTag retrieves a tag number for a participant.
func (s *leaderboardQueryService) GetParticipantTag(ctx context.Context, participantID string) (int, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return 0, fmt.Errorf("GetParticipantTag: failed to get leaderboard: %w", err)
	}

	for tagNumber, discordID := range leaderboard.LeaderboardData {
		if discordID == participantID {
			return tagNumber, nil
		}
	}

	return 0, errors.New("GetParticipantTag: no tag assigned to participant")
}

// GetTagHolder retrieves the Discord ID of the user holding the given tag number
func (s *leaderboardQueryService) GetTagHolder(ctx context.Context, tagNumber int) (string, error) {
	leaderboard, err := s.leaderboardDB.GetLeaderboard(ctx)
	if err != nil {
		return "", fmt.Errorf("GetTagHolder: failed to get leaderboard: %w", err)
	}

	discordID, ok := leaderboard.LeaderboardData[tagNumber]
	if !ok {
		return "", fmt.Errorf("GetTagHolder: tag number %d is not assigned", tagNumber)
	}

	return discordID, nil
}

// GetLeaderboard retrieves the active leaderboard.
func (s *leaderboardQueryService) GetLeaderboard(ctx context.Context) (*leaderboarddb.Leaderboard, error) {
	return s.leaderboardDB.GetLeaderboard(ctx)
}
