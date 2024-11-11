package services

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/tcr-bot/graph/model"
	"google.golang.org/api/iterator"
)

// LeaderboardService handles operations related to the leaderboard
type LeaderboardService struct {
	FirestoreClient *firestore.Client

	// Function fields for mocking
	GetLeaderboardFunc    func(ctx context.Context) (*model.Leaderboard, error)
	UpdateLeaderboardFunc func(ctx context.Context, userID string, newPlacement *model.Tag) error
}

// NewLeaderboardService creates a new instance of LeaderboardService
func NewLeaderboardService(client *firestore.Client) *LeaderboardService {
	return &LeaderboardService{FirestoreClient: client}
}

// GetLeaderboard retrieves the current leaderboard from Firestore
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) (*model.Leaderboard, error) {
	if s.GetLeaderboardFunc != nil {
		return s.GetLeaderboardFunc(ctx) // Call the mock function if set
	}

	users, err := s.getAllUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	placements, err := s.getPlacements(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get placements: %w", err)
	}

	return &model.Leaderboard{
		Users:      users,
		Placements: placements,
	}, nil
}

// getAllUsers retrieves all users from Firestore
func (s *LeaderboardService) getAllUsers(ctx context.Context) ([]*model.User, error) {
	var users []*model.User
	iter := s.FirestoreClient.Collection("users").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate users: %w", err)
		}

		var user model.User
		if err := doc.DataTo(&user); err != nil {
			return nil, fmt.Errorf("failed to convert user data: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// getPlacements retrieves the placements from Firestore or calculates them
func (s *LeaderboardService) getPlacements(ctx context.Context) ([]*model.Tag, error) {
	var placements []*model.Tag
	iter := s.FirestoreClient.Collection("scores").Documents(ctx)
	defer iter.Stop()

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate scores: %w", err)
		}

		var score model.Score
		if err := doc.DataTo(&score); err != nil {
			return nil, fmt.Errorf("failed to convert score data: %w", err)
		}

		// Calculate placement based on score
		placements = append(placements, &model.Tag{
			Name:     score.UserID, // Adjust as needed to get the user's name
			Position: score.Score,  // Adjust as needed
		})
	}

	return placements, nil
}

// UpdateLeaderboard allows authorized users to update the leaderboard
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, userID string, newPlacement *model.Tag) error {
	if s.UpdateLeaderboardFunc != nil {
		return s.UpdateLeaderboardFunc(ctx, userID, newPlacement) // Call the mock function if set
	}

	if !s.hasPermission(userID) {
		return fmt.Errorf("user does not have permission to edit the leaderboard")
	}

	// Logic to update the leaderboard in Firestore
	_, err := s.FirestoreClient.Collection("leaderboard").Doc(newPlacement.Name).Set(ctx, newPlacement)
	if err != nil {
		return fmt.Errorf("failed to update leaderboard: %w", err)
	}

	return nil
}

// hasPermission checks if the user has permission to edit the leaderboard
func (s *LeaderboardService) hasPermission(userID string) bool {
	// Implement your permission logic here
	// Consider using a configuration file or environment variable for the admin user ID
	adminUserID := "admin_user_id" // Replace with your actual permission logic
	return userID == adminUserID
}
