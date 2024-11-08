package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/romero-jace/leaderboard/graph/model"
)

// LeaderboardService handles operations related to the leaderboard
type LeaderboardService struct {
	FirestoreClient *firestore.Client
}

// GetLeaderboard retrieves the current leaderboard from Firestore
func (s *LeaderboardService) GetLeaderboard(ctx context.Context) (*model.Leaderboard, error) {
	// Fetch all users from Firestore
	users, err := s.getAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	// Fetch placements (this could be score-based, ranking-based, etc.)
	placements, err := s.getPlacements(ctx)
	if err != nil {
		return nil, err
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
		if err == firestore.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user model.User
		err = doc.DataTo(&user)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, nil
}

// getPlacements retrieves the placements from Firestore or calculates them
func (s *LeaderboardService) getPlacements(ctx context.Context) ([]*model.Tag, error) {
	// This is a placeholder implementation.
	// You would typically fetch scores or rankings from Firestore or calculate them based on your logic.
	// For this example, let's assume we have a fixed list of placements.

	// Example placements
	placements := []*model.Tag{
		{Name: "User 1", Position: 1},
		{Name: "User 2", Position: 2},
		{Name: "User 3", Position: 3},
	}

	return placements, nil
}

// UpdateLeaderboard allows authorized users to update the leaderboard
func (s *LeaderboardService) UpdateLeaderboard(ctx context.Context, userID string, newPlacement *model.Tag) error {
	if !s.hasPermission(userID) {
		return fmt.Errorf("user does not have permission to edit the leaderboard")
	}

	// Logic to update the leaderboard in Firestore
	// This could involve updating scores or placements based on your requirements
	// For example, you might want to store placements in a Firestore collection

	// Example logic to update placements
	_, err := s.FirestoreClient.Collection("leaderboard").Doc(newPlacement.Name).Set(ctx, newPlacement)
	if err != nil {
		return err
	}

	return nil
}

// hasPermission checks if the user has permission to edit the leaderboard
func (s *LeaderboardService) hasPermission(userID string) bool {
	// Implement your permission logic here
	// For example, check if the user is an admin or has specific roles
	return userID == "admin_user_id" // Replace with your actual permission logic
}
