package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/Black-And-White-Club/tcr-bot/app/models"
	"github.com/Black-And-White-Club/tcr-bot/internal/db"
	"github.com/Black-And-White-Club/tcr-bot/internal/nats"
)

// UserService handles user-related logic and database interactions.
type UserService struct {
	db                 db.UserDB
	natsConnectionPool *nats.NatsConnectionPool
}

// NewUserService creates a new UserService.
func NewUserService(db db.UserDB, natsConnectionPool *nats.NatsConnectionPool) *UserService {
	return &UserService{
		db:                 db,
		natsConnectionPool: natsConnectionPool,
	}
}

// GetUser retrieves a user by Discord ID.
func (s *UserService) GetUser(ctx context.Context, discordID string) (*models.User, error) {
	return s.db.GetUser(ctx, discordID)
}

func (s *UserService) CreateUser(ctx context.Context, user *models.User, tagNumber int) error {
	// Check dependencies
	if s.db == nil {
		log.Println("UserService.db is nil")
		return errors.New("database connection is not initialized")
	}

	// Validate input
	if user == nil {
		log.Println("CreateUser - user is nil")
		return errors.New("user cannot be nil")
	}

	if user.DiscordID == "" || user.Name == "" || user.Role == "" {
		log.Printf("CreateUser - invalid user data: %+v", user)
		return errors.New("user has invalid or missing fields")
	}

	// Debug log user object
	userData, _ := json.MarshalIndent(user, "", "  ")
	log.Printf("Saving user to database: %s", string(userData))

	// Save the user
	if err := s.db.CreateUser(ctx, user); err != nil {
		log.Printf("CreateUser - failed to save user: %v", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user.
func (s *UserService) UpdateUser(ctx context.Context, discordID string, updates *models.User) error {
	// Get the existing user from the database
	existingUser, err := s.db.GetUser(ctx, discordID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Update the fields of the existing user with the provided updates
	if updates.Name != "" {
		existingUser.Name = updates.Name
	}
	if updates.Role != "" {
		existingUser.Role = updates.Role
	}

	// Save the updated user to the database
	err = s.db.UpdateUser(ctx, discordID, existingUser) // Pass discordID to UpdateUser
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Publish UserUpdatedEvent to NATS
	err = s.natsConnectionPool.Publish("user.updated", &nats.UserUpdatedEvent{
		DiscordID: existingUser.DiscordID,
		Name:      existingUser.Name,
		Role:      existingUser.Role,
		// TagNumber is removed from UserUpdatedEvent
	})
	if err != nil {
		log.Printf("Failed to publish user.updated event: %v", err)
	}

	return nil
}

// GetUserRole retrieves the role of a user.
func (s *UserService) GetUserRole(ctx context.Context, discordID string) (models.UserRole, error) {
	user, err := s.db.GetUser(ctx, discordID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}
	return user.Role, nil
}
