package services

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"              // Make sure to import Firestore
	"github.com/romero-jace/tcr-bot/graph/model" // Import the iterator package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	client *firestore.Client
}

// NewUser Service creates a new UserService
func NewUserService(client *firestore.Client) *UserService {
	return &UserService{client: client}
}

// CreateUser  creates a new user in Firestore
func (us *UserService) CreateUser(ctx context.Context, input model.UserInput) (*model.User, error) {
	log.Printf("Creating user with Discord ID: %s", input.DiscordID)

	// Validate input
	if input.DiscordID == "" || input.Name == "" {
		return nil, fmt.Errorf("DiscordID and Name are required")
	}

	// Check if the user already exists
	existingUserDoc, err := us.client.Collection("users").Doc(input.DiscordID).Get(ctx)
	if err == nil && existingUserDoc.Exists() {
		return nil, fmt.Errorf("User  with DiscordID %s already exists", input.DiscordID)
	}

	// Create a new user
	newUser := &model.User{
		DiscordID: input.DiscordID,
		Name:      input.Name,
	}

	// Attempt to create the user document in Firestore
	_, err = us.client.Collection("users").Doc(newUser.DiscordID).Set(ctx, newUser)
	if err != nil {
		return nil, err
	}

	log.Printf("User  created successfully: %v", newUser)
	return newUser, nil
}

// GetUser  retrieves a user from Firestore
func (us *UserService) GetUser(ctx context.Context, discordID string) (*model.User, error) {
	log.Printf("Retrieving user with Discord ID: %s", discordID)

	// Validate input
	if discordID == "" {
		return nil, fmt.Errorf("DiscordID is required")
	}

	// Attempt to retrieve the user document from Firestore
	doc, err := us.client.Collection("users").Doc(discordID).Get(ctx)
	if err != nil {
		// Check if the error is a NotFound error
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			return nil, fmt.Errorf("User with DiscordID %s not found", discordID)
		}
		return nil, err
	}

	// Map the document data to the User model
	var user model.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	log.Printf("User  retrieved successfully: %v", user)
	return &user, nil
}
