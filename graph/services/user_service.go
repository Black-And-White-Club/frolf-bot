package service

import (
	"context"
	"fmt"

	"github.com/romero-jace/leaderboard/graph/model"
)

// CreateUser creates a new user in Firestore
func CreateUser(ctx context.Context, input model.UserInput) (*model.User, error) {
	// Validate input
	if input.DiscordID == "" || input.Name == "" {
		return nil, fmt.Errorf("DiscordID and Name are required")
	}

	// Check if user already exists
	user, err := GetUser(ctx, input.DiscordID)
	if err != nil && err != firestore.ErrNotFound {
		return nil, err
	}

	if user != nil {
		return nil, fmt.Errorf("User with DiscordID %s already exists", input.DiscordID)
	}

	// Create a new user
	newUser := &model.User{
		DiscordID: input.DiscordID,
		Name:      input.Name,
	}

	_, err = FirestoreClient.Collection("users").Doc(newUser.DiscordID).Set(ctx, newUser)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}

// GetUser retrieves a user from Firestore
func GetUser(ctx context.Context, client *firestore.Client, discordID string) (*model.User, error) {
	docRef := client.Collection("users").Doc(discordID)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if err == firestore.ErrNotFound {
			return nil, fmt.Errorf("User not found")
		}
		return nil, err
	}

	var user model.User
	err = docSnap.DataTo(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
