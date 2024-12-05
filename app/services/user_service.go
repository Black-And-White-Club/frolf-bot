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
	// Validate input
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.DiscordID == "" || user.Name == "" || user.Role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Handle tag number logic with a switch statement
	switch {
	case tagNumber != 0: // Tag number provided
		// 1. Prepare the request data for the leaderboard module
		checkTagEvent := &nats.CheckTagAvailabilityEvent{
			TagNumber: tagNumber,
			ReplyTo:   "user.check-tag-availability-response",
		}

		// 2. Send the request to the leaderboard module
		responseData, err := s.natsConnectionPool.Request(ctx, "check-tag-availability", checkTagEvent, 5)
		if err != nil {
			return fmt.Errorf("failed to check tag availability: %w", err)
		}

		// 3. Unmarshal the response from the leaderboard module
		var tagAvailabilityResponse nats.TagAvailabilityResponse
		err = json.Unmarshal(responseData, &tagAvailabilityResponse)
		if err != nil {
			return fmt.Errorf("failed to unmarshal tag availability response: %w", err)
		}

		// 4. If the tag is not available, return an error
		if !tagAvailabilityResponse.IsAvailable {
			return fmt.Errorf("tag number %d is already taken", tagNumber)
		}

		// 5. If the tag is available, proceed with user creation and publish event
		if err := s.createUserAndPublishEvent(ctx, user, tagNumber); err != nil {
			return err
		}

		return nil

	default: // No tag number provided
		// Create the user without a tag
		if err := s.db.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		return nil
	}
}

// Helper function to create user and publish event
func (s *UserService) createUserAndPublishEvent(ctx context.Context, user *models.User, tagNumber int) error {
	if err := s.db.CreateUser(ctx, user); err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	userCreatedEvent := &nats.UserCreatedEvent{
		DiscordID: user.DiscordID,
		TagNumber: tagNumber,
	}
	if err := s.natsConnectionPool.Publish("user.created", userCreatedEvent); err != nil {
		return fmt.Errorf("failed to publish user.created event: %w", err)
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
