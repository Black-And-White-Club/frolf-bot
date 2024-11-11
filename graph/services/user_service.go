package services

import (
	"context"
	"fmt"
	"log"

	// Make sure to import Firestore
	"github.com/romero-jace/tcr-bot/graph/model" // Import the model package
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UserService struct with function fields for mocking
type UserService struct {
	client FirestoreClient // Change to use the interface

	// Function fields for mocking
	CreateUserFunc func(ctx context.Context, input model.UserInput) (*model.User, error)
	GetUserFunc    func(ctx context.Context, discordID string) (*model.User, error)
}

// NewUser Service creates a new UserService
func NewUserService(client FirestoreClient) *UserService {
	return &UserService{client: client}
}

// CreateUser  creates a new user in Firestore
func (us *UserService) CreateUser(ctx context.Context, input model.UserInput) (*model.User, error) {
	if us.CreateUserFunc != nil {
		return us.CreateUserFunc(ctx, input) // Call the mock function if set
	}

	log.Printf("Creating user with Discord ID: %s", input.DiscordID)

	// Validate input
	if input.DiscordID == "" || input.Name == "" {
		return nil, fmt.Errorf("DiscordID and Name are required")
	}

	// Check if the user already exists
	_, err := us.client.Collection("users").Doc(input.DiscordID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			// User does not exist, proceed to create a new user
		} else {
			// Some other error occurred
			return nil, fmt.Errorf("failed to check if user exists: %v", err)
		}
		// If we reach here, it means the user exists
		return nil, fmt.Errorf("user with Discord ID %s already exists", input.DiscordID)
	}

	// Create a new user
	newUser := &model.User{
		DiscordID: input.DiscordID,
		Name:      input.Name,
	}

	// Save the user to Firestore
	_, err = us.client.Collection("users").Doc(newUser.DiscordID).Set(ctx, newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return newUser, nil
}

// GetUser  retrieves a user by Discord ID
func (us *UserService) GetUser(ctx context.Context, discordID string) (*model.User, error) {
	if us.GetUserFunc != nil {
		return us.GetUserFunc(ctx, discordID) // Call the mock function if set
	}

	if discordID == "" {
		return nil, fmt.Errorf("DiscordID is required")
	}

	doc, err := us.client.Collection("users").Doc(discordID).Get(ctx)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	var user model.User
	if err := doc.DataTo(&user); err != nil {
		return nil, fmt.Errorf("failed to convert document data: %v", err)
	}

	return &user, nil
}
