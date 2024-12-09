// user/commands/command_service.go
package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/Black-And-White-Club/tcr-bot/nats"
	"github.com/Black-And-White-Club/tcr-bot/user"
	"github.com/ThreeDotsLabs/watermill/message"
)

// UserCommandService implements the api.CommandService interface.
type UserCommandService struct {
	userDB             db.UserDB
	natsConnectionPool *nats.NatsConnectionPool
	publisher          message.Publisher
	eventHandler       user.UserEventHandler // Add the eventHandler field
}

// NewUserCommandService creates a new UserCommandService.
func NewUserCommandService(userDB db.UserDB, natsConnectionPool *nats.NatsConnectionPool, publisher message.Publisher, eventHandler user.UserEventHandler) api.CommandService {
	return &UserCommandService{
		userDB:             userDB,
		natsConnectionPool: natsConnectionPool,
		publisher:          publisher,
		eventHandler:       eventHandler,
	}
}

// CreateUser handles user creation logic.
func (s *UserCommandService) CreateUser(ctx context.Context, user *db.User, tagNumber int) error {
	// Validate input
	if user == nil {
		return errors.New("user cannot be nil")
	}

	if user.DiscordID == "" || user.Name == "" || user.Role == "" {
		return errors.New("user has invalid or missing fields")
	}

	// Call the event handler to handle user creation
	if err := s.eventHandler.HandleUserCreated(ctx, eventhandling.UserCreatedEvent{
		DiscordID: user.DiscordID,
		TagNumber: tagNumber,
	}); err != nil {
		return fmt.Errorf("failed to handle user created event: %w", err)
	}

	return nil
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
	err = s.userDB.UpdateUser(ctx, discordID, existingUser)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Call the event handler to handle user update
	if err := s.eventHandler.HandleUserUpdated(ctx, eventhandling.UserUpdatedEvent{
		DiscordID: discordID,
		Name:      updates.Name,
		Role:      updates.Role,
	}); err != nil {
		return fmt.Errorf("failed to handle user updated event: %w", err)
	}

	return nil
}
