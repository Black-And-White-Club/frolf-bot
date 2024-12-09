// user/commands/command_service.go
package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/nats"
)

// UserCommandService implements the api.CommandService interface.
type UserCommandService struct {
	userDB             db.UserDB
	natsConnectionPool *nats.NatsConnectionPool
}

// NewUserCommandService creates a new UserCommandService.
func NewUserCommandService(userDB db.UserDB, natsConnectionPool *nats.NatsConnectionPool) api.CommandService {
	return &UserCommandService{
		userDB:             userDB,
		natsConnectionPool: natsConnectionPool,
	}
}

// CreateUser handles user creation logic, including tag availability checks.
func (s *UserCommandService) CreateUser(ctx context.Context, user *db.User, tagNumber int) error {
	// Validate input
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.DiscordID == "" || user.Name == "" || user.Role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Handle tag number logic
	switch {
	case tagNumber != 0: // Tag number provided
		// 1. Prepare the request data for the leaderboard module
		checkTagEvent := &nats.CheckTagAvailabilityEvent{
			TagNumber: tagNumber,
			// No ReplyTo field needed here
		}

		// 2. Marshal the request data
		payload, err := json.Marshal(checkTagEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal request data for subject %s: %w", "check-tag-availability", err)
		}

		// 3. Publish the request
		err = s.natsConnectionPool.Publish("check-tag-availability", payload)
		if err != nil {
			return fmt.Errorf("failed to publish request: %w", err)
		}

		//  4. If the tag is available, create the user
		if err := s.userDB.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		// 5. Publish the "user.created" event
		userCreatedEvent := &nats.UserCreatedEvent{
			DiscordID: user.DiscordID,
			TagNumber: tagNumber,
		}
		if err := s.natsConnectionPool.Publish("user.created", userCreatedEvent); err != nil {
			return fmt.Errorf("failed to publish user.created event: %w", err)
		}

		return nil

	default: // No tag number provided
		// Create the user without a tag
		if err := s.userDB.CreateUser(ctx, user); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		return nil
	}
}

// UpdateUser updates an existing user.
func (s *UserCommandService) UpdateUser(ctx context.Context, discordID string, updates *db.User) error {
	// Get the existing user from the database
	existingUser, err := s.userDB.GetUser(ctx, discordID)
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
	err = s.userDB.UpdateUser(ctx, discordID, existingUser) // Pass discordID to UpdateUser
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Publish UserUpdatedEvent to NATS
	err = s.natsConnectionPool.Publish("user.updated", &nats.UserUpdatedEvent{
		DiscordID: existingUser.DiscordID,
		Name:      existingUser.Name,
		Role:      existingUser.Role,
	})
	if err != nil {
		return fmt.Errorf("failed to publish user.updated event: %w", err)
	}

	return nil
}
